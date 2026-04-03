package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thomasrubini/polymove/common"
)

type Subscriber struct {
	StudentID int    `json:"student_id"`
	Domain    string `json:"domain"`
	Channel   string `json:"channel"`
	Contact   string `json:"contact"`
	Enabled   bool   `json:"enabled"`
}

type StudentRegisteredEvent struct {
	StudentID int    `json:"student_id"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	CreatedAt string `json:"created_at"`
}

type SubscriberUpdateRequest struct {
	Domain  string `json:"domain"`
	Channel string `json:"channel"`
	Contact string `json:"contact"`
	Enabled bool   `json:"enabled"`
}

var (
	subscribers   = make(map[int]Subscriber)
	subscribersMu sync.RWMutex
)

// main boots the RabbitMQ consumer and starts La Poste REST endpoints.
func main() {
	log.SetOutput(os.Stdout)

	rmqConn, rmqChannel := initRabbitMQ()
	defer rmqChannel.Close()
	defer rmqConn.Close()

	go consumeStudentRegisteredEvents(rmqChannel)

	router := mux.NewRouter()
	router.HandleFunc("/subscribers/{studentId}", getSubscriber).Methods(http.MethodGet)
	router.HandleFunc("/subscribers/{studentId}", updateSubscriber).Methods(http.MethodPut)
	router.HandleFunc("/subscribrs/{studentId}", updateSubscriber).Methods(http.MethodPut)
	router.HandleFunc("/subscribers/{studentId}", deleteSubscriber).Methods(http.MethodDelete)

	log.Println("La Poste server starting on :8083")
	log.Fatal(http.ListenAndServe(":8083", router))
}

// initRabbitMQ connects La Poste to RabbitMQ and returns a ready channel.
func initRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	host := getEnv("RABBITMQ_HOST", "localhost")
	port := getEnv("RABBITMQ_PORT", "5672")
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

// consumeStudentRegisteredEvents subscribes to student.registered and creates defaults.
func consumeStudentRegisteredEvents(ch *amqp.Channel) {
	queueName := "laposte.student.registered"
	routingKey := common.RoutingKeyStudentRegistered

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
		if err := processStudentRegisteredEvent(msg.Body); err != nil {
			log.Printf("Failed to process student.registered event: %v", err)
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

// processStudentRegisteredEvent stores default subscriber preferences for a student.
func processStudentRegisteredEvent(payload []byte) error {
	var event StudentRegisteredEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if event.StudentID <= 0 {
		return fmt.Errorf("invalid student_id in event")
	}

	subscribersMu.Lock()
	defer subscribersMu.Unlock()

	subscriber, exists := subscribers[event.StudentID]
	if !exists {
		subscriber = Subscriber{
			StudentID: event.StudentID,
			Domain:    event.Domain,
			Channel:   "email",
			Contact:   "",
			Enabled:   true,
		}
	} else if event.Domain != "" {
		subscriber.Domain = event.Domain
	}

	subscribers[event.StudentID] = subscriber
	return nil
}

// getSubscriber returns subscriber preferences for a student.
func getSubscriber(w http.ResponseWriter, r *http.Request) {
	studentID, err := parseStudentID(mux.Vars(r)["studentId"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	subscribersMu.RLock()
	subscriber, exists := subscribers[studentID]
	subscribersMu.RUnlock()
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscriber not found"})
		return
	}

	writeJSON(w, http.StatusOK, subscriber)
}

// updateSubscriber updates notification preferences for a student.
func updateSubscriber(w http.ResponseWriter, r *http.Request) {
	studentID, err := parseStudentID(mux.Vars(r)["studentId"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var req SubscriberUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	subscribersMu.Lock()
	subscriber := subscribers[studentID]
	subscriber.StudentID = studentID
	if req.Domain != "" {
		subscriber.Domain = req.Domain
	}
	if req.Channel != "" {
		subscriber.Channel = req.Channel
	}
	subscriber.Contact = req.Contact
	subscriber.Enabled = req.Enabled
	subscribers[studentID] = subscriber
	subscribersMu.Unlock()

	writeJSON(w, http.StatusOK, subscriber)
}

// deleteSubscriber removes a student from notification preferences.
func deleteSubscriber(w http.ResponseWriter, r *http.Request) {
	studentID, err := parseStudentID(mux.Vars(r)["studentId"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	subscribersMu.Lock()
	delete(subscribers, studentID)
	subscribersMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// parseStudentID converts route params into a valid student ID.
func parseStudentID(raw string) (int, error) {
	studentID, err := strconv.Atoi(raw)
	if err != nil || studentID <= 0 {
		return 0, fmt.Errorf("invalid student id")
	}
	return studentID, nil
}

// writeJSON sends JSON responses with a status code.
func writeJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// getEnv returns env var values with fallback defaults.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
