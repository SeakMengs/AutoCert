package config

import (
	"strings"
	"time"

	"github.com/SeakMengs/AutoCert/internal/env"
)

type Config struct {
	Port         string
	ENV          string
	FRONTEND_URL string
	APP          APPConfig
	DB           DatabaseConfig
	RateLimiter  RateLimiterConfig
	Mail         MailConfig
	Auth         AuthConfig
	Minio        MinioConfig
	RabbitMQ     RabbitMQConfig
}

type APPConfig struct {
	MAX_CERTIFICATES_PER_PROJECT int
}

type RateLimiterConfig struct {
	RequestsPerTimeFrame int
	TimeFrame            time.Duration
	Enabled              bool
}

type AuthConfig struct {
	JWT_SECRET        string
	GoogleOAuthConfig GoogleOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type DatabaseConfig struct {
	HOST         string
	PORT         string
	DATABASE     string
	USERNAME     string
	PASSWORD     string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  string
}

type MailConfig struct {
	GMAIL_USERNAME     string
	GMAIL_APP_PASSWORD string
}

type MinioConfig struct {
	ACCESS_KEY string
	SECRET_KEY string
	BUCKET     string
	ENDPOINT   string
	USE_SSL    bool
}

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.ENV, "production")
}

func GetRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		Host:     env.GetString("RABBITMQ_HOST", "127.0.0.1"),
		Port:     env.GetString("RABBITMQ_PORT", "5672"),
		User:     env.GetString("RABBITMQ_USER", "guest"),
		Password: env.GetString("RABBITMQ_PASSWORD", "guest"),
	}
}

func (rmqc RabbitMQConfig) GetConnectionString() string {
	return "amqp://" + rmqc.User + ":" + rmqc.Password + "@" + rmqc.Host + ":" + rmqc.Port + "/"
}

func GetAppConfig() APPConfig {
	return APPConfig{
		MAX_CERTIFICATES_PER_PROJECT: env.GetInt("MAX_CERTIFICATES_PER_PROJECT", 1000),
	}
}

func GetDBConfig() DatabaseConfig {
	return DatabaseConfig{
		HOST:         env.GetString("DB_HOST", "127.0.0.1"),
		PORT:         env.GetString("DB_PORT", "5432"),
		USERNAME:     env.GetString("DB_USERNAME", "root"),
		PASSWORD:     env.GetString("DB_PASSWORD", ""),
		DATABASE:     env.GetString("DB_DATABASE", "database_name"),
		MaxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
		MaxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
		MaxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
	}
}

func GetRateLimiterConfig() RateLimiterConfig {
	rateLimiteTimeFrame, err := time.ParseDuration(env.GetString("RATE_LIMIT_TIME_FRAME", "1m"))
	if err != nil {
		rateLimiteTimeFrame = 60 * time.Second
	}

	return RateLimiterConfig{
		// Default to 5000 requests per minute
		RequestsPerTimeFrame: env.GetInt("RATE_LIMIT_REQUESTS_PER_TIME_FRAME", 5000),
		TimeFrame:            rateLimiteTimeFrame,
		Enabled:              env.GetBool("RATE_LIMIT_ENABLED", true),
	}
}

func GetMailConfig() MailConfig {
	return MailConfig{
		GMAIL_USERNAME:     env.GetString("GMAIL_USERNAME", ""),
		GMAIL_APP_PASSWORD: env.GetString("GMAIL_APP_PASSWORD", ""),
	}
}

func GetMinioConfig() MinioConfig {
	return MinioConfig{
		ACCESS_KEY: env.GetString("MINIO_ACCESS_KEY", ""),
		SECRET_KEY: env.GetString("MINIO_SECRET_KEY", ""),
		BUCKET:     env.GetString("MINIO_BUCKET", "autocert"),
		// If using docker, specify container service name or service host name
		ENDPOINT: env.GetString("MINIO_ENDPOINT", "s3-minio:9000"),
		USE_SSL:  env.GetBool("MINIO_USE_SSL", false),
	}
}

func GetGoogleOAuthConfig() GoogleOAuthConfig {
	return GoogleOAuthConfig{
		ClientID:     env.GetString("GOOGLE_OAUTH_CLIENT_ID", ""),
		ClientSecret: env.GetString("GOOGLE_OAUTH_CLIENT_SECRET", ""),
		RedirectURL:  env.GetString("GOOGLE_OAUTH_CALLBACK", "http://localhost:8080/api/v1/oauth/google/callback"),
	}
}

func GetAuthConfig() AuthConfig {
	return AuthConfig{
		JWT_SECRET:        env.GetString("AUTH_JWT_SECRET", ""),
		GoogleOAuthConfig: GetGoogleOAuthConfig(),
	}
}

func GetEnvironment() string {
	return env.GetString("ENV", "development")
}

func GetConfig() Config {
	return Config{
		Port:         env.GetString("PORT", "8080"),
		ENV:          GetEnvironment(),
		APP:          GetAppConfig(),
		FRONTEND_URL: env.GetString("FRONTEND_URL", "http://localhost:3000"),
		DB:           GetDBConfig(),
		RateLimiter:  GetRateLimiterConfig(),
		Mail:         GetMailConfig(),
		Auth:         GetAuthConfig(),
		Minio:        GetMinioConfig(),
		RabbitMQ:     GetRabbitMQConfig(),
	}
}
