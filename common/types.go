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
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	City      string `json:"city"`
	Domain    string `json:"domain"`
	Salary    int    `json:"salary"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Available bool   `json:"available"`
}

type News struct {
	ID        int      `json:"id"`
	City      string   `json:"city,omitempty"`
	Title     string   `json:"title"`
	Content   string   `json:"content,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}
