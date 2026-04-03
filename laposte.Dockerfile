FROM golang:1.21-alpine

WORKDIR /app

COPY common/ common/
COPY laposte/ laposte/

RUN cd laposte && go mod download && go build -o /laposte

WORKDIR /app/laposte
EXPOSE 8083
CMD ["/laposte"]
