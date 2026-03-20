package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thomasrubini/polymove/common/proto"
	"google.golang.org/grpc"
)

var rdb *redis.Client
var ctx = context.Background()

func main() {
	log.SetOutput(os.Stdout)

	initRedis()

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
