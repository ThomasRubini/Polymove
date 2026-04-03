package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thomasrubini/polymove/common"
	"github.com/thomasrubini/polymove/common/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const baseCityScore = 1000.0

var tagEffects = map[string]map[string]float64{
	"innovation":    {"safety": 20, "economy": 60, "qol": 30, "culture": 5},
	"culture":       {"safety": 15, "economy": 40, "qol": 75},
	"healthcare":    {"safety": 30, "economy": 20, "qol": 30},
	"entertainment": {"economy": 20, "qol": 25, "culture": 35},
	"crisis":        {"safety": -80, "economy": -100, "qol": -60, "culture": -30},
	"crime":         {"safety": -120, "economy": -50, "qol": -80, "culture": -40},
	"disaster":      {"safety": -100, "economy": -70, "qol": -90, "culture": -30},
}

type server struct {
	proto.UnimplementedMI8ServiceServer
}

type NewsEvent struct {
	City    string   `json:"city"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func (s *server) GetScores(ctx context.Context, req *proto.GetScoresRequest) (*proto.GetScoresResponse, error) {
	city := req.City

	var scores []*proto.CityScore

	if city != "" {
		score, err := getScoreFromRedis(ctx, city)
		if err == nil {
			scores = append(scores, score)
		}
	} else {
		cities, err := rdb.Keys(ctx, "city_score:*").Result()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get city keys: %v", err)
		}

		for _, key := range cities {
			cityName := key[11:]
			score, err := getScoreFromRedis(ctx, cityName)
			if err == nil {
				scores = append(scores, score)
			}
		}
	}

	return &proto.GetScoresResponse{Scores: scores}, nil
}

func getScoreFromRedis(ctx context.Context, city string) (*proto.CityScore, error) {
	data, err := rdb.HGetAll(ctx, "city_score:"+city).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("city not found")
	}

	safety, _ := strconv.ParseFloat(data["safety"], 64)
	economy, _ := strconv.ParseFloat(data["economy"], 64)
	qol, _ := strconv.ParseFloat(data["qol"], 64)
	culture, _ := strconv.ParseFloat(data["culture"], 64)
	relevance, _ := strconv.ParseFloat(data["relevance"], 64)

	return &proto.CityScore{
		City:      city,
		Safety:    safety,
		Economy:   economy,
		Qol:       qol,
		Culture:   culture,
		Relevance: relevance,
	}, nil
}

func (s *server) GetNews(ctx context.Context, req *proto.GetNewsRequest) (*proto.GetNewsResponse, error) {
	city := req.City

	var newsList []*proto.News

	if city != "" {
		newsIDs, err := rdb.SMembers(ctx, "city:news:"+city).Result()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get news IDs: %v", err)
		}

		for _, idStr := range newsIDs {
			news, err := getNewsFromRedis(ctx, "news:"+idStr)
			if err == nil && news.Id != 0 {
				newsList = append(newsList, news)
			}
		}
	} else {
		keys, err := rdb.Keys(ctx, "news:*").Result()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get news keys: %v", err)
		}

		for _, key := range keys {
			news, err := getNewsFromRedis(ctx, key)
			if err == nil && news.Id != 0 {
				newsList = append(newsList, news)
			}
		}
	}

	return &proto.GetNewsResponse{News: newsList}, nil
}

// GetCityStats returns offer counters and domain breakdown for a specific city.
func (s *server) GetCityStats(ctx context.Context, req *proto.GetCityStatsRequest) (*proto.CityStats, error) {
	if req.City == "" {
		return nil, status.Error(codes.InvalidArgument, "city is required")
	}

	stats, err := getCityStatsFromRedis(ctx, req.City)
	if err == nil {
		return stats, nil
	}

	return &proto.CityStats{
		City:           req.City,
		TotalOffers:    0,
		OffersByDomain: map[string]int32{},
		LastOfferDate:  "",
	}, nil
}

func getNewsFromRedis(ctx context.Context, key string) (*proto.News, error) {
	data, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("news not found")
	}

	id, _ := strconv.Atoi(data["id"])
	createdAt, _ := time.Parse(time.RFC3339, data["created_at"])
	tags := strings.Split(data["tags"], ",")
	if tags[0] == "" {
		tags = []string{}
	}

	return &proto.News{
		Id:        int32(id),
		City:      data["city"],
		Title:     data["title"],
		Content:   data["content"],
		CreatedAt: createdAt.Format(time.RFC3339),
		Tags:      tags,
	}, nil
}

// processNewsEvent validates and converts a RabbitMQ payload into a stored news entry.
func processNewsEvent(ctx context.Context, payload []byte) error {
	var event NewsEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal news event: %w", err)
	}

	if event.City == "" || event.Title == "" {
		return fmt.Errorf("invalid news event: city and title are required")
	}

	_, err := createNewsRecord(ctx, event.City, event.Title, event.Content, event.Tags)
	return err
}

// processOfferCreatedEvent validates and stores city offer counters from a RabbitMQ payload.
func processOfferCreatedEvent(ctx context.Context, payload []byte) error {
	var event common.OfferCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal offer event: %w", err)
	}

	if event.City == "" || event.Domain == "" || event.OfferID <= 0 {
		return fmt.Errorf("invalid offer.created event")
	}

	if err := updateCityOfferStats(ctx, event); err != nil {
		return fmt.Errorf("failed to update city stats: %w", err)
	}

	return nil
}

// createNewsRecord persists one news item and updates city scores and relevance.
func createNewsRecord(ctx context.Context, city, title, content string, tags []string) (*proto.News, error) {
	newsID, err := rdb.Incr(ctx, "news_count").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to generate news ID: %w", err)
	}

	createdAt := time.Now().UTC()

	tagsStr := strings.Join(tags, ",")

	err = rdb.HSet(ctx, fmt.Sprintf("news:%d", newsID), map[string]interface{}{
		"id":         newsID,
		"city":       city,
		"title":      title,
		"content":    content,
		"created_at": createdAt.Format(time.RFC3339),
		"tags":       tagsStr,
	}).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to save news: %w", err)
	}

	err = rdb.SAdd(ctx, "city:news:"+city, newsID).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to add news to city set: %w", err)
	}

	applyTagEffects(ctx, city, tags)
	updateCityRelevance(ctx, city)

	return &proto.News{
		Id:        int32(newsID),
		City:      city,
		Title:     title,
		Content:   content,
		CreatedAt: createdAt.Format(time.RFC3339),
		Tags:      tags,
	}, nil
}

// updateCityOfferStats increments per-city and per-domain offer counters.
func updateCityOfferStats(ctx context.Context, event common.OfferCreatedEvent) error {
	cityStatsKey := "city_offer_stats:" + event.City
	cityDomainStatsKey := "city_offer_stats_domain:" + event.City

	pipe := rdb.TxPipeline()
	pipe.HIncrBy(ctx, cityStatsKey, "total_offers", 1)
	pipe.HSet(ctx, cityStatsKey, "city", event.City)
	pipe.HSet(ctx, cityStatsKey, "last_offer_date", event.CreatedAt)
	pipe.HIncrBy(ctx, cityDomainStatsKey, event.Domain, 1)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

// getCityStatsFromRedis loads city offer stats and domain counters from Redis.
func getCityStatsFromRedis(ctx context.Context, city string) (*proto.CityStats, error) {
	stats, err := rdb.HGetAll(ctx, "city_offer_stats:"+city).Result()
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("city stats not found")
	}

	domainCounts, err := rdb.HGetAll(ctx, "city_offer_stats_domain:"+city).Result()
	if err != nil {
		return nil, err
	}

	totalOffers, _ := strconv.Atoi(stats["total_offers"])
	offersByDomain := make(map[string]int32, len(domainCounts))
	for domain, countRaw := range domainCounts {
		count, parseErr := strconv.Atoi(countRaw)
		if parseErr != nil {
			continue
		}
		offersByDomain[domain] = int32(count)
	}

	return &proto.CityStats{
		City:           city,
		TotalOffers:    int32(totalOffers),
		OffersByDomain: offersByDomain,
		LastOfferDate:  stats["last_offer_date"],
	}, nil
}

func applyTagEffects(ctx context.Context, city string, tags []string) {
	currentScore, err := getScoreFromRedis(ctx, city)
	if err != nil {
		initCityScore(ctx, city)
		currentScore, _ = getScoreFromRedis(ctx, city)
	}

	safety := currentScore.Safety
	economy := currentScore.Economy
	qol := currentScore.Qol
	culture := currentScore.Culture

	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		effects, exists := tagEffects[tagLower]
		if !exists {
			continue
		}

		if safetyEffect, ok := effects["safety"]; ok {
			safety = max(0, safety+safetyEffect)
		}
		if economyEffect, ok := effects["economy"]; ok {
			economy = max(0, economy+economyEffect)
		}
		if qolEffect, ok := effects["qol"]; ok {
			qol = max(0, qol+qolEffect)
		}
		if cultureEffect, ok := effects["culture"]; ok {
			culture = max(0, culture+cultureEffect)
		}
	}

	rdb.HSet(ctx, "city_score:"+city, map[string]interface{}{
		"safety":  safety,
		"economy": economy,
		"qol":     qol,
		"culture": culture,
	})
}

func initCityScore(ctx context.Context, city string) {
	rdb.HSet(ctx, "city_score:"+city, map[string]interface{}{
		"city":      city,
		"safety":    baseCityScore,
		"economy":   baseCityScore,
		"qol":       baseCityScore,
		"culture":   baseCityScore,
		"relevance": 0,
	})
}

func updateCityRelevance(ctx context.Context, city string) {
	newsCount, _ := rdb.SCard(ctx, "city:news:"+city).Result()

	rdb.HSet(ctx, "city_score:"+city, "relevance", newsCount)
}
