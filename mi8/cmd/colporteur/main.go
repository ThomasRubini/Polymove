package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/thomasrubini/polymove/common/proto"
)

var (
	cities = []string{"Paris", "London", "Berlin", "Madrid", "Rome", "Amsterdam", "Lyon"}
	titles = []struct {
		template string
		tags     []string
	}{
		{"New tech hub opens in", []string{"innovation", "economy"}},
		{"University announces scholarship program for", []string{"culture", "education"}},
		{"Cultural festival brings excitement to", []string{"culture", "entertainment"}},
		{"Startup scene flourishing in", []string{"innovation", "economy"}},
		{"Research center launches in", []string{"innovation", "healthcare"}},
		{"New museum exhibition in", []string{"culture", "entertainment"}},
		{"City invests in sustainable transport for", []string{"innovation", "healthcare"}},
		{"International conference coming to", []string{"culture", "economy"}},
		{"Student exchange program expands to", []string{"culture", "education"}},
		{"Innovation district announced for", []string{"innovation", "economy"}},
		{"Healthcare crisis in", []string{"crisis", "healthcare"}},
		{"Crime wave hits", []string{"crime"}},
		{"Natural disaster strikes", []string{"disaster"}},
		{"Economic downturn in", []string{"crisis", "economy"}},
		{"New entertainment district opens in", []string{"entertainment", "economy"}},
	}
	contents = []string{
		"Students from around the world are excited about this new opportunity.",
		"This marks a significant milestone for the city's educational institutions.",
		"The initiative is expected to boost local economy and tourism.",
		"Experts believe this will attract more international students.",
		"Local authorities have expressed their full support for the project.",
		"The program aims to foster collaboration between universities.",
		"This is part of a broader strategy to internationalize education.",
		"Researchers are already planning groundbreaking studies as a result.",
	}
)

func main() {
	host := getEnv("MI8_HOST", "localhost")
	port := getEnv("MI8_PORT", "8082")
	addr := net.JoinHostPort(host, port)

	log.Printf("Connecting to MI8 at %s", addr)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to MI8: %v", err)
	}
	defer conn.Close()

	client := proto.NewMI8ServiceClient(conn)
	ctx := context.Background()

	log.Printf("Colporteur started")

	for i := 0; i < 10; i++ {
		news := generateRandomNews()

		resp, err := client.CreateNews(ctx, &proto.CreateNewsRequest{
			City:    news.City,
			Title:   news.Title,
			Content: news.Content,
			Tags:    news.Tags,
		})
		if err != nil {
			log.Printf("Failed to create news: %v", err)
		} else {
			log.Printf("Created news #%d: %s [%s]", resp.Id, resp.City, strings.Join(resp.Tags, ", "))
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

type News struct {
	City    string
	Title   string
	Content string
	Tags    []string
}

func generateRandomNews() News {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	city := cities[r.Intn(len(cities))]
	entry := titles[r.Intn(len(titles))]
	content := contents[r.Intn(len(contents))]

	title := fmt.Sprintf("%s %s", entry.template, city)

	return News{
		City:    city,
		Title:   title,
		Content: content,
		Tags:    entry.tags,
	}
}
