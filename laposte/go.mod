module github.com/thomasrubini/polymove/laposte

go 1.21

replace github.com/thomasrubini/polymove/common => ../common

require (
	github.com/gorilla/mux v1.8.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/thomasrubini/polymove/common v0.0.0-00010101000000-000000000000
)
