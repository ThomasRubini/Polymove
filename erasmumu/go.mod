module github.com/thomasrubini/polymove/erasmumu

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/thomasrubini/polymove/common v0.0.0
)

require github.com/rabbitmq/amqp091-go v1.10.0 // indirect

replace github.com/thomasrubini/polymove/common => ../common
