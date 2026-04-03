package main

import (
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thomasrubini/polymove/common"
)

var rmqConn *amqp.Connection
var rmqChannel *amqp.Channel

// initRabbitMQ initializes a RabbitMQ connection and channel for Erasmumu publishers.
func initRabbitMQ() {
	rmqConn, rmqChannel = common.InitRabbitMQ(
		getEnv("RABBITMQ_HOST", "localhost"),
		getEnv("RABBITMQ_PORT", "5672"),
	)
}

// publishOfferCreatedEvent publishes an offer.created event for a newly inserted offer.
func publishOfferCreatedEvent(offer common.Offer) error {
	event := common.OfferCreatedEvent{
		OfferID:   offer.ID,
		Title:     offer.Title,
		Domain:    offer.Domain,
		City:      offer.City,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal offer event: %w", err)
	}

	err = rmqChannel.Publish(
		"amq.topic",
		common.RoutingKeyOfferCreated,
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
