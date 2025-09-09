# Pitchlake Plugin

This is a refactored version of the Pitchlake plugin for Juno, organized into smaller, focused components for better maintainability and testability.

## Structure

The plugin is now organized into the following packages:

### Core Components

- **`config/`** - Configuration management
  - `config.go` - Handles loading and validation of environment variables

- **`vault/`** - Vault management
  - `vault_manager.go` - Handles vault initialization, catchup, and event processing

- **`event/`** - Event processing
  - `event_processor.go` - Processes events from blocks

- **`block/`** - Block processing
  - `block_processor.go` - Handles block processing and catchup logic

- **`listener/`** - Vault registry listener
  - `listener.go` - Listens for new vault registrations

- **`core/`** - Plugin orchestration
  - `plugin_core.go` - Main orchestrator that coordinates all components

### Main Files

- **`myplugin.go`** - Main plugin file that implements the JunoPlugin interface

## Key Improvements

1. **Separation of Concerns**: Each package has a single responsibility
2. **Dependency Injection**: Components are injected rather than tightly coupled
3. **Testability**: Each component can be tested independently
4. **Maintainability**: Smaller, focused files are easier to understand and modify
5. **Configuration Management**: Centralized configuration handling
6. **Error Handling**: Better error propagation and handling

## Usage

The plugin follows the same interface as before, but internally uses the new modular structure. The main entry point is still `JunoPluginInstance` which implements the `JunoPlugin` interface.

## Environment Variables

- `DB_URL` - Database connection URL (required)
- `RPC_URL` - StarkNet RPC URL (required)
- `UDC_ADDRESS` - Universal Deployer Contract address (optional)
- `CURSOR` - Starting block number for indexing (optional)

## Building

```bash
go build -buildmode=plugin -o ../../build/plugin.so ./myplugin.go
```