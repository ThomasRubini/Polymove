package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Student struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var db *sql.DB

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
				log.Printf("Failed to send error to user: %v", err)
			}
		}
	}
}

func main() {
	log.SetOutput(os.Stdout)

	initDB()

	router := mux.NewRouter()
	router.HandleFunc("/student", errorHandler(createStudent)).Methods(http.MethodPost)
	router.HandleFunc("/student/{id}", errorHandler(getStudent)).Methods(http.MethodGet)
	router.HandleFunc("/student", errorHandler(getStudentsByDomain)).Methods(http.MethodGet)
	router.HandleFunc("/student/{id}", errorHandler(updateStudent)).Methods(http.MethodPut)
	router.HandleFunc("/student/{id}", errorHandler(deleteStudent)).Methods(http.MethodDelete)

	router.HandleFunc("/internship", errorHandler(createInternship)).Methods(http.MethodPost)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initDB() {
	host := getEnv("DB_HOST", "db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "school")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("Failed to ping DB, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal(err)
	}

	createTable()
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func createTable() {
	studentsQuery := `
	CREATE TABLE IF NOT EXISTS students (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		domain VARCHAR(255) NOT NULL
	);
	`
	_, err := db.Exec(studentsQuery)
	if err != nil {
		log.Fatal(err)
	}

	offersQuery := `
	CREATE TABLE IF NOT EXISTS offers (
		id SERIAL PRIMARY KEY,
		university VARCHAR(255) NOT NULL,
		city VARCHAR(255) NOT NULL,
		country VARCHAR(255) NOT NULL,
		description TEXT
	);
	`
	_, err = db.Exec(offersQuery)
	if err != nil {
		log.Fatal(err)
	}

	internshipsQuery := `
	CREATE TABLE IF NOT EXISTS internships (
		id SERIAL PRIMARY KEY,
		student_id INTEGER NOT NULL REFERENCES students(id),
		offer_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(internshipsQuery)
	if err != nil {
		log.Fatal(err)
	}
}
