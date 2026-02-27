FROM golang:1.21-alpine
WORKDIR /app

COPY * ./
RUN go build -o /erasmumu

EXPOSE 8081
CMD ["/erasmumu"]
