package main

import (
	"context"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/thomasrubini/polymove/common"
	"github.com/thomasrubini/polymove/common/proto"
)

var (
	mi8Client   proto.MI8ServiceClient
	mi8Conn     *grpc.ClientConn
	mi8ConnOnce sync.Once
)

func getMI8Client() proto.MI8ServiceClient {
	mi8ConnOnce.Do(func() {
		host := getEnv("MI8_GRPC_HOST", "localhost")
		port := getEnv("MI8_GRPC_PORT", "9090")

		addr := net.JoinHostPort(host, port)
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to MI8 gRPC server: %v", err)
		}
		mi8Conn = conn
		mi8Client = proto.NewMI8ServiceClient(conn)
	})
	return mi8Client
}

func closeMI8Client() {
	if mi8Conn != nil {
		mi8Conn.Close()
	}
}

func getCityScoresFromMI8(ctx context.Context, city string) (*common.CityScore, error) {
	client := getMI8Client()
	resp, err := client.GetScores(ctx, &proto.GetScoresRequest{City: city})
	if err != nil {
		return nil, err
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
	resp, err := client.GetNews(ctx, &proto.GetNewsRequest{City: city})
	if err != nil {
		return nil, err
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
