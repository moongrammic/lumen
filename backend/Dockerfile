FROM golang:1.25-alpine

WORKDIR /app

# 1. Копируем зависимости
COPY go.mod go.sum ./

# 2. Копируем остальной код
COPY . .

# 3. Причесываем и скачиваем
RUN go mod tidy && go mod download

# 4. СОБИРАЕМ В ПАПКУ /usr/local/bin (её не перекроет Volume!)
RUN go build -o /usr/local/bin/lumen-app ./cmd/api/main.go
RUN chmod +x /usr/local/bin/lumen-app

EXPOSE 8080

# 5. Запускаем из системной папки
CMD ["lumen-app"]