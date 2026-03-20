package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

type CityScore struct {
	City      string  `json:"city"`
	Safety    float64 `json:"safety"`
	Economy   float64 `json:"economy"`
	QoL       float64 `json:"qol"`
	Culture   float64 `json:"culture"`
	Relevance float64 `json:"relevance"`
}

type News struct {
	ID        int       `json:"id"`
	City      string    `json:"city"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var rdb *redis.Client
var ctx = context.Background()

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{w, http.StatusOK}
}

func (rw *ResponseWriter) EncodeJSON(data interface{}) error {
	rw.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(rw).Encode(data)
}

func (rw *ResponseWriter) EncodeError(statusCode int, err error) error {
	rw.WriteHeader(statusCode)
	return json.NewEncoder(rw).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: err.Error(),
	})
}

func (rw *ResponseWriter) NoContent() {
	rw.WriteHeader(http.StatusNoContent)
}

func (rw *ResponseWriter) JSON(statusCode int, data interface{}) error {
	rw.WriteHeader(statusCode)
	return rw.EncodeJSON(data)
}

func errorHandler(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := NewResponseWriter(w)
		if err := fn(w, r); err != nil {
			log.Printf("Error: %v", err)
			if err2 := rw.EncodeError(http.StatusInternalServerError, err); err2 != nil {
				log.Printf("Failed to send error to user: %v", err2)
			}
		}
	}
}

func main() {
	log.SetOutput(os.Stdout)

	initRedis()

	router := mux.NewRouter()
	router.HandleFunc("/scores", errorHandler(getScores)).Methods(http.MethodGet)
	router.HandleFunc("/scores", errorHandler(createScore)).Methods(http.MethodPost)
	router.HandleFunc("/news", errorHandler(getNews)).Methods(http.MethodGet)
	router.HandleFunc("/news", errorHandler(createNews)).Methods(http.MethodPost)

	log.Println("Server starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", router))
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
