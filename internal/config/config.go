package config

import (
	"sync"
)

var (
	globalConfig Config
	initOnce     sync.Once
)

type Config struct {
	Server   ServerConfig   `json:"server" envPrefix:"SERVER_" validate:"required"`
	Database DatabaseConfig `json:"database" envPrefix:"DB_" validate:"required"`
	Redis    RedisConfig    `json:"redis" envPrefix:"REDIS_" validate:"required"`
	JWT      JWTConfig      `json:"jwt" envPrefix:"JWT_" validate:"required"`
	Webhook  WebhookConfig  `json:"webhook" envPrefix:"WEBHOOK_" validate:"required"`
}

type ServerConfig struct {
	Port         string   `json:"port" env:"PORT" validate:"required,numeric"`
	Host         string   `json:"host" env:"HOST" validate:"required,hostname|ip"`
	ReadTimeout  Duration `json:"read_timeout" env:"READ_TIMEOUT" validate:"required,duration_gt0"`
	WriteTimeout Duration `json:"write_timeout" env:"WRITE_TIMEOUT" validate:"required,duration_gt0"`
}

type DatabaseConfig struct {
	Host     string `json:"host" env:"HOST" validate:"required,hostname|ip"`
	Port     string `json:"port" env:"PORT" validate:"required,numeric"`
	User     string `json:"user" env:"USER" validate:"required"`
	Password string `json:"password" env:"PASSWORD" validate:"required"`
	DBName   string `json:"db_name" env:"NAME" validate:"required"`
	SSLMode  string `json:"ssl_mode" env:"SSL_MODE" validate:"required,oneof=disable require verify-ca verify-full"`
}

type RedisConfig struct {
	Addr     string   `json:"addr" env:"REDIS_ADDR" validate:"required,hostname_port"`
	Password string   `json:"password" env:"REDIS_PASSWORD" validate:"omitempty"`
	DB       int      `json:"db" env:"REDIS_DB" validate:"gte=0"`
	TTL      Duration `json:"ttl" env:"REDIS_TTL" validate:"required,duration_gt0"`
}

type JWTConfig struct {
	AccessTokenTTL  Duration `json:"access_token_ttl" env:"ACCESS_TOKEN_TTL" validate:"required,duration_gt0"`
	RefreshTokenTTL Duration `json:"refresh_token_ttl" env:"REFRESH_TOKEN_TTL" validate:"required,duration_gt0"`
	SecretKey       string   `json:"secret_key" env:"SECRET_KEY" validate:"required"`
}

type WebhookConfig struct {
	URL     string   `json:"url" env:"URL" validate:"omitempty,url"`
	Timeout Duration `json:"timeout" env:"TIMEOUT" validate:"required,duration_gt0"`
}
