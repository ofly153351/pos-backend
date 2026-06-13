package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	AppName          string
	Env              string
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
	MinIOEndpoint    string
	MinIOAccessKey   string
	MinIOSecretKey   string
	MinIOBucketName  string
	MinIOUseSSL      bool
	MinIOPublicURL   string
}

var dotenvOnce sync.Once

// defaultTokenKey is the placeholder JWT signing secret shipped in .env.example.
// It is only acceptable in local development (APP_ENV=development).
const defaultTokenKey = "change-this-secret"

func Load() Config {
	loadDotEnvFile()

	return Config{
		AppName:          getEnv("APP_NAME", "pos-backend"),
		Env:              strings.ToLower(getEnv("APP_ENV", "development")),
		Host:             getEnv("APP_HOST", "0.0.0.0"),
		Port:             getEnv("APP_PORT", "8080"),
		TokenKey:         getEnv("APP_TOKEN_KEY", defaultTokenKey),
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
		MinIOEndpoint:    getEnv("MINIO_ENDPOINT", "127.0.0.1:9000"),
		MinIOAccessKey:   firstEnv("MINIO_ACCESS_KEY", "MINIO_ROOT_USER", "minioadmin"),
		MinIOSecretKey:   firstEnv("MINIO_SECRET_KEY", "MINIO_ROOT_PASSWORD", "change-me"),
		MinIOBucketName:  getEnv("MINIO_BUCKET_NAME", "pos-assets"),
		MinIOUseSSL:      getEnvBool("MINIO_USE_SSL", false),
		MinIOPublicURL:   getEnv("MINIO_PUBLIC_URL", "http://127.0.0.1:9000"),
	}
}

func (c Config) HTTPAddress() string {
	return c.Host + ":" + c.Port
}

// Validate enforces production-safety invariants. It must be called once at
// startup (see cmd/api.go) before the server boots. It refuses to start a
// staging/production process that is still using the shipped default secret.
func (c Config) Validate() error {
	switch c.Env {
	case "production", "prod", "staging":
		if c.TokenKey == defaultTokenKey {
			return fmt.Errorf(
				"APP_TOKEN_KEY is still the insecure default %q but APP_ENV=%q: set APP_TOKEN_KEY to a strong random secret (e.g. `openssl rand -hex 32`) before starting in %s",
				defaultTokenKey, c.Env, c.Env,
			)
		}
	}
	return nil
}

func (c Config) DatabaseURL() string {
	if c.DatabaseURLValue != "" {
		return c.DatabaseURLValue
	}

	// client_encoding=UTF8 ensures the client↔server connection negotiates UTF-8
	// so Thai text round-trips correctly (paired with the UTF8 DB from docker-compose).
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&client_encoding=UTF8",
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

func firstEnv(primary, secondary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	if value := os.Getenv(secondary); value != "" {
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

func loadDotEnvFile() {
	dotenvOnce.Do(func() {
		file, err := os.Open(".env")
		if err != nil {
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}

			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}

			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			_ = os.Setenv(key, value)
		}
	})
}
