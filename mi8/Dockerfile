FROM golang:1.21-alpine
WORKDIR /app

COPY * ./
RUN go build -o /mi8

EXPOSE 8082
CMD ["/mi8"]
