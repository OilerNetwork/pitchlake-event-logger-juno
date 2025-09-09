package main

import (
	pluginCore "junoplugin/plugin/core"
	"junoplugin/plugin/listener"
	"log"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	junoplugin "github.com/NethermindEth/juno/plugin"
)

//go:generate go build -buildmode=plugin -o ../../build/plugin.so ./myplugin.go

// pitchlakePlugin is the main plugin struct that implements the JunoPlugin interface
type pitchlakePlugin struct {
	core     *pluginCore.PluginCore
	listener *listener.Service
	log      *log.Logger
}

// Important: "JunoPluginInstance" needs to be exported for Juno to load the plugin correctly
var JunoPluginInstance = pitchlakePlugin{}

// Ensure the plugin and Juno client follow the same interface
var _ junoplugin.JunoPlugin = (*pitchlakePlugin)(nil)

// Init initializes the plugin
func (p *pitchlakePlugin) Init() error {
	p.log = log.Default()
	p.log.Println("Initializing Pitchlake Plugin")

	// Initialize the plugin core
	pluginCoreInstance, err := pluginCore.NewPluginCore()
	if err != nil {
		return err
	}
	p.core = pluginCoreInstance

	// Initialize the plugin
	if err := p.core.Initialize(); err != nil {
		return err
	}

	// Start the vault registry listener
	p.listener = listener.NewService(p.core.GetDB(), p.core.GetVaultManager())
	if err := p.listener.Start(); err != nil {
		return err
	}

	p.log.Println("Pitchlake Plugin initialized successfully")
	return nil
}

// Shutdown shuts down the plugin
func (p *pitchlakePlugin) Shutdown() error {
	p.log.Println("Shutting down Pitchlake Plugin")

	if p.listener != nil {
		p.listener.Stop()
	}

	if p.core != nil {
		return p.core.Shutdown()
	}

	return nil
}

// NewBlock processes a new block
func (p *pitchlakePlugin) NewBlock(
	block *core.Block,
	stateUpdate *core.StateUpdate,
	newClasses map[felt.Felt]core.Class,
) error {
	return p.core.NewBlock(block, stateUpdate, newClasses)
}

// RevertBlock reverts a block
func (p *pitchlakePlugin) RevertBlock(
	from,
	to *junoplugin.BlockAndStateUpdate,
	reverseStateDiff *core.StateDiff,
) error {
	return p.core.RevertBlock(from, to, reverseStateDiff)
}
