package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/thomasrubini/polymove/common/proto"
	"google.golang.org/grpc"
)

var rdb *redis.Client
var ctx = context.Background()

func main() {
	log.SetOutput(os.Stdout)

	initRedis()
	rmqConn, rmqChannel := initRabbitMQ()
	defer rmqChannel.Close()
	defer rmqConn.Close()

	go consumeNewsEvents(rmqChannel)

	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterMI8ServiceServer(s, &server{})

	log.Println("gRPC server starting on :8082")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// initRabbitMQ connects to RabbitMQ with retries and returns a ready channel.
func initRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	host := getEnv("RABBITMQ_HOST", "localhost")
	port := getEnv("RABBITMQ_PORT", "5672")
	addr := "amqp://guest:guest@" + host + ":" + port + "/"

	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(addr)
		if err == nil {
			ch, chErr := conn.Channel()
			if chErr != nil {
				conn.Close()
				log.Printf("Failed to open RabbitMQ channel, retrying... (%d/10)", i+1)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Println("Connected to RabbitMQ")
			return conn, ch
		}

		log.Printf("Failed to connect to RabbitMQ, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatal("Failed to connect to RabbitMQ after 10 attempts")
	return nil, nil
}

// consumeNewsEvents subscribes to mi8.news and stores each event in Redis.
func consumeNewsEvents(ch *amqp.Channel) {
	const topic = "mi8.news"

	queue, err := ch.QueueDeclare(
		topic,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return
	}

	err = ch.QueueBind(
		queue.Name,
		topic,
		"amq.topic",
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to bind queue: %v", err)
		return
	}

	deliveries, err := ch.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to register consumer: %v", err)
		return
	}

	log.Printf("Subscribed to RabbitMQ topic %s", topic)

	for msg := range deliveries {
		if err := processNewsEvent(ctx, msg.Body); err != nil {
			log.Printf("Failed to process news event: %v", err)
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

func initRedis() {
	host := getEnv("REDIS_HOST", "redis")
	rdb = redis.NewClient(&redis.Options{
		Addr: host + ":6379",
	})

	for i := 0; i < 10; i++ {
		_, err := rdb.Ping(ctx).Result()
		if err == nil {
			log.Println("Connected to Redis")
			return
		}
		log.Printf("Failed to ping Redis, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Failed to connect to Redis after 10 attempts")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
