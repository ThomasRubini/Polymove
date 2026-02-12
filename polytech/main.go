package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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

const (
	host     = "db"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "school"
)

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

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initDB() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTable()
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS students (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		domain VARCHAR(255) NOT NULL
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}
