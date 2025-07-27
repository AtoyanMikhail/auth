package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"log"

	"github.com/caarlos0/env/v10"
	"github.com/go-playground/validator/v10"
)

// GetConfig sets default values to the Config struct, then tries to override them with a .json config file (the path is stored in the CONFIG_PATH environment variable),
// and finally overrides values from environment variables on the first usage. Then, it returns a pointer to the global config instance.
func GetConfig() (*Config, error) {
	initOnce.Do(func() {
		setDefaults(&globalConfig)

		// Overriding values from json if it is possible
		if err := loadFromJSON(&globalConfig); err != nil {
			log.Printf("failed to load config from JSON: %s\n", err.Error())
		}

		// Overriding values from env
		loadFromEnv(&globalConfig)

		if err := validate(&globalConfig); err != nil {
			log.Fatalf("config validation failed: %s", err.Error())
		}
	})

	return &globalConfig, nil
}

func setDefaults(cfg *Config) {
	cfg.Server = ServerConfig{
		Port:         "8080",
		Host:         "0.0.0.0",
		ReadTimeout:  Duration(30 * time.Second),
		WriteTimeout: Duration(30 * time.Second),
	}

	cfg.Database = DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "jwt",
		SSLMode:  "disable",
	}

	cfg.Redis = RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		TTL:      Duration(10 * time.Minute),
	}

	cfg.JWT = JWTConfig{
		AccessTokenTTL:  Duration(15 * time.Minute),
		RefreshTokenTTL: Duration(7 * 24 * time.Hour),
		SecretKey:       "secret_key",
	}

	cfg.Webhook = WebhookConfig{
		URL:     "",
		Timeout: Duration(5 * time.Second),
	}
}

func loadFromJSON(cfg *Config) error {
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(cfg)
}

// loadFromEnv unmarshalles env variables for config from enviroment
func loadFromEnv(cfg *Config) {
	_ = env.Parse(cfg)
}

// getConfigPaths reads path to .json config from CONFIG_PATH env variable
func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return filepath.Join("config", "config.json")
}

func validate(cfg *Config) error {
	validate := validator.New()

	// Custom validation for Duration type: must be greater than 0
	validate.RegisterValidation("duration_gt0", func(fl validator.FieldLevel) bool {
		d, ok := fl.Field().Interface().(Duration)
		return ok && d > 0
	})

	return validate.Struct(cfg)
}
