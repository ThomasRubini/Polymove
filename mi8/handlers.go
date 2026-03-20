package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"mi8/proto"
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

func (s *server) CreateScore(ctx context.Context, req *proto.CreateScoreRequest) (*proto.CityScore, error) {
	score := req.Score

	err := rdb.HSet(ctx, "city_score:"+score.City, map[string]interface{}{
		"city":    score.City,
		"safety":  score.Safety,
		"economy": score.Economy,
		"qol":     score.Qol,
		"culture": score.Culture,
	}).Err()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save score: %v", err)
	}

	return score, nil
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

func (s *server) CreateNews(ctx context.Context, req *proto.CreateNewsRequest) (*proto.News, error) {
	newsID, err := rdb.Incr(ctx, "news_count").Result()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate news ID: %v", err)
	}

	createdAt := time.Now().UTC()

	tagsStr := strings.Join(req.Tags, ",")

	err = rdb.HSet(ctx, fmt.Sprintf("news:%d", newsID), map[string]interface{}{
		"id":         newsID,
		"city":       req.City,
		"title":      req.Title,
		"content":    req.Content,
		"created_at": createdAt.Format(time.RFC3339),
		"tags":       tagsStr,
	}).Err()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save news: %v", err)
	}

	err = rdb.SAdd(ctx, "city:news:"+req.City, newsID).Err()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add news to city set: %v", err)
	}

	applyTagEffects(ctx, req.City, req.Tags)
	updateCityRelevance(ctx, req.City)

	return &proto.News{
		Id:        int32(newsID),
		City:      req.City,
		Title:     req.Title,
		Content:   req.Content,
		CreatedAt: createdAt.Format(time.RFC3339),
		Tags:      req.Tags,
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
