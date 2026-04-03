package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/thomasrubini/polymove/common"
)

type Student struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

var seedStudents = []Student{
	{Name: "Alice Martin", Domain: "software"},
	{Name: "Bilal Rahmani", Domain: "cybersecurity"},
	{Name: "Chloe Bernard", Domain: "data"},
	{Name: "Diego Rossi", Domain: "software"},
	{Name: "Emma Leroy", Domain: "networks"},
}

var seedOffers = []common.Offer{
	{
		Title:     "Backend Engineering Intern",
		Link:      "https://example.com/offers/backend-engineering-intern",
		City:      "Lyon",
		Domain:    "software",
		Salary:    1400,
		StartDate: "2026-06-01",
		EndDate:   "2026-08-31",
		Available: true,
	},
	{
		Title:     "Cybersecurity Operations Intern",
		Link:      "https://example.com/offers/cybersecurity-operations-intern",
		City:      "Berlin",
		Domain:    "cybersecurity",
		Salary:    1550,
		StartDate: "2026-06-15",
		EndDate:   "2026-09-15",
		Available: true,
	},
	{
		Title:     "Data Analyst Intern",
		Link:      "https://example.com/offers/data-analyst-intern",
		City:      "Barcelona",
		Domain:    "data",
		Salary:    1300,
		StartDate: "2026-05-15",
		EndDate:   "2026-08-15",
		Available: true,
	},
	{
		Title:     "Cloud Network Intern",
		Link:      "https://example.com/offers/cloud-network-intern",
		City:      "Amsterdam",
		Domain:    "networks",
		Salary:    1500,
		StartDate: "2026-06-01",
		EndDate:   "2026-09-01",
		Available: true,
	},
	{
		Title:     "Full Stack Developer Intern",
		Link:      "https://example.com/offers/full-stack-developer-intern",
		City:      "Paris",
		Domain:    "software",
		Salary:    1450,
		StartDate: "2026-07-01",
		EndDate:   "2026-10-01",
		Available: true,
	},
}

func main() {
	polytechURL := envOrDefault("POLYTECH_URL", "http://localhost:8080")
	erasMumuURL := envOrDefault("ERASMUMU_URL", "http://localhost:8081")

	client := &http.Client{Timeout: 10 * time.Second}

	if err := seedPolytechStudents(client, polytechURL); err != nil {
		exitWithError(err)
	}

	if err := seedErasmumuOffers(client, erasMumuURL); err != nil {
		exitWithError(err)
	}

	fmt.Println("Seed completed successfully.")
	fmt.Printf("Polytech: %s\n", polytechURL)
	fmt.Printf("Erasmumu: %s\n", erasMumuURL)
	fmt.Printf("Students seeded: %d\n", len(seedStudents))
	fmt.Printf("Offers seeded: %d\n", len(seedOffers))
}

func seedPolytechStudents(client *http.Client, baseURL string) error {
	existing, err := fetchStudents(client, baseURL+"/student")
	if err != nil {
		return fmt.Errorf("fetch existing students: %w", err)
	}

	for _, student := range seedStudents {
		if hasStudent(existing, student) {
			fmt.Printf("Skipping existing student: %s (%s)\n", student.Name, student.Domain)
			continue
		}

		if err := postJSON(client, baseURL+"/student", student); err != nil {
			return fmt.Errorf("create student %q: %w", student.Name, err)
		}

		fmt.Printf("Created student: %s (%s)\n", student.Name, student.Domain)
	}

	return nil
}

func seedErasmumuOffers(client *http.Client, baseURL string) error {
	existing, err := fetchOffers(client, baseURL+"/offers")
	if err != nil {
		return fmt.Errorf("fetch existing offers: %w", err)
	}

	for _, offer := range seedOffers {
		if hasOffer(existing, offer) {
			fmt.Printf("Skipping existing offer: %s (%s, %s)\n", offer.Title, offer.City, offer.Domain)
			continue
		}

		if err := postJSON(client, baseURL+"/offers", offer); err != nil {
			return fmt.Errorf("create offer %q: %w", offer.Title, err)
		}

		fmt.Printf("Created offer: %s (%s, %s)\n", offer.Title, offer.City, offer.Domain)
	}

	return nil
}

func fetchStudents(client *http.Client, url string) ([]Student, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp)
	}

	var students []Student
	if err := json.NewDecoder(resp.Body).Decode(&students); err != nil {
		return nil, err
	}

	return students, nil
}

func fetchOffers(client *http.Client, url string) ([]common.Offer, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp)
	}

	var offers []common.Offer
	if err := json.NewDecoder(resp.Body).Decode(&offers); err != nil {
		return nil, err
	}

	return offers, nil
}

func postJSON(client *http.Client, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return responseError(resp)
	}

	return nil
}

func hasStudent(existing []Student, candidate Student) bool {
	for _, student := range existing {
		if strings.EqualFold(student.Name, candidate.Name) && strings.EqualFold(student.Domain, candidate.Domain) {
			return true
		}
	}

	return false
}

func hasOffer(existing []common.Offer, candidate common.Offer) bool {
	for _, offer := range existing {
		if strings.EqualFold(offer.Title, candidate.Title) &&
			strings.EqualFold(offer.Link, candidate.Link) &&
			strings.EqualFold(offer.City, candidate.City) &&
			strings.EqualFold(offer.Domain, candidate.Domain) {
			return true
		}
	}

	return false
}

func responseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, message)
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
