FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/main .

# Создаем директорию для статики
RUN mkdir -p static
COPY --from=builder /app/static ./static

# Экспортируем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./main"]