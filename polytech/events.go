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
	rmqConn, rmqChannel = common.InitRabbitMQ(
		getEnv("RABBITMQ_HOST", "localhost"),
		getEnv("RABBITMQ_PORT", "5672"),
	)
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

// consumeOfferCreatedEvents subscribes to offer.created and stores notifications for matching students.
func consumeOfferCreatedEvents(ch *amqp.Channel) {
	queueName := "polytech.offer.created"
	routingKey := common.RoutingKeyOfferCreated

	queue, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return
	}

	err = ch.QueueBind(queue.Name, routingKey, "amq.topic", false, nil)
	if err != nil {
		log.Printf("Failed to bind queue: %v", err)
		return
	}

	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to register consumer: %v", err)
		return
	}

	log.Printf("Subscribed to routing key %s", routingKey)

	for msg := range deliveries {
		if err := processOfferCreatedEvent(msg.Body); err != nil {
			log.Printf("Failed to process offer.created event: %v", err)
			if nackErr := msg.Nack(false, true); nackErr != nil {
				log.Printf("Failed to nack message: %v", nackErr)
			}
			continue
		}

		if ackErr := msg.Ack(false); ackErr != nil {
			log.Printf("Failed to ack message: %v", ackErr)
		}
	}
}

// processOfferCreatedEvent creates one notification for each student matching the offer domain.
func processOfferCreatedEvent(payload []byte) error {
	var event common.OfferCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if event.OfferID <= 0 || event.Domain == "" {
		return fmt.Errorf("invalid offer.created event")
	}

	rows, err := db.Query("SELECT id FROM students WHERE domain = $1", event.Domain)
	if err != nil {
		return fmt.Errorf("failed to query matching students: %w", err)
	}
	defer func() { _ = rows.Close() }()

	message := fmt.Sprintf("New offer '%s' in %s matches your domain %s.", event.Title, event.City, event.Domain)
	for rows.Next() {
		var studentID int
		if err := rows.Scan(&studentID); err != nil {
			return fmt.Errorf("failed to scan student: %w", err)
		}

		_, err := db.Exec(
			"INSERT INTO notifications (student_id, type, offer_id, message, read) VALUES ($1, $2, $3, $4, false) ON CONFLICT (student_id, offer_id, type) DO NOTHING",
			studentID,
			"new_offer",
			event.OfferID,
			message,
		)
		if err != nil {
			return fmt.Errorf("failed to insert notification: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed iterating students: %w", err)
	}

	return nil
}
