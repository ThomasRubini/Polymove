package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type Offer struct {
	ID          int    `json:"id"`
	University  string `json:"university"`
	City        string `json:"city"`
	Country     string `json:"country"`
	Domain      string `json:"domain"`
	Description string `json:"description"`
}

type CityScore struct {
	City    string  `json:"city"`
	Safety  float64 `json:"safety"`
	Economy float64 `json:"economy"`
	QoL     float64 `json:"qol"`
	Culture float64 `json:"culture"`
}

type Internship struct {
	ID        int        `json:"id"`
	StudentID int        `json:"student_id"`
	OfferID   int        `json:"offer_id"`
	Offer     *Offer     `json:"offer,omitempty"`
	CityScore *CityScore `json:"city_score,omitempty"`
}

type InternshipRequest struct {
	StudentID int `json:"student_id"`
	OfferID   int `json:"offer_id"`
}

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

func createInternship(w http.ResponseWriter, r *http.Request) error {
	var req InternshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	var student Student
	query := "SELECT id, name, domain FROM students WHERE id = $1"
	err := db.QueryRow(query, req.StudentID).Scan(&student.ID, &student.Name, &student.Domain)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("student with id %d not found", req.StudentID)
		}
		return fmt.Errorf("failed to get student: %w", err)
	}

	erasmumuURL := getEnv("ERASMUMU_URL", "http://erasmumu:8081")
	resp, err := http.Get(fmt.Sprintf("%s/offers/%d", erasmumuURL, req.OfferID))
	if err != nil {
		return fmt.Errorf("failed to fetch offer from erasmumu: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("offer with id %d not found", req.OfferID)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erasmumu returned status %d", resp.StatusCode)
	}

	var offer Offer
	if err := json.NewDecoder(resp.Body).Decode(&offer); err != nil {
		return fmt.Errorf("failed to decode offer response: %w", err)
	}

	if offer.Domain != student.Domain {
		return fmt.Errorf("student domain '%s' does not match offer domain '%s'", student.Domain, offer.Domain)
	}

	var internship Internship
	query = "INSERT INTO internships (student_id, offer_id) VALUES ($1, $2) RETURNING id"
	if err := db.QueryRow(query, req.StudentID, req.OfferID).Scan(&internship.ID); err != nil {
		return fmt.Errorf("failed to insert internship: %w", err)
	}

	internship.StudentID = req.StudentID
	internship.OfferID = req.OfferID
	internship.Offer = &offer

	mi8URL := getEnv("MI8_URL", "http://mi8:8082")
	mi8Resp, err := http.Get(fmt.Sprintf("%s/scores?city=%s", mi8URL, offer.City))
	if err == nil && mi8Resp.StatusCode == http.StatusOK {
		var scores []CityScore
		if err := json.NewDecoder(mi8Resp.Body).Decode(&scores); err == nil && len(scores) > 0 {
			internship.CityScore = &scores[0]
		}
		mi8Resp.Body.Close()
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, internship)
}
