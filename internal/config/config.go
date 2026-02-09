package config

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	AppName    string
	AppEnv     string
	Port       string
	LogLevel   string
	LogFormat  string
	Database   DatabaseConfig
	ApiVersion string
	ApiTimeout int
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MinConns int
}

var AppConfig *Config

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = viper.GetString("PORT")
	}
	if port == "" {
		port = "8080"
	}

	viper.SetDefault("APP_NAME", "api-limiter")
	viper.SetDefault("APP_ENV", "development")

	// Database defaults
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "accounts_db")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_CONNS", 25)
	viper.SetDefault("DB_MIN_CONNS", 5)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "text")
	viper.SetDefault("API_VERSION", "v1")
	viper.SetDefault("API_TIMEOUT", 30)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("No .env file found, using environment variables and defaults")
		} else {
			log.Printf("Error reading config file: %v", err)
		}
	}

	config := &Config{
		AppName:    viper.GetString("APP_NAME"),
		AppEnv:     viper.GetString("APP_ENV"),
		Port:       port,
		LogLevel:   viper.GetString("LOG_LEVEL"),
		LogFormat:  viper.GetString("LOG_FORMAT"),
		ApiVersion: viper.GetString("API_VERSION"),
		ApiTimeout: viper.GetInt("API_TIMEOUT"),
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
			MaxConns: viper.GetInt("DB_MAX_CONNS"),
			MinConns: viper.GetInt("DB_MIN_CONNS"),
		},
	}

	if config.Database.Host == "" || config.Database.User == "" {
		return nil, fmt.Errorf("database configuration is incomplete")
	}

	AppConfig = config
	return config, nil
}

func GetConfig() *Config {
	if AppConfig == nil {
		log.Fatal("Config not initialized. Call LoadConfig() first.")
	}
	return AppConfig
}
