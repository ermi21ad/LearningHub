package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Server
	ServerPort string
	ServerEnv  string

	// JWT
	JWTSecret string
	JWTExpiry time.Duration

	// File Upload
	MaxImageSize    int64
	MaxVideoSize    int64
	MaxDocumentSize int64

	// Stripe
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string

	// Chapa Payment Integration
	ChapaSecretKey     string
	ChapaWebhookSecret string
	AppBaseURL         string

	// Firebase
	FirebaseCredentialsPath string
	FirebaseBucketName      string

	// Email
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
}

func LoadConfig() (*Config, error) {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	config := &Config{
		// Database Configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "learning_hub"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// Server Configuration
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerEnv:  getEnv("SERVER_ENV", "development"),

		// JWT Configuration
		JWTSecret: getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
		JWTExpiry: parseDuration(getEnv("JWT_EXPIRY", "24h")),

		// File Upload Configuration
		MaxImageSize:    parseInt64(getEnv("MAX_IMAGE_SIZE", "10485760")),
		MaxVideoSize:    parseInt64(getEnv("MAX_VIDEO_SIZE", "104857600")),
		MaxDocumentSize: parseInt64(getEnv("MAX_DOCUMENT_SIZE", "5242880")),

		// Stripe Configuration
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),

		// Chapa Configuration
		ChapaSecretKey:     getEnv("CHAPA_SECRET_KEY", ""),
		ChapaWebhookSecret: getEnv("CHAPA_WEBHOOK_SECRET", ""),
		AppBaseURL:         getEnv("APP_BASE_URL", "http://localhost:8080"),

		// Firebase Configuration
		FirebaseCredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", ""),
		FirebaseBucketName:      getEnv("FIREBASE_BUCKET_NAME", ""),

		// Email Configuration
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     parseInt(getEnv("SMTP_PORT", "587")),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
	}

	// Validate required fields
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) GetDBDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
}

// IsChapaEnabled checks if Chapa payment is configured
func (c *Config) IsChapaEnabled() bool {
	return c.ChapaSecretKey != ""
}

// IsStripeEnabled checks if Stripe payment is configured
func (c *Config) IsStripeEnabled() bool {
	return c.StripeSecretKey != ""
}

// GetPaymentProvider returns the active payment provider
func (c *Config) GetPaymentProvider() string {
	if c.IsChapaEnabled() {
		return "chapa"
	}
	if c.IsStripeEnabled() {
		return "stripe"
	}
	return "none"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	value, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s, using default: %v", s, err)
		// Return default value based on context
		defaultValue := 0
		if s == "587" {
			defaultValue = 587
		} else if s == "8080" {
			defaultValue = 8080
		}
		return defaultValue
	}
	return value
}

func parseInt64(s string) int64 {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Printf("Warning: Invalid int64 value for %s, using default: %v", s, err)
		defaultValue := int64(0)
		if s == "10485760" {
			defaultValue = 10485760
		} else if s == "104857600" {
			defaultValue = 104857600
		} else if s == "5242880" {
			defaultValue = 5242880
		}
		return defaultValue
	}
	return value
}

func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Warning: Invalid duration value for %s, using default 24h: %v", s, err)
		return 24 * time.Hour
	}
	return duration
}

func validateConfig(config *Config) error {

	// Validate database configuration
	if config.DBHost == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if config.DBPort == "" {
		return fmt.Errorf("DB_PORT is required")
	}
	if config.DBUser == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if config.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	// Validate server configuration
	if config.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}

	// Validate JWT configuration
	if config.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	// Validate file upload sizes
	if config.MaxImageSize <= 0 {
		return fmt.Errorf("MAX_IMAGE_SIZE must be greater than 0")
	}
	if config.MaxVideoSize <= 0 {
		return fmt.Errorf("MAX_VIDEO_SIZE must be greater than 0")
	}
	if config.MaxDocumentSize <= 0 {
		return fmt.Errorf("MAX_DOCUMENT_SIZE must be greater than 0")
	}

	// Validate payment configuration
	if config.IsChapaEnabled() && config.AppBaseURL == "" {
		return fmt.Errorf("APP_BASE_URL is required when using Chapa payments")
	}

	// Validate SMTP configuration if credentials are provided
	if config.SMTPUsername != "" && config.SMTPPassword == "" {
		return fmt.Errorf("SMTP_PASSWORD is required when SMTP_USERNAME is provided")
	}
	if config.SMTPUsername == "" && config.SMTPPassword != "" {
		return fmt.Errorf("SMTP_USERNAME is required when SMTP_PASSWORD is provided")
	}
	if config.SMTPHost != "" {
		if config.SMTPUsername == "" {
			return fmt.Errorf("SMTP_USERNAME is required when SMTP_HOST is provided")
		}
		if config.SMTPPassword == "" {
			return fmt.Errorf("SMTP_PASSWORD is required when SMTP_HOST is provided")
		}
		if config.SMTPPort == 0 {
			return fmt.Errorf("SMTP_PORT is required when SMTP_HOST is provided")
		}
	}

	return nil
}
