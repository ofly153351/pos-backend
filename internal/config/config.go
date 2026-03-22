package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppName          string
	Host             string
	Port             string
	TokenKey         string
	TokenTTL         int64
	CORSAllowOrigins string
	DatabaseURLValue string
	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	DBSSLMode        string
	AutoMigrate      bool
	MigrationsDir    string
	UploadDir        string
}

func Load() Config {
	return Config{
		AppName:          getEnv("APP_NAME", "pos-backend"),
		Host:             getEnv("APP_HOST", "0.0.0.0"),
		Port:             getEnv("APP_PORT", "8080"),
		TokenKey:         getEnv("APP_TOKEN_KEY", "change-this-secret"),
		TokenTTL:         24,
		CORSAllowOrigins: getEnv("APP_CORS_ALLOW_ORIGINS", "http://localhost:3000"),
		DatabaseURLValue: getEnv("DATABASE_URL", ""),
		DBHost:           getEnv("POSTGRES_HOST", "127.0.0.1"),
		DBPort:           getEnv("POSTGRES_PORT", "5432"),
		DBName:           getEnv("POSTGRES_DB", "pos_db"),
		DBUser:           getEnv("POSTGRES_USER", "postgres"),
		DBPassword:       getEnv("POSTGRES_PASSWORD", "change-me"),
		DBSSLMode:        getEnv("POSTGRES_SSLMODE", "disable"),
		AutoMigrate:      getEnvBool("APP_AUTO_MIGRATE", true),
		MigrationsDir:    getEnv("APP_MIGRATIONS_DIR", "init-db"),
		UploadDir:        getEnv("APP_UPLOAD_DIR", "storage"),
	}
}

func (c Config) HTTPAddress() string {
	return c.Host + ":" + c.Port
}

func (c Config) DatabaseURL() string {
	if c.DatabaseURLValue != "" {
		return c.DatabaseURLValue
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
