package database

import (
	"errors"
	"fmt"
	"gokafka/migrations"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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

	return db
}

func RunMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		var dirtyErr migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			// dirty state: undo partial migration and retry
			if ferr := m.Force(dirtyErr.Version); ferr != nil {
				return fmt.Errorf("migrate force clean: %w", ferr)
			}
			if derr := m.Down(); derr != nil && !errors.Is(derr, migrate.ErrNoChange) {
				return fmt.Errorf("migrate down after dirty: %w", derr)
			}
			if rerr := m.Up(); rerr != nil && !errors.Is(rerr, migrate.ErrNoChange) {
				return fmt.Errorf("migrate up retry: %w", rerr)
			}
			return nil
		}
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
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

	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbName, port, sslMode)
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
