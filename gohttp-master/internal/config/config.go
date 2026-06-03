package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config holds all server configuration
type Config struct {
	Server  ServerConfig  `json:"server"`
	Limits  LimitsConfig  `json:"limits"`
	Logging LoggingConfig `json:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
	IdleTimeout  int    `json:"idle_timeout_seconds"`
}

// LimitsConfig holds rate limiting and resource configuration
type LimitsConfig struct {
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
	MaxRequestBodySize    int `json:"max_request_body_size_bytes"`
	RequestTimeoutSeconds int `json:"request_timeout_seconds"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Enabled     bool   `json:"enabled"`
	Level       string `json:"level"`
	ColorOutput bool   `json:"color_output"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		Limits: LimitsConfig{
			MaxConcurrentRequests: 100,
			MaxRequestBodySize:    1048576, // 1MB
			RequestTimeoutSeconds: 30,
		},
		Logging: LoggingConfig{
			Enabled:     true,
			Level:       "info",
			ColorOutput: true,
		},
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Load from file if it exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			file, err := os.Open(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open config file: %v", err)
			}
			defer file.Close()

			decoder := json.NewDecoder(file)
			if err := decoder.Decode(config); err != nil {
				return nil, fmt.Errorf("failed to decode config file: %v", err)
			}
		}
	}

	// Override with environment variables
	if port := os.Getenv("GOHTTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if host := os.Getenv("GOHTTP_HOST"); host != "" {
		config.Server.Host = host
	}

	if maxConn := os.Getenv("GOHTTP_MAX_CONNECTIONS"); maxConn != "" {
		if mc, err := strconv.Atoi(maxConn); err == nil {
			config.Limits.MaxConcurrentRequests = mc
		}
	}

	if logLevel := os.Getenv("GOHTTP_LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}

	return config, nil
}

// SaveConfig saves the current configuration to a file
func (c *Config) SaveConfig(configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %v", err)
	}

	return nil
}

// Address returns the full server address
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	if c.Limits.MaxConcurrentRequests < 1 {
		return fmt.Errorf("invalid max concurrent requests: %d (must be > 0)", c.Limits.MaxConcurrentRequests)
	}

	if c.Limits.MaxRequestBodySize < 1 {
		return fmt.Errorf("invalid max request body size: %d (must be > 0)", c.Limits.MaxRequestBodySize)
	}

	return nil
}
