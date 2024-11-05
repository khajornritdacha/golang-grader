package mem

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type PeakMemoryUsage struct {
	Snapshot      int
	Time          int
	MemHeapB      int
	MemHeapExtraB int
	MemStacksB    int
}

func ParseMemPeakUsage(log string, peakUsage *PeakMemoryUsage) error {
	if strings.HasPrefix(log, "time=") {
		peakUsage.Time, _ = strconv.Atoi(strings.Split(log, "=")[1])
	} else if strings.HasPrefix(log, "mem_heap_B=") {
		peakUsage.MemHeapB, _ = strconv.Atoi(strings.Split(log, "=")[1])
	} else if strings.HasPrefix(log, "mem_heap_extra_B=") {
		peakUsage.MemHeapExtraB, _ = strconv.Atoi(strings.Split(log, "=")[1])
	} else if strings.HasPrefix(log, "mem_stacks_B=") {
		peakUsage.MemStacksB, _ = strconv.Atoi(strings.Split(log, "=")[1])
	}
	return nil
}

func ExtractPeakMemoryUsage(filePath string) (*PeakMemoryUsage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var peakUsage PeakMemoryUsage
	var logs []string

	for scanner.Scan() {
		line := scanner.Text()
		logs = append(logs, line)

		// Detect the start of each snapshot
		if strings.HasPrefix(line, "snapshot=") {
			snapshotNumber, err := strconv.Atoi(strings.Split(line, "=")[1])
			if err == nil {
				peakUsage.Snapshot = snapshotNumber
			}
		}

		// Check if we are in the peak snapshot
		if strings.Contains(line, "heap_tree=peak") {
			// fmt.Println("Found peak snapshot")
			for i := 1; len(logs) >= i && i <= 5; i++ {
				// fmt.Println(logs[len(logs)-i])
				ParseMemPeakUsage(logs[len(logs)-i], &peakUsage)
			}

			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return &peakUsage, nil
}

func main() {
	filePath := "massif.out.296304" // Replace with your actual file path
	peakUsage, err := ExtractPeakMemoryUsage(filePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Peak Memory Usage Details:")
	fmt.Printf("Snapshot: %d\n", peakUsage.Snapshot)
	fmt.Printf("Time: %d bytes executed\n", peakUsage.Time)
	fmt.Printf("Heap Memory (mem_heap_B): %d bytes\n", peakUsage.MemHeapB)
	fmt.Printf("Extra Heap Memory (mem_heap_extra_B): %d bytes\n", peakUsage.MemHeapExtraB)
	fmt.Printf("Stack Memory (mem_stacks_B): %d bytes\n", peakUsage.MemStacksB)
}
