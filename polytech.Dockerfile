FROM golang:1.21-alpine

RUN apk add --no-cache protobuf && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

WORKDIR /app

COPY common/ common/
RUN cd common && go mod download && go generate ./...

COPY polytech/ polytech/
RUN cd polytech && go build -o /polytech

WORKDIR /app/polytech
EXPOSE 8082
CMD ["/polytech"]
