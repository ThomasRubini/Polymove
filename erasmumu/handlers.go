package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/thomasrubini/polymove/common"
)

// getOffers handles GET /offers - Lists all offers, optionally filtered by city
func getOffers(w http.ResponseWriter, r *http.Request) error {
	city := r.URL.Query().Get("city")

	var query string
	var args []interface{}

	if city != "" {
		query = "SELECT id, title, link, city, domain, salary, TO_CHAR(start_date, 'YYYY-MM-DD'), TO_CHAR(end_date, 'YYYY-MM-DD'), available FROM offers WHERE city = $1"
		args = append(args, city)
	} else {
		query = "SELECT id, title, link, city, domain, salary, TO_CHAR(start_date, 'YYYY-MM-DD'), TO_CHAR(end_date, 'YYYY-MM-DD'), available FROM offers"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query offers: %w", err)
	}
	defer rows.Close()

	var offers []common.Offer
	for rows.Next() {
		var offer common.Offer
		if err := rows.Scan(&offer.ID, &offer.Title, &offer.Link, &offer.City, &offer.Domain, &offer.Salary, &offer.StartDate, &offer.EndDate, &offer.Available); err != nil {
			return fmt.Errorf("failed to scan offer: %w", err)
		}
		offers = append(offers, offer)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, offers)
}

// createOffer handles POST /offers - Creates a new Erasmus offer
func createOffer(w http.ResponseWriter, r *http.Request) error {
	var offer common.Offer
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	query := "INSERT INTO offers (title, link, city, domain, salary, start_date, end_date, available) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id"
	if err := db.QueryRow(query, offer.Title, offer.Link, offer.City, offer.Domain, offer.Salary, offer.StartDate, offer.EndDate, offer.Available).Scan(&offer.ID); err != nil {
		return fmt.Errorf("failed to insert offer: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, offer)
}

// getOfferByID handles GET /offers/{id} - Retrieves a specific offer by ID
func getOfferByID(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id := vars["id"]

	var offer common.Offer
	query := "SELECT id, title, link, city, domain, salary, TO_CHAR(start_date, 'YYYY-MM-DD'), TO_CHAR(end_date, 'YYYY-MM-DD'), available FROM offers WHERE id = $1"
	err := db.QueryRow(query, id).Scan(&offer.ID, &offer.Title, &offer.Link, &offer.City, &offer.Domain, &offer.Salary, &offer.StartDate, &offer.EndDate, &offer.Available)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("offer with id %s not found", id)
		}
		return fmt.Errorf("failed to get offer: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusOK, offer)
}
