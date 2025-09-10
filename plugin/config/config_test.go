package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment variables
	originalDBURL := os.Getenv("DB_URL")
	originalRPCURL := os.Getenv("RPC_URL")
	originalUDCAddress := os.Getenv("UDC_ADDRESS")
	originalCursor := os.Getenv("CURSOR")

	// Clean up after test
	defer func() {
		os.Setenv("DB_URL", originalDBURL)
		os.Setenv("RPC_URL", originalRPCURL)
		os.Setenv("UDC_ADDRESS", originalUDCAddress)
		os.Setenv("CURSOR", originalCursor)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		expected    *Config
	}{
		{
			name: "valid config with all variables",
			envVars: map[string]string{
				"DB_URL":      "postgres://localhost:5432/test",
				"RPC_URL":     "https://starknet-mainnet.infura.io",
				"UDC_ADDRESS": "0x123",
				"CURSOR":      "1000",
			},
			expectError: false,
			expected: &Config{
				DatabaseURL: "postgres://localhost:5432/test",
				RPCURL:      "https://starknet-mainnet.infura.io",
				UDCAddress:  "0x123",
				Cursor:      1000,
			},
		},
		{
			name: "valid config with required variables only",
			envVars: map[string]string{
				"DB_URL":  "postgres://localhost:5432/test",
				"RPC_URL": "https://starknet-mainnet.infura.io",
			},
			expectError: false,
			expected: &Config{
				DatabaseURL: "postgres://localhost:5432/test",
				RPCURL:      "https://starknet-mainnet.infura.io",
				UDCAddress:  "",
				Cursor:      0,
			},
		},
		{
			name: "missing DB_URL",
			envVars: map[string]string{
				"RPC_URL": "https://starknet-mainnet.infura.io",
			},
			expectError: true,
		},
		{
			name: "missing RPC_URL",
			envVars: map[string]string{
				"DB_URL": "postgres://localhost:5432/test",
			},
			expectError: true,
		},
		{
			name: "invalid CURSOR value",
			envVars: map[string]string{
				"DB_URL":  "postgres://localhost:5432/test",
				"RPC_URL": "https://starknet-mainnet.infura.io",
				"CURSOR":  "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("DB_URL")
			os.Unsetenv("RPC_URL")
			os.Unsetenv("UDC_ADDRESS")
			os.Unsetenv("CURSOR")

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config, err := LoadConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config.DatabaseURL != tt.expected.DatabaseURL {
				t.Errorf("Expected DatabaseURL %s, got %s", tt.expected.DatabaseURL, config.DatabaseURL)
			}

			if config.RPCURL != tt.expected.RPCURL {
				t.Errorf("Expected RPCURL %s, got %s", tt.expected.RPCURL, config.RPCURL)
			}

			if config.UDCAddress != tt.expected.UDCAddress {
				t.Errorf("Expected UDCAddress %s, got %s", tt.expected.UDCAddress, config.UDCAddress)
			}

			if config.Cursor != tt.expected.Cursor {
				t.Errorf("Expected Cursor %d, got %d", tt.expected.Cursor, config.Cursor)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				DatabaseURL: "postgres://localhost:5432/test",
				RPCURL:      "https://starknet-mainnet.infura.io",
			},
			expectError: false,
		},
		{
			name: "missing database URL",
			config: &Config{
				RPCURL: "https://starknet-mainnet.infura.io",
			},
			expectError: true,
		},
		{
			name: "missing RPC URL",
			config: &Config{
				DatabaseURL: "postgres://localhost:5432/test",
			},
			expectError: true,
		},
		{
			name:        "both URLs missing",
			config:      &Config{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
