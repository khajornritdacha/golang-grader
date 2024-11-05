package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type RequestBody struct {
	CppCode   string   `json:"cpp_code"`
	TestCases []string `json:"test_cases"`
}

type ResponseBody struct {
	Results []TestResult `json:"results"`
}

type TestResult struct {
	Result string `json:"result"`
	Time   string `json:"time"`
}

var semaphore = make(chan struct{}, 10) // Limit to 3 concurrent requests
var ch = make(chan string)

var cpuQuota = -1
var cpuPeriod = 100000
var groupName = "test"
var cgroupPath = filepath.Join("/sys/fs/cgroup/cpu", groupName)

func main() {
	if err := createCgroup(cgroupPath, cpuQuota, cpuPeriod); err != nil {
		fmt.Println("Error creating cgroup test:", err)
		return
	}
	defer removeCgroup(cgroupPath)

	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/submit", handleMultipleCodes)

	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func createCgroup(cgroupPath string, cpuQuota int, cpuPeriod int) error {

	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		fmt.Println(err)
		return fmt.Errorf("failed to create cgroup: %v", err)
	}

	cpuPeriodPath := filepath.Join(cgroupPath, "cpu.cfs_period_us")
	if err := os.WriteFile(cpuPeriodPath, []byte(fmt.Sprintf("%d", cpuPeriod)), 0644); err != nil {
		return fmt.Errorf("failed to set cpu period: %w", err)
	}

	cpu_quota_filepath := filepath.Join(cgroupPath, "cpu.cfs_quota_us")
	if err := os.WriteFile(cpu_quota_filepath, []byte(fmt.Sprintf("%d", cpuQuota)), 0644); err != nil {
		return fmt.Errorf("failed to set cpu quota: %w", err)
	}

	fmt.Printf("Cgroup %s created with CPU quota %d/%d microseconds\n", cgroupPath, cpuQuota, cpuPeriod)
	return nil
}

func removeCgroup(cgroupPath string) error {
	if err := os.RemoveAll(cgroupPath); err != nil {
		return fmt.Errorf("failed to remove cgroup: %w", err)
	}
	fmt.Printf("Cgroup %s removed\n", cgroupPath)
	return nil
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func handleMultipleCodes(w http.ResponseWriter, r *http.Request) {
	// Acquire a spot in the semaphore before proceeding
	semaphore <- struct{}{}
	defer func() {
		<-semaphore // Release the spot when the request is done
	}()

	var requestBody RequestBody
	json.NewDecoder(r.Body).Decode(&requestBody)

	fmt.Printf("Received request with %d test cases\n", len(requestBody.TestCases))

	tempDir, err := os.MkdirTemp("", "cpp-")
	if err != nil {
		http.Error(w, "Error creating temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)
	fmt.Println("Temporary dir name: ", tempDir)
	fmt.Printf("Hello World\n")

	codeFileName := fmt.Sprintf("%s/code.cpp", tempDir)
	binaryFileName := fmt.Sprintf("%s/output_binary", tempDir)

	err = os.WriteFile(codeFileName, []byte(requestBody.CppCode), 0644)
	if err != nil {
		http.Error(w, "Error writing C++ code", http.StatusInternalServerError)
		return
	}

	compileCmd := exec.Command("g++", "-o", binaryFileName, codeFileName)
	compileErr := compileCmd.Run()
	if compileErr != nil {
		http.Error(w, "Compilation error", http.StatusInternalServerError)
		return
	}

	var results []TestResult
	for _, testCase := range requestBody.TestCases {
		result, err, duration := runBinaryWithInput(binaryFileName, testCase)

		fmt.Println(duration)

		if err != nil {
			results = append(results, TestResult{
				Result: "Runtime error: " + err.Error(),
				Time:   "0",
			})
		} else {
			results = append(results, TestResult{
				Result: result,
				Time:   duration.String(),
			})
		}
	}

	fmt.Sprintln("Results: ", results)

	w.Header().Set("Content-Type", "application/json")
	resBody := ResponseBody{Results: results}
	json.NewEncoder(w).Encode(resBody)
}

func addProcessToCgroup(cgroupPath string, pid int) error {
	tasksPath := filepath.Join(cgroupPath, "tasks")
	res := os.WriteFile(tasksPath, []byte(fmt.Sprintf("%d", pid)), 0644)
	data, _ := os.ReadFile(tasksPath)
	fmt.Println("Tasks Path: ", string(data))
	return res
}

func runBinaryWithInput(binaryFile, input string) (string, error, time.Duration) {
	testCmd := exec.Command(binaryFile)

	stdin, err := testCmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	stdout, err := testCmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()

	if err := testCmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start the binary: %w", err), 0
	}
	// testCmd.Stdin = strings.NewReader(input)
	fmt.Printf("Pid: %d\n", testCmd.Process.Pid)
	if err := addProcessToCgroup(cgroupPath, testCmd.Process.Pid); err != nil {
		testCmd.Process.Kill()
		return "", fmt.Errorf("failed to add process to cgroup: %w", err), 0
	}

	// Read from stdout and parse the integer result
	scanner := bufio.NewScanner(stdout)
	var result int
	if scanner.Scan() {
		// Parse the integer from the first line of output
		result, err = strconv.Atoi(scanner.Text())
		if err != nil {
			return string(0), fmt.Errorf("failed to parse output as integer: %w", err), 0
		}
	}

	// compileCmd := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(testCmd.Process.Pid))
	compileCmd := exec.Command("ps", "-o", "rss=", "-p", "1")
	output, err := compileCmd.Output()
	outputStr := strings.TrimSpace(string(output))
	memoryUsageKB, err := strconv.Atoi(outputStr)
	// compileErr := compileCmd.Run()
	println("Memory used: ", memoryUsageKB)

	if err := testCmd.Wait(); err != nil {
		return "", fmt.Errorf("binary execution failed: %w", err), 0
	}

	duration := time.Since(start)

	fmt.Println("Time take:", duration)
	fmt.Println("return result: ", result)
	resultStr := strconv.Itoa(result)

	return resultStr, nil, duration
}
