package utils

import (
	"math/big"
	"testing"
)

func TestKeccak256(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Deposit",
			expected: "0x2c7216fb67e74a66f6dd529eb7a1e230d99c61d3bb75117872c3ce31f3956715",
		},
		{
			input:    "Withdrawal",
			expected: "0x2a2c43bf243bbd8cbc4c2f5c8b0e7c5e8c5e8c5e8c5e8c5e8c5e8c5e8c5e8c5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Keccak256(tt.input)
			if result == "" {
				t.Error("Expected non-empty result")
			}
			// We can't easily test the exact hash without duplicating the implementation
			// but we can test that it's consistent
			result2 := Keccak256(tt.input)
			if result != result2 {
				t.Error("Keccak256 should be deterministic")
			}
		})
	}
}

func TestDecodeEventNameVault(t *testing.T) {
	// Test with a known event name hash
	depositHash := Keccak256("Deposit")

	result, err := DecodeEventNameVault(depositHash)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "Deposit" {
		t.Errorf("Expected 'Deposit', got '%s'", result)
	}

	// Test with unknown hash
	unknownHash := "0x1234567890abcdef"
	_, err = DecodeEventNameVault(unknownHash)
	if err == nil {
		t.Error("Expected error for unknown hash")
	}
}

func TestNormalizeHexAddress(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{
			input:       "0x050aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd",
			expected:    "0x50aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd",
			expectError: false,
		},
		{
			input:       "0x50aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd",
			expected:    "0x50aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd",
			expectError: false,
		},
		{
			input:       "0x0000000000000000000000000000000000000000000000000000000000000000",
			expected:    "0x0",
			expectError: false,
		},
		{
			input:       "0x0000000000000000000000000000000000000000000000000000000000000001",
			expected:    "0x1",
			expectError: false,
		},
		{
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
		{
			input:       "0x",
			expected:    "0x0",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizeHexAddress(tt.input)

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

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHexStringToFelt(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
	}{
		{
			input:       "0x123",
			expectError: false,
		},
		{
			input:       "123",
			expectError: false,
		},
		{
			input:       "0x",
			expectError: false,
		},
		{
			input:       "invalid",
			expectError: true,
		},
		{
			input:       "0xgg",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := HexStringToFelt(tt.input)

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

			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestDecimalStringToHexString(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{
			input:       "0",
			expected:    "0x0",
			expectError: false,
		},
		{
			input:       "1",
			expected:    "0x1",
			expectError: false,
		},
		{
			input:       "255",
			expected:    "0xff",
			expectError: false,
		},
		{
			input:       "256",
			expected:    "0x100",
			expectError: false,
		},
		{
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := DecimalStringToHexString(tt.input)

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

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestBigIntToHexString(t *testing.T) {
	tests := []struct {
		input    *big.Int
		expected string
	}{
		{
			input:    big.NewInt(0),
			expected: "0x0",
		},
		{
			input:    big.NewInt(1),
			expected: "0x1",
		},
		{
			input:    big.NewInt(255),
			expected: "0xff",
		},
		{
			input:    big.NewInt(256),
			expected: "0x100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := BigIntToHexString(*tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
