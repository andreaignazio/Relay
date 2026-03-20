package database

import (
	"fmt"
	"gokafka/internal/models"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	envPaths := []string{".env", "backend/GoGORM/.env", "../.env"}
	envLoaded := false
	for _, envPath := range envPaths {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Printf("warning: .env not loaded from known paths")
	}

	dsn := resolvePostgresDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Error:", err)
	}

	db.AutoMigrate(&models.Workspace{})

	return db

}

func resolvePostgresDSN() string {
	if dsn := firstEnv("DATABASE_URL", "POSTGRES_DSN"); dsn != "" {
		return dsn
	}

	host := firstEnv("DB_HOST", "PGHOST")
	port := firstEnv("DB_PORT", "PGPORT")
	user := firstEnv("DB_USER", "PGUSER")
	password := firstEnv("DB_PASSWORD", "PGPASSWORD")
	dbName := firstEnv("DB_NAME", "PGDATABASE")
	sslMode := firstEnv("DB_SSLMODE", "PGSSLMODE")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbName, port, sslMode)
	// "host=localhost user=postgres password=<tua_password> dbname=gokafka_db port=5432 sslmode=disable"
	return dsn
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return ""
}
