package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"example.com/proj/config"
	"example.com/proj/pkg/utils"
	"github.com/segmentio/kafka-go"
)

var conn *kafka.Conn

type RequestBody struct {
	CppCode   string   `json:"cpp_code"`
	TestCases []string `json:"test_cases"`
}

func main() {
	CODE_TOPIC := "code"
	cfg := config.KafkaConnCfg{
		Url:   "localhost:9092",
		Topic: CODE_TOPIC,
	}
	utils.SetupKafkaTopic(cfg)
	conn = utils.KafkaConn(cfg)

	defer conn.Close()

	http.HandleFunc("/submit", handleSubmit)

	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	fmt.Println(requestBody)

	// Convert to kafka.Message{}
	messages := make([]kafka.Message, 0)
	messages = append(messages, kafka.Message{
		Value: utils.CompressToJsonBytes(requestBody),
	})

	log.Default().Println("Writing messages to Kafka")

	_, err = conn.WriteMessages(messages...)

	if err != nil {
		log.Fatal("failed to write messages: ", err)
	}

	log.Default().Println("Submitted")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Submitted"))
}
