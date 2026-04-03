package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"

	"github.com/gorilla/mux"

	"github.com/thomasrubini/polymove/common"
)

// Internship represents a student's internship placement
type Internship struct {
	ID        int               `json:"id"`
	StudentID int               `json:"student_id"`
	OfferID   int               `json:"offer_id"`
	Offer     *common.Offer     `json:"offer,omitempty"`
	CityScore *common.CityScore `json:"city_score,omitempty"`
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

	if err := publishStudentRegisteredEvent(student); err != nil {
		return fmt.Errorf("failed to publish student.registered event: %w", err)
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
	defer func() { _ = rows.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("offer with id %d not found", req.OfferID)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erasmumu returned status %d", resp.StatusCode)
	}

	var offer common.Offer
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
	common.Offer
	Scores     *common.CityScore `json:"scores,omitempty"`
	LatestNews []NewsTitle       `json:"latest_news,omitempty"`
}

// NewsTitle represents just the title of a news article
type NewsTitle struct {
	Title string `json:"title"`
}

type cityIntelligence struct {
	Scores     *common.CityScore
	LatestNews []NewsTitle
}

// fetchCityIntelligence loads MI8 data once per unique city using bounded parallel calls.
func fetchCityIntelligence(ctx context.Context, offers []common.Offer) map[string]cityIntelligence {
	uniqueCities := make(map[string]struct{})
	for _, offer := range offers {
		if offer.City == "" {
			continue
		}
		uniqueCities[offer.City] = struct{}{}
	}

	cityData := make(map[string]cityIntelligence, len(uniqueCities))
	if len(uniqueCities) == 0 {
		return cityData
	}

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, 5)
	)

	for city := range uniqueCities {
		city := city
		wg.Add(1)
		go func() {
			defer wg.Done()

			sem <- struct{}{}
			score, scoreErr := getCityScoresFromMI8(ctx, city)
			<-sem
			if scoreErr != nil {
				log.Printf("mi8 city scores unavailable for city=%s: %v", city, scoreErr)
			}

			sem <- struct{}{}
			news, newsErr := getNewsFromMI8(ctx, city)
			<-sem
			if newsErr != nil {
				log.Printf("mi8 news unavailable for city=%s: %v", city, newsErr)
			}

			intel := cityIntelligence{Scores: score}
			if newsErr == nil {
				titles := make([]NewsTitle, 0, len(news))
				for _, n := range news {
					titles = append(titles, NewsTitle{Title: n.Title})
				}
				intel.LatestNews = titles
			}

			mu.Lock()
			cityData[city] = intel
			mu.Unlock()
		}()
	}

	wg.Wait()
	return cityData
}

