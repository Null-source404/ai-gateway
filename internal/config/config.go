// Copyright 2024 John <
// Reads environment values from .env file
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr          string
	Provider      string
	APIKey        string
	OpenAIBaseURL string
	OpenAIAPIKey  string
	Model         string
	RatePerMin    int
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	cfg := Config{
		Addr:          getEnv("GATEWAY_ADDR", ":8080"),
		Provider:      getEnv("GATEWAY_PROVIDER", "mock"),
		APIKey:        getEnv("GATEWAY_API_KEY", "dev-gateway-key-change-me"),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "http://localhost:11434/v1"),
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
		Model:         getEnv("GATEWAY_MODEL", "default"),
		RatePerMin:    getEnvAsInt("GATEWAY_RATE_PER_MIN", 60),
	}
	if cfg.Provider == "" {
		cfg.Provider = "mock"
	}
	if cfg.Model == "" {
		cfg.Model = "default"
	}
	return cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return fmt.Errorf("invalid .env entry on line %d", lineNumber)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s from .env: %w", key, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		n, err := strconv.Atoi(value)
		if err == nil {
			return n
		}
	}
	return fallback
}

func (c Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("GATEWAY_ADDR is required")
	}
	if c.RatePerMin <= 0 {
		c.RatePerMin = 60
	}
	return nil
}
