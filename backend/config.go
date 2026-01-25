package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Глобальная переменная для пула подключений, доступная во всем пакете main
var db *pgxpool.Pool

// LoadConfig считывает настройки из переменных окружения или использует значения по умолчанию
func LoadConfig() Config {
	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "00000000"),
		DBName:     getEnv("DB_NAME", "postgres"),
	}
}

// getEnv — вспомогательная функция для получения переменной окружения
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetConnectionString формирует строку подключения (DSN) для PostgreSQL
func (c Config) GetConnectionString() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=disable"
}

// initDB инициализирует подключение к базе данных
func initDB() {
	var err error
	config := LoadConfig()
	connStr := config.GetConnectionString()

	// Создаем пул подключений (pgxpool оптимальнее одиночного подключения)
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal("Error creating connection pool:", err)
	}

	// Проверяем физическое наличие связи с БД
	err = db.Ping(context.Background())
	if err != nil {
		log.Fatal("Failed to connect to DB (Ping):", err)
	}

	log.Println("Successfully connected to PostgreSQL!")
}
