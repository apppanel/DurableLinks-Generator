package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Server *ServerConfig
	App    *AppConfig
}

func New() *Config {
	return &Config{
		Server: NewServerConfig(),
		App:    NewAppConfig(),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultVal []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.Split(value, ",")
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	if valStr, ok := os.LookupEnv(name); ok {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	if valStr, ok := os.LookupEnv(name); ok {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
