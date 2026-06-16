FROM golang:1.26-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN go build -o gateway ./main.go

# Финальный образ
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gateway .

RUN apk add --no-cache ca-certificates

EXPOSE 8081

CMD ["./gateway"]
