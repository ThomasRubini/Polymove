package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Offer struct {
	ID          int    `json:"id"`
	University  string `json:"university"`
	City        string `json:"city"`
	Country     string `json:"country"`
	Description string `json:"description"`
}

func getOffers(w http.ResponseWriter, r *http.Request) error {
	city := r.URL.Query().Get("city")

	var query string
	var args []interface{}

	if city != "" {
		query = "SELECT id, university, city, country, description FROM offers WHERE city = $1"
		args = append(args, city)
	} else {
		query = "SELECT id, university, city, country, description FROM offers"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query offers: %w", err)
	}
	defer rows.Close()

	var offers []Offer
	for rows.Next() {
		var offer Offer
		if err := rows.Scan(&offer.ID, &offer.University, &offer.City, &offer.Country, &offer.Description); err != nil {
			return fmt.Errorf("failed to scan offer: %w", err)
		}
		offers = append(offers, offer)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, offers)
}

func createOffer(w http.ResponseWriter, r *http.Request) error {
	var offer Offer
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	query := "INSERT INTO offers (university, city, country, description) VALUES ($1, $2, $3, $4) RETURNING id"
	if err := db.QueryRow(query, offer.University, offer.City, offer.Country, offer.Description).Scan(&offer.ID); err != nil {
		return fmt.Errorf("failed to insert offer: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, offer)
}
