package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func createStudent(w http.ResponseWriter, r *http.Request) error {
	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	query := "INSERT INTO students (name, domain) VALUES ($1, $2) RETURNING id"
	if err := db.QueryRow(query, student.Name, student.Domain).Scan(&student.ID); err != nil {
		return fmt.Errorf("failed to insert student: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, student)
}

func getStudent(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id := vars["id"]

	var student Student
	query := "SELECT id, name, domain FROM students WHERE id = $1"
	err := db.QueryRow(query, id).Scan(&student.ID, &student.Name, &student.Domain)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("student with id %s not found", id)
		}
		return fmt.Errorf("failed to get student: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, student)
}

func getStudentsByDomain(w http.ResponseWriter, r *http.Request) error {
	domain := r.URL.Query().Get("domain")

	var query string
	var args []interface{}

	if domain != "" {
		query = "SELECT id, name, domain FROM students WHERE domain = $1"
		args = append(args, domain)
	} else {
		query = "SELECT id, name, domain FROM students"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query students: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var student Student
		if err := rows.Scan(&student.ID, &student.Name, &student.Domain); err != nil {
			return fmt.Errorf("failed to scan student: %w", err)
		}
		students = append(students, student)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, students)
}

func updateStudent(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id := vars["id"]

	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	query := "UPDATE students SET name = $1, domain = $2 WHERE id = $3"
	result, err := db.Exec(query, student.Name, student.Domain, id)
	if err != nil {
		return fmt.Errorf("failed to update student: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("student with id %s not found", id)
	}

	student.ID, _ = fmt.Sscanf(id, "%d", &student.ID)
	return NewResponseWriter(w).JSON(http.StatusOK, student)
}

func deleteStudent(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id := vars["id"]

	query := "DELETE FROM students WHERE id = $1"
	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("student with id %s not found", id)
	}

	NewResponseWriter(w).NoContent()
	return nil
}
