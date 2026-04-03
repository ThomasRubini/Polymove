package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thomasrubini/polymove/common"
)

var rmqConn *amqp.Connection
var rmqChannel *amqp.Channel

type StudentRegisteredEvent struct {
	StudentID int    `json:"student_id"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	CreatedAt string `json:"created_at"`
}

// initRabbitMQ initializes a RabbitMQ connection and channel for Polytech publishers.
func initRabbitMQ() {
	host := getEnv("RABBITMQ_HOST", "localhost")
	port := getEnv("RABBITMQ_PORT", "5672")
	addr := fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port)

	var err error
	for i := 0; i < 10; i++ {
		rmqConn, err = amqp.Dial(addr)
		if err == nil {
			rmqChannel, err = rmqConn.Channel()
			if err == nil {
				log.Println("Connected to RabbitMQ")
				return
			}
			rmqConn.Close()
		}

		log.Printf("Failed to connect to RabbitMQ, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("Failed to connect to RabbitMQ after 10 attempts: %v", err)
}

// publishStudentRegisteredEvent emits the student.registered event for new students.
func publishStudentRegisteredEvent(student Student) error {
	event := StudentRegisteredEvent{
		StudentID: student.ID,
		Name:      student.Name,
		Domain:    student.Domain,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal student event: %w", err)
	}

	err = rmqChannel.Publish(
		"amq.topic",
		common.RoutingKeyStudentRegistered,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
