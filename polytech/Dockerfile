FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY *.go ./
RUN go build -o /student-api

EXPOSE 8080

CMD ["/student-api"]
