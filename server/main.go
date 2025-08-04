// server/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	db "start/db/sqlc"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest определяет структуру JSON-тела для запроса на регистрацию
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest определяет структуру для запроса на вход
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ВАЖНО: Этот ключ используется для подписи токенов.
// В реальном приложении он ДОЛЖЕН быть загружен из переменных окружения и быть гораздо сложнее!
var jwtSecret = []byte("my-super-secret-key")

// UserIDKey - это ключ для хранения ID пользователя в контексте запроса
type contextKey string

const UserIDKey contextKey = "userID"

func main() {
	// ЗАМЕНИ YOUR_PASSWORD на свой пароль от PostgreSQL
	connStr := "postgres://postgres:memodel@localhost:5432/concordia_db"

	dbpool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Ошибка: не удалось подключиться к базе данных: %v\n", err)
	}
	defer dbpool.Close()

	// Создаем экземпляр нашего сгенерированного клиента к БД
	queries := db.New(dbpool)

	// --- Настройка роутера ---
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)    // Логирует все запросы в консоль
	r.Use(middleware.Recoverer) // Восстанавливается после паники
	r.Use(middleware.Timeout(60 * time.Second))

	// --- Определение маршрутов (Routes) ---
	r.Route("/api", func(r chi.Router) {
		// Публичные роуты (не требуют токена)
		r.Post("/auth/register", registerHandler(queries))
		r.Post("/auth/login", loginHandler(queries))

		// Защищенные роуты (требуют токен)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware) // Применяем наш "охранник" ко всей группе

			r.Get("/me", meHandler(queries))

			// Роуты для пространств
			r.Post("/workspaces", createWorkspaceHandler(queries))
			r.Get("/workspaces", listWorkspacesHandler(queries))

			// 👇 Наши новые вложенные роуты для каналов
			r.Route("/workspaces/{workspaceID}", func(r chi.Router) {
				r.Post("/channels", createChannelHandler(queries))
				r.Get("/channels", listChannelsHandler(queries))
			})
		})
	})

	log.Println("✅ Сервер запущен на порту :8080")
	http.ListenAndServe(":8080", r)
}

// registerHandler - это наш обработчик запроса на регистрацию
func registerHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Парсим JSON из тела запроса
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
			return
		}

		// 2. Хешируем пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}

		// 3. Вызываем сгенерированную sqlc функцию для создания пользователя
		params := db.CreateUserParams{
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: pgtype.Text{String: string(hashedPassword), Valid: true},
		}
		user, err := queries.CreateUser(r.Context(), params)
		if err != nil {
			// В реальном приложении здесь нужна проверка на дубликат email/username
			http.Error(w, "Не удалось создать пользователя", http.StatusInternalServerError)
			return
		}

		// 4. Отправляем успешный ответ (ВАЖНО: без хеша пароля)
		// Для этого можно создать отдельную структуру UserResponse
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"created_at": user.CreatedAt,
		})
	}
}

// loginHandler - это наш обработчик запроса на вход (ИСПРАВЛЕННАЯ ВЕРСИЯ)
func loginHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Парсим JSON
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
			return
		}

		// 2. Находим пользователя по email
		user, err := queries.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
			return
		}

		// 3. Сравниваем хеш пароля
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(req.Password)); err != nil {
			http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
			return
		}

		// 4. ПРАВИЛЬНОЕ преобразование pgtype.UUID в строку
		// Создаем google/uuid.UUID из байтов, которые хранятся в pgtype.UUID
		userID := uuid.UUID(user.ID.Bytes)
		// Теперь получаем стандартную строку с дефисами
		userIDString := userID.String()

		// 5. Генерируем JWT-токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": userIDString,
			"exp":     time.Now().Add(time.Hour * 72).Unix(),
		})

		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			http.Error(w, "Ошибка при создании токена", http.StatusInternalServerError)
			return
		}

		// 6. Отправляем токен клиенту
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": tokenString,
		})
	}
}

// authMiddleware - наш "охранник"
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
			return
		}

		// 2. Проверяем, что заголовок в формате "Bearer <token>"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(w, "Неверный формат заголовка авторизации", http.StatusUnauthorized)
			return
		}
		tokenString := headerParts[1]

		// 3. Парсим и валидируем токен
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			// Добавляем лог, чтобы видеть настоящую причину ошибки в консоли сервера
			log.Printf("Ошибка валидации токена: %v", err)
			http.Error(w, "Невалидный токен", http.StatusUnauthorized)
			return
		}

		// 4. Извлекаем данные пользователя (claims) и ID
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID, ok := claims["user_id"].(string)
			if !ok {
				http.Error(w, "Невалидный токен", http.StatusUnauthorized)
				return
			}
			// 5. Сохраняем ID пользователя в контекст запроса для дальнейшего использования
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Невалидный токен", http.StatusUnauthorized)
		}
	})
}

