FROM golang:1.21-alpine

RUN apk add --no-cache protobuf && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

WORKDIR /app

COPY go.mod go.sum ./
COPY *.go ./
COPY proto/ ./proto/

RUN go mod download && \
    go generate ./... && \
    go build -o /mi8

WORKDIR /app/cmd/colporteur
COPY cmd/colporteur/*.go ./
RUN go build -o /colporteur

WORKDIR /app
EXPOSE 8082
CMD ["/mi8"]
