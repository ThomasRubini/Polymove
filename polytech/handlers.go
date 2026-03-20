package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// Offer represents an Erasmus offer from Erasmumu service
type Offer struct {
	ID          int    `json:"id"`
	University  string `json:"university"`
	City        string `json:"city"`
	Country     string `json:"country"`
	Domain      string `json:"domain"`
	Description string `json:"description"`
}

// CityScore represents city metrics from MI8 service
type CityScore struct {
	Safety    float64 `json:"safety"`
	Economy   float64 `json:"economy"`
	QoL       float64 `json:"qol"`
	Culture   float64 `json:"culture"`
	Relevance float64 `json:"relevance"`
}

// Internship represents a student's internship placement
type Internship struct {
	ID        int        `json:"id"`
	StudentID int        `json:"student_id"`
	OfferID   int        `json:"offer_id"`
	Offer     *Offer     `json:"offer,omitempty"`
	CityScore *CityScore `json:"city_score,omitempty"`
}

// InternshipRequest is the payload for creating an internship
type InternshipRequest struct {
	StudentID int `json:"student_id"`
	OfferID   int `json:"offer_id"`
}

// createStudent handles POST /student - Creates a new student
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

// getStudent handles GET /student/{id} - Retrieves a student by ID
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

// getStudentsByDomain handles GET /student - Lists students, optionally filtered by domain
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

// updateStudent handles PUT /student/{id} - Updates an existing student
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

// deleteStudent handles DELETE /student/{id} - Deletes a student
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

// createInternship handles POST /internship - Creates an internship for a student
// Validates student exists, checks domain match with offer, fetches offer from Erasmumu
// and city scores from MI8
func createInternship(w http.ResponseWriter, r *http.Request) error {
	var req InternshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	// Validate student exists
	var student Student
	query := "SELECT id, name, domain FROM students WHERE id = $1"
	err := db.QueryRow(query, req.StudentID).Scan(&student.ID, &student.Name, &student.Domain)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("student with id %d not found", req.StudentID)
		}
		return fmt.Errorf("failed to get student: %w", err)
	}

	// Fetch offer from Erasmumu
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

	// Check domain match between student and offer
	if offer.Domain != student.Domain {
		return fmt.Errorf("student domain '%s' does not match offer domain '%s'", student.Domain, offer.Domain)
	}

	// Insert internship into database
	var internship Internship
	query = "INSERT INTO internships (student_id, offer_id) VALUES ($1, $2) RETURNING id"
	if err := db.QueryRow(query, req.StudentID, req.OfferID).Scan(&internship.ID); err != nil {
		return fmt.Errorf("failed to insert internship: %w", err)
	}

	internship.StudentID = req.StudentID
	internship.OfferID = req.OfferID
	internship.Offer = &offer

	// Fetch city scores from MI8 via gRPC
	cityScore, err := getCityScoresFromMI8(r.Context(), offer.City)
	if err == nil && cityScore != nil {
		internship.CityScore = cityScore
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, internship)
}

// OfferWithScore represents an offer with its associated city score
type OfferWithScore struct {
	Offer
	Scores     *CityScore  `json:"scores,omitempty"`
	LatestNews []NewsTitle `json:"latest_news,omitempty"`
}

// NewsTitle represents just the title of a news article
type NewsTitle struct {
	Title string `json:"title"`
}

// News represents a news article from MI8
type News struct {
	ID        int      `json:"id"`
	City      string   `json:"city"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	CreatedAt string   `json:"created_at"`
	Tags      []string `json:"tags"`
}

// getOffersGateway handles GET /offers - Gateway endpoint to fetch offers from Erasmumu with city scores
func getOffersGateway(w http.ResponseWriter, r *http.Request) error {
	erasmumuURL := getEnv("ERASMUMU_URL", "http://erasmumu:8081")
	resp, err := http.Get(erasmumuURL + "/offers")
	if err != nil {
		return fmt.Errorf("failed to fetch offers from erasmumu: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erasmumu returned status %d", resp.StatusCode)
	}

	var offers []Offer
	if err := json.NewDecoder(resp.Body).Decode(&offers); err != nil {
		return fmt.Errorf("failed to decode offers response: %w", err)
	}

	offersWithScores := make([]OfferWithScore, 0, len(offers))
	for _, offer := range offers {
		offerWithScore := OfferWithScore{Offer: offer}
		cityScore, err := getCityScoresFromMI8(r.Context(), offer.City)
		if err == nil && cityScore != nil {
			offerWithScore.Scores = cityScore
		}
		news, err := getNewsFromMI8(r.Context(), offer.City)
		if err == nil {
			titles := make([]NewsTitle, 0, len(news))
			for _, n := range news {
				titles = append(titles, NewsTitle{Title: n.Title})
			}
			offerWithScore.LatestNews = titles
		}
		offersWithScores = append(offersWithScores, offerWithScore)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, offersWithScores)
}

// getCityScoresGateway handles GET /city-scores - Gateway endpoint to fetch city scores from MI8 via gRPC
func getCityScoresGateway(w http.ResponseWriter, r *http.Request) error {
	city := r.URL.Query().Get("city")
	if city == "" {
		return fmt.Errorf("city query parameter is required")
	}

	cityScore, err := getCityScoresFromMI8(r.Context(), city)
	if err != nil {
		return fmt.Errorf("failed to fetch scores from mi8: %w", err)
	}

	var scores []CityScore
	if cityScore != nil {
		scores = append(scores, *cityScore)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, scores)
}
