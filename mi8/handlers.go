package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func getScores(w http.ResponseWriter, r *http.Request) error {
	city := r.URL.Query().Get("city")

	var query string
	var args []interface{}

	if city != "" {
		query = "SELECT city, safety, economy, qol, culture FROM city_scores WHERE city = $1"
		args = append(args, city)
	} else {
		query = "SELECT city, safety, economy, qol, culture FROM city_scores"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query scores: %w", err)
	}
	defer rows.Close()

	var scores []CityScore
	for rows.Next() {
		var cs CityScore
		if err := rows.Scan(&cs.City, &cs.Safety, &cs.Economy, &cs.QoL, &cs.Culture); err != nil {
			return fmt.Errorf("failed to scan score: %w", err)
		}
		scores = append(scores, cs)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, scores)
}

func createNews(w http.ResponseWriter, r *http.Request) error {
	var news News
	if err := json.NewDecoder(r.Body).Decode(&news); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	query := "INSERT INTO news (city, title, content) VALUES ($1, $2, $3) RETURNING id, created_at"
	if err := db.QueryRow(query, news.City, news.Title, news.Content).Scan(&news.ID, &news.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert news: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, news)
}
