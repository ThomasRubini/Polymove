FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod ./
COPY *.go ./
RUN go get github.com/gorilla/mux github.com/lib/pq

RUN go build -o /erasmumu

EXPOSE 8081

CMD ["/erasmumu"]
