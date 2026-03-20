package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// getScores handles GET /scores - Retrieves city scores, optionally filtered by city
// Returns Safety, Economy, QoL, and Culture metrics for each city
func getScores(w http.ResponseWriter, r *http.Request) error {
	city := r.URL.Query().Get("city")

	var scores []CityScore

	if city != "" {
		score, err := getScoreFromRedis(city)
		if err == nil {
			scores = append(scores, score)
		}
	} else {
		cities, err := rdb.Keys(ctx, "city_score:*").Result()
		if err != nil {
			return fmt.Errorf("failed to get city keys: %w", err)
		}

		for _, key := range cities {
			cityName := key[11:]
			score, err := getScoreFromRedis(cityName)
			if err == nil {
				scores = append(scores, score)
			}
		}
	}

	return NewResponseWriter(w).JSON(http.StatusOK, scores)
}

func getScoreFromRedis(city string) (CityScore, error) {
	var score CityScore
	data, err := rdb.HGetAll(ctx, "city_score:"+city).Result()
	if err != nil {
		return score, err
	}

	if len(data) == 0 {
		return score, fmt.Errorf("city not found")
	}

	score.City = city
	score.Safety, _ = strconv.ParseFloat(data["safety"], 64)
	score.Economy, _ = strconv.ParseFloat(data["economy"], 64)
	score.QoL, _ = strconv.ParseFloat(data["qol"], 64)
	score.Culture, _ = strconv.ParseFloat(data["culture"], 64)
	score.Relevance, _ = strconv.ParseFloat(data["relevance"], 64)

	return score, nil
}

// createScore handles POST /scores - Creates or updates city scores
func createScore(w http.ResponseWriter, r *http.Request) error {
	var score CityScore
	if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	err := rdb.HSet(ctx, "city_score:"+score.City, map[string]interface{}{
		"city":    score.City,
		"safety":  score.Safety,
		"economy": score.Economy,
		"qol":     score.QoL,
		"culture": score.Culture,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to save score: %w", err)
	}

	return NewResponseWriter(w).JSON(http.StatusCreated, score)
}

// getNews handles GET /news - Retrieves news articles, optionally filtered by city
func getNews(w http.ResponseWriter, r *http.Request) error {
	city := r.URL.Query().Get("city")

	var newsList []News

	if city != "" {
		newsList = getNewsByCity(city)
	} else {
		keys, err := rdb.Keys(ctx, "news:*").Result()
		if err != nil {
			return fmt.Errorf("failed to get news keys: %w", err)
		}

		for _, key := range keys {
			news, err := getNewsFromRedis(key)
			if err == nil && news.ID != 0 {
				newsList = append(newsList, news)
			}
		}
	}

	return NewResponseWriter(w).JSON(http.StatusOK, newsList)
}

func getNewsByCity(city string) []News {
	var newsList []News

	newsIDs, err := rdb.SMembers(ctx, "city:news:"+city).Result()
	if err != nil {
		return newsList
	}

	for _, id := range newsIDs {
		news, err := getNewsFromRedis("news:" + id)
		if err == nil && news.ID != 0 {
			newsList = append(newsList, news)
		}
	}

	return newsList
}

func getNewsFromRedis(key string) (News, error) {
	var news News
	data, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return news, err
	}

	if len(data) == 0 {
		return news, fmt.Errorf("news not found")
	}

	news.ID, _ = strconv.Atoi(data["id"])
	news.City = data["city"]
	news.Title = data["title"]
	news.Content = data["content"]
	news.CreatedAt, _ = time.Parse(time.RFC3339, data["created_at"])

	return news, nil
}

// createNews handles POST /news - Creates a news article for a city
// Updates city relevance score based on news count
func createNews(w http.ResponseWriter, r *http.Request) error {
	var news News
	if err := json.NewDecoder(r.Body).Decode(&news); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	newsID, err := rdb.Incr(ctx, "news_count").Result()
	if err != nil {
		return fmt.Errorf("failed to generate news ID: %w", err)
	}
	news.ID = int(newsID)

	news.CreatedAt = time.Now().UTC()

	err = rdb.HSet(ctx, fmt.Sprintf("news:%d", news.ID), map[string]interface{}{
		"id":         news.ID,
		"city":       news.City,
		"title":      news.Title,
		"content":    news.Content,
		"created_at": news.CreatedAt.Format(time.RFC3339),
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to save news: %w", err)
	}

	err = rdb.SAdd(ctx, "city:news:"+news.City, news.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to add news to city set: %w", err)
	}

	updateCityRelevance(news.City)

	return NewResponseWriter(w).JSON(http.StatusCreated, news)
}

// updateCityRelevance updates the city relevance score based on news count
func updateCityRelevance(city string) {
	newsCount, _ := rdb.SCard(ctx, "city:news:"+city).Result()

	rdb.HSet(ctx, "city_score:"+city, "relevance", newsCount)
}
