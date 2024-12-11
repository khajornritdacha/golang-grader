package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/proj/config"
	"example.com/proj/pkg/utils"
)

func readWorkerURLs() []string {
	env := os.Getenv("WORKER_URLS")
	if env == "" {
		log.Default().Println("WORKER_URLS is not set using default value")
		env = "['http://localhost:8081']"
	}

	log.Default().Printf("WORKER_URL: %s\n", env)

	workerURLs := strings.ReplaceAll(env, "'", "\"")

	var urls []string
	err := json.Unmarshal([]byte(workerURLs), &urls)
	if err != nil {
		log.Fatalf("Failed to parse WORKER_URLS: %v", err)
	}
	// Print the parsed URLs
	fmt.Printf("Parsed worker URLs: %+v\n", urls)

	return urls
}

func main() {
	CODE_TOPIC := "code"
	cfg := config.KafkaConnCfg{
		Url:   "localhost:9092",
		Topic: CODE_TOPIC,
	}
	utils.SetupKafkaTopic(cfg)
	conn := utils.KafkaConn(cfg)

	workers := readWorkerURLs()

	defer conn.Close()

	for {
		message, err := conn.ReadMessage(10e3)
		if err != nil {
			log.Fatalf("Failed to parse JSON: %v", err)
		}

		for _, workerURL := range workers {
			isUsing := isUsingWorker(workerURL)
			fmt.Println(isUsing)
			if !isUsing {
				go forwardRequest(message.Value, workerURL)
			}
		}

		time.Sleep(500 * time.Millisecond) // Small delay to reduce CPU usage
	}

}

func isUsingWorker(workerURL string) bool {
	_, err := http.Get(workerURL)
	if err != nil {
		return false
	} else {
		return true
	}
}

func forwardRequest(requestBody []byte, workerURL string) {
	bodyReader := bytes.NewReader(requestBody)
	req, _ := http.NewRequest(http.MethodPost, workerURL, bodyReader)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to forward request: %v", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("client: response body: %s\n", resBody)

	// TODO: save in db and let frontend fetch it
}
