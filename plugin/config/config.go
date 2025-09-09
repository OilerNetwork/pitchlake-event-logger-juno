package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the plugin
type Config struct {
	DatabaseURL string
	RPCURL      string
	UDCAddress  string
	Cursor      uint64
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Required environment variables
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL environment variable is required")
	}
	config.DatabaseURL = dbURL

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("RPC_URL environment variable is required")
	}
	config.RPCURL = rpcURL

	// Optional environment variables
	config.UDCAddress = os.Getenv("UDC_ADDRESS")

	cursor := os.Getenv("CURSOR")
	if cursor != "" {
		var err error
		config.Cursor, err = strconv.ParseUint(cursor, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid CURSOR value: %w", err)
		}
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if c.RPCURL == "" {
		return fmt.Errorf("RPC URL is required")
	}
	return nil
}