// buildOffersURL forwards supported filters to Erasmumu.
func buildOffersURL(baseURL, city string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid erasmumu url: %w", err)
	}

	parsedURL.Path = "/offers"
	query := parsedURL.Query()
	if city != "" {
		query.Set("city", city)
	}
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// getOffersGateway handles GET /offers - Gateway endpoint to fetch offers from Erasmumu with city scores
func getOffersGateway(w http.ResponseWriter, r *http.Request) error {
	limit := 10
	if limitValue := r.URL.Query().Get("limit"); limitValue != "" {
		if parsedLimit, err := strconv.Atoi(limitValue); err == nil {
			limit = parsedLimit
		}
	}
	city := r.URL.Query().Get("city")
	domain := r.URL.Query().Get("domain")

	erasmumuURL := getEnv("ERASMUMU_URL", "http://erasmumu:8081")
	offersURL, err := buildOffersURL(erasmumuURL, city)
	if err != nil {
		return err
	}

	resp, err := http.Get(offersURL)
	if err != nil {
		log.Printf("erasmumu unavailable for /offers: %v", err)
		return NewResponseWriter(w).JSON(http.StatusOK, []*OfferWithScore{})
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Printf("erasmumu returned non-OK status for /offers: %d", resp.StatusCode)
		return NewResponseWriter(w).JSON(http.StatusOK, []*OfferWithScore{})
	}

	var offers []common.Offer
	if err := json.NewDecoder(resp.Body).Decode(&offers); err != nil {
		return fmt.Errorf("failed to decode offers response: %w", err)
	}

	filteredOffers := make([]common.Offer, 0, len(offers))
	for _, offer := range offers {
		if domain != "" && offer.Domain != domain {
			continue
		}
		filteredOffers = append(filteredOffers, offer)
		if len(filteredOffers) == limit {
			break
		}
	}

	cityData := fetchCityIntelligence(r.Context(), filteredOffers)
	offersWithScores := make([]*OfferWithScore, 0, len(filteredOffers))
	for _, offer := range filteredOffers {
		offerWithScore := &OfferWithScore{Offer: offer}
		if intel, exists := cityData[offer.City]; exists {
			offerWithScore.Scores = intel.Scores
			offerWithScore.LatestNews = intel.LatestNews
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

	var scores []common.CityScore
	if cityScore != nil {
		scores = append(scores, *cityScore)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, scores)
}

// getSortScore extracts the selected sortable score from an offer.
func getSortScore(offer *OfferWithScore, sortBy string) float64 {
	if offer == nil || offer.Scores == nil {
		return 0
	}

	switch sortBy {
	case "safety":
		return offer.Scores.Safety
	case "economy":
		return offer.Scores.Economy
	case "qol":
		fallthrough
	case "quality_of_life":
		return offer.Scores.QoL
	case "culture":
		return offer.Scores.Culture
	default:
		return 0
	}
}

// getRecommendedOffers handles GET /students/{id}/recommended-offers.
func getRecommendedOffers(w http.ResponseWriter, r *http.Request) error {
	studentID := mux.Vars(r)["id"]

	var student Student
	query := "SELECT id, name, domain FROM students WHERE id = $1"
	err := db.QueryRow(query, studentID).Scan(&student.ID, &student.Name, &student.Domain)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("student with id %s not found", studentID)
		}
		return fmt.Errorf("failed to get student: %w", err)
	}

	limit := 5
	if limitValue := r.URL.Query().Get("limit"); limitValue != "" {
		if parsedLimit, parseErr := strconv.Atoi(limitValue); parseErr == nil {
			limit = parsedLimit
		}
	}
	sortBy := r.URL.Query().Get("sort_by")

	erasmumuURL := getEnv("ERASMUMU_URL", "http://erasmumu:8081")
	resp, err := http.Get(erasmumuURL + "/offers")
	if err != nil {
		log.Printf("erasmumu unavailable for /students/%s/recommended-offers: %v", studentID, err)
		return NewResponseWriter(w).JSON(http.StatusOK, []*OfferWithScore{})
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Printf("erasmumu returned non-OK status for /students/%s/recommended-offers: %d", studentID, resp.StatusCode)
		return NewResponseWriter(w).JSON(http.StatusOK, []*OfferWithScore{})
	}

	var offers []common.Offer
	if err := json.NewDecoder(resp.Body).Decode(&offers); err != nil {
		return fmt.Errorf("failed to decode offers response: %w", err)
	}

	matchingOffers := make([]common.Offer, 0, len(offers))
	for _, offer := range offers {
		if offer.Domain == student.Domain {
			matchingOffers = append(matchingOffers, offer)
		}
	}

	cityData := fetchCityIntelligence(r.Context(), matchingOffers)
	recommendedOffers := make([]*OfferWithScore, 0, len(matchingOffers))
	for _, offer := range matchingOffers {
		recommendedOffer := &OfferWithScore{Offer: offer}
		if intel, exists := cityData[offer.City]; exists {
			recommendedOffer.Scores = intel.Scores
			recommendedOffer.LatestNews = intel.LatestNews
		}
		recommendedOffers = append(recommendedOffers, recommendedOffer)
	}

	sort.Slice(recommendedOffers, func(i, j int) bool {
		return getSortScore(recommendedOffers[i], sortBy) > getSortScore(recommendedOffers[j], sortBy)
	})

	if limit >= 0 && len(recommendedOffers) > limit {
		recommendedOffers = recommendedOffers[:limit]
	}

	return NewResponseWriter(w).JSON(http.StatusOK, recommendedOffers)
}
