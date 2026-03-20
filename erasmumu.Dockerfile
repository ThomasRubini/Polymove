FROM golang:1.21-alpine

RUN apk add --no-cache protobuf && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

WORKDIR /app

COPY common/ common/
RUN cd common && go mod download && go generate ./...

COPY erasmumu/ erasmumu/
RUN cd erasmumu && go build -o /erasmumu

WORKDIR /app/erasmumu
EXPOSE 8082
CMD ["/erasmumu"]
