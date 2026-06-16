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

# Копируем бинарник из builder
COPY --from=builder /app/gateway .

# Открываем порт
EXPOSE 8080

# Запускаем
CMD ["./gateway"]
