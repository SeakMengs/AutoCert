package config

import (
	"strings"
	"time"

	"github.com/SeakMengs/AutoCert/internal/env"
)

type Config struct {
	Port        string
	ENV         string
	DB          DatabaseConfig
	RateLimiter RateLimiterConfig
	Mail        MailConfig
	Auth        AuthConfig
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
	DB_HOST      string
	DB_PORT      string
	DB_DATABASE  string
	DB_USERNAME  string
	DB_PASSWORD  string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  string
}

type MailConfig struct {
	SEND_GRID  SendGridConfig
	FROM_EMAIL string
}

type SendGridConfig struct {
	API_KEY string
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.ENV, "production")
}

func GetConfig() Config {
	rateLimiteTimeFrame, err := time.ParseDuration(env.GetString("RATE_LIMIT_TIME_FRAME", "1m"))
	if err != nil {
		rateLimiteTimeFrame = 60 * time.Second
	}

	return Config{
		Port: env.GetString("PORT", "8080"),
		ENV:  env.GetString("ENV", "development"),
		DB: DatabaseConfig{
			DB_HOST:      env.GetString("DB_HOST", "127.0.0.1"),
			DB_PORT:      env.GetString("DB_PORT", "5432"),
			DB_USERNAME:  env.GetString("DB_USERNAME", "root"),
			DB_PASSWORD:  env.GetString("DB_PASSWORD", ""),
			DB_DATABASE:  env.GetString("DB_DATABASE", "database_name"),
			MaxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			MaxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			MaxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		// By default if not specified, we allow 5000 requests per minute on all routes
		RateLimiter: RateLimiterConfig{
			RequestsPerTimeFrame: env.GetInt("RATE_LIMIT_REQUESTS_PER_TIME_FRAME", 5000),
			TimeFrame:            rateLimiteTimeFrame,
			Enabled:              env.GetBool("RATE_LIMIT_ENABLED", true),
		},
		Mail: MailConfig{
			FROM_EMAIL: env.GetString("MAIL_FROM_MAIL", ""),
			SEND_GRID: SendGridConfig{
				API_KEY: env.GetString("MAIL_SEND_GRID_API_KEY", ""),
			},
		},
		Auth: AuthConfig{
			JWT_SECRET: env.GetString("AUTH_JWT_SECRET", ""),
			GoogleOAuthConfig: GoogleOAuthConfig{
				ClientID:     env.GetString("GOOGLE_OAUTH_CLIENT_ID", ""),
				ClientSecret: env.GetString("GOOGLE_OAUTH_CLIENT_SECRET", ""),
				RedirectURL:  env.GetString("GOOGLE_OAUTH_CALLBACK", "http://localhost:8080/api/v1/oauth/google/callback"),
			},
		},
	}
}
