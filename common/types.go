//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/common.proto
package common

type CityScore struct {
	City      string  `json:"city,omitempty"`
	Safety    float64 `json:"safety"`
	Economy   float64 `json:"economy"`
	QoL       float64 `json:"qol"`
	Culture   float64 `json:"culture"`
	Relevance float64 `json:"relevance"`
}

type Offer struct {
	ID          int    `json:"id"`
	University  string `json:"university"`
	City        string `json:"city"`
	Country     string `json:"country"`
	Domain      string `json:"domain"`
	Description string `json:"description"`
}

type News struct {
	ID        int      `json:"id"`
	City      string   `json:"city,omitempty"`
	Title     string   `json:"title"`
	Content   string   `json:"content,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}
