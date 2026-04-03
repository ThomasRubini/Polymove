package common

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// InitRabbitMQ opens a RabbitMQ connection/channel with retries and exits on failure.
func InitRabbitMQ(host, port string) (*amqp.Connection, *amqp.Channel) {
	addr := fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port)

	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(addr)
		if err == nil {
			ch, chErr := conn.Channel()
			if chErr == nil {
				log.Println("Connected to RabbitMQ")
				return conn, ch
			}
			conn.Close()
			err = chErr
		}

		log.Printf("Failed to connect to RabbitMQ, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("Failed to connect to RabbitMQ after 10 attempts: %v", err)
	return nil, nil
}
