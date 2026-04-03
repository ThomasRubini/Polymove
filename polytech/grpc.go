package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/thomasrubini/polymove/common"
	"github.com/thomasrubini/polymove/common/proto"
)

var (
	mi8Client   proto.MI8ServiceClient
	mi8ConnOnce sync.Once
)

const mi8RPCTimeout = 1500 * time.Millisecond

func getMI8Client() proto.MI8ServiceClient {
	mi8ConnOnce.Do(func() {
		host := getEnv("MI8_GRPC_HOST", "localhost")
		port := getEnv("MI8_GRPC_PORT", "9090")

		addr := net.JoinHostPort(host, port)
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to MI8 gRPC server: %v", err)
		}
		mi8Client = proto.NewMI8ServiceClient(conn)
	})
	return mi8Client
}

func getCityScoresFromMI8(ctx context.Context, city string) (*common.CityScore, error) {
	client := getMI8Client()
	rpcCtx, cancel := context.WithTimeout(ctx, mi8RPCTimeout)
	defer cancel()

	resp, err := client.GetScores(rpcCtx, &proto.GetScoresRequest{City: city})
	if err != nil {
		return nil, fmt.Errorf("mi8 GetScores failed for city %q: %w", city, err)
	}

	if len(resp.Scores) == 0 {
		return nil, nil
	}

	s := resp.Scores[0]
	return &common.CityScore{
		Safety:    s.Safety,
		Economy:   s.Economy,
		QoL:       s.Qol,
		Culture:   s.Culture,
		Relevance: s.Relevance,
	}, nil
}

func getNewsFromMI8(ctx context.Context, city string) ([]common.News, error) {
	client := getMI8Client()
	rpcCtx, cancel := context.WithTimeout(ctx, mi8RPCTimeout)
	defer cancel()

	resp, err := client.GetNews(rpcCtx, &proto.GetNewsRequest{City: city})
	if err != nil {
		return nil, fmt.Errorf("mi8 GetNews failed for city %q: %w", city, err)
	}

	news := make([]common.News, 0, len(resp.News))
	for _, n := range resp.News {
		news = append(news, common.News{
			ID:        int(n.Id),
			City:      n.City,
			Title:     n.Title,
			Content:   n.Content,
			CreatedAt: n.CreatedAt,
			Tags:      n.Tags,
		})
	}
	return news, nil
}
