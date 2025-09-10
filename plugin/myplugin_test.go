package main

import (
	"os"
	"testing"

	junoplugin "github.com/NethermindEth/juno/plugin"
)

func TestMain(m *testing.M) {
	// Set up test environment variables
	os.Setenv("DB_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("RPC_URL", "https://starknet-mainnet.infura.io")
	os.Setenv("UDC_ADDRESS", "0x123")
	os.Setenv("CURSOR", "1000")

	// Run tests
	code := m.Run()

	// Clean up
	os.Unsetenv("DB_URL")
	os.Unsetenv("RPC_URL")
	os.Unsetenv("UDC_ADDRESS")
	os.Unsetenv("CURSOR")

	os.Exit(code)
}

func TestJunoPluginInstance(t *testing.T) {
	// Test that the plugin instance is properly initialized
	if JunoPluginInstance.core != nil {
		t.Error("Expected plugin core to be nil before initialization")
	}

	if JunoPluginInstance.listener != nil {
		t.Error("Expected listener to be nil before initialization")
	}

	if JunoPluginInstance.log != nil {
		t.Error("Expected logger to be nil before initialization")
	}
}

func TestPluginInterface(t *testing.T) {
	// Test that the plugin implements the required interface
	var _ junoplugin.JunoPlugin = (*pitchlakePlugin)(nil)
	// This will fail at compile time if the interface is not implemented
}

// Note: We don't test Init() here because it requires actual database and network connections
// which would make this an integration test rather than a unit test.
// For unit testing, we would need to refactor the code to use dependency injection
// and interfaces, which would be a significant architectural change.
