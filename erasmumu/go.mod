module github.com/thomasrubini/polymove/erasmumu

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/thomasrubini/polymove/common v0.0.0
)

replace github.com/thomasrubini/polymove/common => ../common