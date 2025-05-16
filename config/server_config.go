package config

import (
	"time"
)

type ServerConfig struct {
	Port            string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	DBDriver        string
	DBConnectionStr string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:            getEnv("PORT", "9010"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DBDriver:        getEnv("DB_DRIVER", "postgres"),
		DBConnectionStr: getEnv("DATABASE_URL", ""),
		ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
	}
}
