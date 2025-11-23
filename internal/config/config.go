// Package config отвечает за загрузку конфигурации приложения
package config

import (
	"fmt"
	"os"
)

// Config содержит конфигурацию всего приложения
type Config struct {
	DB DBConfig
}

// DBConfig содержит параметры подключения к базе данных PostgreSQL
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	return &Config{
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "pr_reviewer"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
}

// ConnString возвращает строку подключения к PostgreSQL
func (c DBConfig) ConnString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
