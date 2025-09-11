package core

import (
	"fmt"
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/network"
	"junoplugin/plugin/block"
	"junoplugin/plugin/config"
	"junoplugin/plugin/events"
	"junoplugin/plugin/vault"
	"log"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	junoplugin "github.com/NethermindEth/juno/plugin"
)

// PluginCore orchestrates all plugin components
type PluginCore struct {
	config         *config.Config
	db             *db.DB
	network        *network.Network
	vaultManager   *vault.Manager
	blockProcessor *block.Processor
	log            *log.Logger
	synced         bool
}

// NewPluginCore creates a new plugin core
func NewPluginCore() (*PluginCore, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize database
	dbClient, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Get last block from database
	lastBlockDB, err := dbClient.GetLastBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get last block: %w", err)
	}

	// Initialize network
	networkClient, err := network.NewNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}

	// Initialize vault manager
	vaultManager := vault.NewManager(dbClient, networkClient, cfg.UDCAddress)

	// Initialize block processor
	blockProcessor := block.NewProcessor(
		dbClient,
		networkClient,
		vaultManager,
		lastBlockDB,
		cfg.Cursor,
	)

	// Wire up vault catchup events
	eventSender := events.NewVaultCatchupEventSender(blockProcessor.SendVaultCatchupEvent)
	vaultManager.SetEventSender(eventSender)

	return &PluginCore{
		config:         cfg,
		db:             dbClient,
		network:        networkClient,
		vaultManager:   vaultManager,
		blockProcessor: blockProcessor,
		log:            log.Default(),
	}, nil
}

// Initialize initializes the plugin
func (pc *PluginCore) Initialize() error {
	return nil
}

// Shutdown shuts down the plugin
func (pc *PluginCore) Shutdown() error {
	pc.log.Println("Shutting down plugin core")
	pc.db.Shutdown()
	return nil
}

func (pc *PluginCore) CheckAndSync(block *models.StarknetBlocks) error {

	if pc.synced == true {
		return nil
	}

	log.Printf("Syncing vaults")
	if err := pc.vaultManager.LoadVaultsFromRegistry(block); err != nil {
		return fmt.Errorf("failed to initialize vaults: %w", err)
	}

	pc.log.Println("Plugin core initialized successfully")

	//Only set this to true if no failures
	pc.synced = true
	return nil
}

// NewBlock processes a new block
func (pc *PluginCore) NewBlock(
	block *core.Block,
	stateUpdate *core.StateUpdate,
	newClasses map[felt.Felt]core.Class,
) error {
	starknetBlock := models.CoreToStarknetBlock(*block)
	if err := pc.CheckAndSync(&starknetBlock); err != nil {
		return err
	}
	return pc.blockProcessor.ProcessNewBlock(block, stateUpdate, newClasses)
}

// RevertBlock reverts a block
func (pc *PluginCore) RevertBlock(
	from,
	to *junoplugin.BlockAndStateUpdate,
	reverseStateDiff *core.StateDiff,
) error {

	starknetBlock := models.CoreToStarknetBlock(*from.Block)
	if err := pc.CheckAndSync(&starknetBlock); err != nil {
		return err
	}
	return pc.blockProcessor.RevertBlock(from, to, reverseStateDiff)
}

// GetVaultManager returns the vault manager
func (pc *PluginCore) GetVaultManager() *vault.Manager {
	return pc.vaultManager
}

// GetDB returns the database instance
func (pc *PluginCore) GetDB() *db.DB {
	return pc.db
}