// Заменяем старый meHandler на новый с поддержкой uuid и возвратом данных пользователя
func meHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDString, ok := r.Context().Value(UserIDKey).(string)
		if !ok {
			http.Error(w, "Не удалось получить ID пользователя", http.StatusInternalServerError)
			return
		}

		// Парсим строку в uuid.UUID
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пользователя", http.StatusInternalServerError)
			return
		}

		// Преобразуем uuid.UUID в pgtype.UUID
		var pgUserID pgtype.UUID
		pgUserID = pgtype.UUID{
			Bytes: userID,
			Valid: true,
		}

		// Получаем пользователя из базы
		user, err := queries.GetUserByID(r.Context(), pgUserID)
		if err != nil {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// --- Добавляем обработчики для workspaces ---
type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

// CreateChannelRequest определяет структуру JSON-тела для запроса на создание канала
type CreateChannelRequest struct {
	Name string             `json:"name"`
	Type db.NullChannelType `json:"type"` // Используем тип из sqlc
}

func createWorkspaceHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDString := r.Context().Value(UserIDKey).(string)
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пользователя", http.StatusInternalServerError)
			return
		}

		var pgUserID pgtype.UUID
		pgUserID = pgtype.UUID{
			Bytes: userID,
			Valid: true,
		}

		var req CreateWorkspaceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
			return
		}

		// Создаем workspace
		wsParams := db.CreateWorkspaceParams{
			Name:    req.Name,
			OwnerID: pgUserID, // Используем правильный тип
		}
		workspace, err := queries.CreateWorkspace(r.Context(), wsParams)
		if err != nil {
			http.Error(w, "Не удалось создать пространство", http.StatusInternalServerError)
			return
		}

		// Добавляем создателя как участника
		memberParams := db.AddWorkspaceMemberParams{
			UserID:      pgUserID, // Используем правильный тип
			WorkspaceID: workspace.ID,
		}
		_, err = queries.AddWorkspaceMember(r.Context(), memberParams)
		if err != nil {
			http.Error(w, "Не удалось добавить участника", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(workspace)
	}
}

func listWorkspacesHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDString := r.Context().Value(UserIDKey).(string)
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пользователя", http.StatusInternalServerError)
			return
		}

		var pgUserID pgtype.UUID
		pgUserID = pgtype.UUID{
			Bytes: userID,
			Valid: true,
		}

		workspaces, err := queries.ListUserWorkspaces(r.Context(), pgUserID) // Используем правильный тип
		if err != nil {
			http.Error(w, "Не удалось получить список пространств", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workspaces)
	}
}

func createChannelHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем ID из URL и контекста
		workspaceIDString := chi.URLParam(r, "workspaceID")
		workspaceID, err := uuid.Parse(workspaceIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пространства", http.StatusBadRequest)
			return
		}
		var pgWorkspaceID pgtype.UUID
		pgWorkspaceID = pgtype.UUID{
			Bytes: workspaceID,
			Valid: true,
		}

		userIDString := r.Context().Value(UserIDKey).(string)
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пользователя", http.StatusInternalServerError)
			return
		}
		var pgUserID pgtype.UUID
		pgUserID = pgtype.UUID{
			Bytes: userID,
			Valid: true,
		}

		// 2. Проверяем, является ли пользователь участником этого пространства
		isMember, err := queries.IsWorkspaceMember(r.Context(), db.IsWorkspaceMemberParams{UserID: pgUserID, WorkspaceID: pgWorkspaceID})
		if err != nil || !isMember {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}

		// 3. Парсим тело запроса с временной структурой
		type TempChannelRequest struct {
			Name string `json:"name"`
			Type string `json:"type"` // Временная строка
		}
		var tempReq TempChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&tempReq); err != nil {
			http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
			return
		}

		// 4. Преобразуем type в db.NullChannelType
		var channelType db.NullChannelType
		if tempReq.Type != "" {
			switch strings.ToUpper(tempReq.Type) {
			case "TEXT":
				channelType = db.NullChannelType{ChannelType: db.ChannelTypeTEXT, Valid: true}
			case "VOICE":
				channelType = db.NullChannelType{ChannelType: db.ChannelTypeVOICE, Valid: true}
			default:
				http.Error(w, "Недопустимый тип канала", http.StatusBadRequest)
				return
			}
		} else {
			channelType = db.NullChannelType{ChannelType: db.ChannelTypeTEXT, Valid: true} // Значение по умолчанию
		}

		// 5. Создаем канал
		params := db.CreateChannelParams{
			WorkspaceID: pgWorkspaceID,
			Name:        tempReq.Name,
			Type:        channelType.ChannelType,
		}
		channel, err := queries.CreateChannel(r.Context(), params)
		if err != nil {
			http.Error(w, "Не удалось создать канал", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(channel)
	}
}

func listChannelsHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceIDString := chi.URLParam(r, "workspaceID")
		workspaceID, err := uuid.Parse(workspaceIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пространства", http.StatusBadRequest)
			return
		}
		var pgWorkspaceID pgtype.UUID
		pgWorkspaceID = pgtype.UUID{
			Bytes: workspaceID,
			Valid: true,
		}

		userIDString := r.Context().Value(UserIDKey).(string)
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			http.Error(w, "Неверный формат ID пользователя", http.StatusInternalServerError)
			return
		}
		var pgUserID pgtype.UUID
		pgUserID = pgtype.UUID{
			Bytes: userID,
			Valid: true,
		}

		// Проверяем, является ли пользователь участником этого пространства
		isMember, err := queries.IsWorkspaceMember(r.Context(), db.IsWorkspaceMemberParams{UserID: pgUserID, WorkspaceID: pgWorkspaceID})
		if err != nil || !isMember {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}

		channels, err := queries.ListWorkspaceChannels(r.Context(), pgWorkspaceID)
		if err != nil {
			http.Error(w, "Не удалось получить список каналов", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(channels)
	}
}
