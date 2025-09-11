package block

import (
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/network"
	"junoplugin/plugin/vault"
	"log"
	"sync"
	"time"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	junoplugin "github.com/NethermindEth/juno/plugin"
)

// Processor handles block processing logic
type Processor struct {
	db                *db.DB
	network           *network.Network
	vaultManager      *vault.Manager
	lastBlockDB       *models.StarknetBlocks
	cursor            uint64
	mu                sync.Mutex
	log               *log.Logger
	driverEventChan   chan models.DriverEvent
	vaultCatchupChan  chan models.VaultCatchupEvent
}

// NewProcessor creates a new block processor
func NewProcessor(
	db *db.DB,
	network *network.Network,
	vaultManager *vault.Manager,
	lastBlockDB *models.StarknetBlocks,
	cursor uint64,
) *Processor {
	return &Processor{
		db:               db,
		network:          network,
		vaultManager:     vaultManager,
		lastBlockDB:      lastBlockDB,
		cursor:           cursor,
		log:              log.Default(),
		driverEventChan:  make(chan models.DriverEvent, 100), // Buffered channel
		vaultCatchupChan: make(chan models.VaultCatchupEvent, 100), // Buffered channel
	}
}

// ProcessNewBlock processes a new block
func (bp *Processor) ProcessNewBlock(
	block *core.Block,
	stateUpdate *core.StateUpdate,
	newClasses map[felt.Felt]core.Class,
) error {
	if block.Number < bp.cursor {
		return nil
	}

	bp.mu.Lock()
	defer bp.mu.Unlock()
	// Check if we need to catch up

	bp.db.BeginTx()
	bp.log.Println("Processing new block", block.Number)

	// Process events in the block
	err := bp.processBlockEvents(block)
	if err != nil {
		bp.db.RollbackTx()
		bp.log.Println("Error processing block events", err)
		return err
	}

	// Store the block
	starknetBlock := models.CoreToStarknetBlock(*block)

	err = bp.db.InsertBlock(&starknetBlock)
	if err != nil {
		bp.db.RollbackTx()
		bp.log.Println("Error inserting block", err)
		return err
	}

	bp.lastBlockDB = &starknetBlock
	
	// Send StartBlock event right before commit
	bp.sendDriverEvent("StartBlock", block.Number, block.Hash.String())
	bp.db.CommitTx()

	return nil
}

// RevertBlock reverts a block
func (bp *Processor) RevertBlock(
	from,
	to *junoplugin.BlockAndStateUpdate,
	reverseStateDiff *core.StateDiff,
) error {
	// FIXED: Add proper transaction handling for revert
	bp.db.BeginTx()
	
	err := bp.db.RevertBlock(from.Block.Number, from.Block.Hash.String())
	if err != nil {
		bp.db.RollbackTx()
		return err
	}

	// TODO: Implement vault event reversion if needed
	// This was commented out in the original code

	// Send RevertBlock event right before commit
	bp.sendDriverEvent("RevertBlock", from.Block.Number, from.Block.Hash.String())
	bp.db.CommitTx()

	return nil
}

func (bp *Processor) CatchupBlocks(latestBlock uint64) error {

	//Leaving this as a potential usage,  we use this to decide how much block data we wanna back fill (in case of a very edge case of clean starting block getting reorged)
	backFillIndex := uint64(3)
	startBlock := latestBlock - uint64(backFillIndex)
	if bp.lastBlockDB != nil {
		startBlock = bp.lastBlockDB.BlockNumber
	}

	for startBlock < latestBlock-1 {
		endBlock := startBlock + 1000
		if endBlock >= latestBlock {
			endBlock = latestBlock - 1
		}

		bp.log.Println("Catching up indexer from", startBlock, "to", endBlock)
		blocks, err := bp.network.GetBlocks(startBlock, endBlock)
		if err != nil {
			bp.log.Println("Error getting blocks", err)
			return err
		}

		for _, block := range blocks {

			//Should create a seperate version of insert block, the ideal solution is better refactor to be able to use tx more easily
			bp.db.BeginTx()
			err := bp.db.InsertBlock(block)
			if err != nil {
				bp.db.RollbackTx()
				bp.log.Println("Error inserting block", err)
				return err
			}
			// Send CatchupBlock event for each individual block
			bp.sendDriverEvent("CatchupBlock", block.BlockNumber, block.BlockHash)
			bp.db.CommitTx()
		}
		startBlock = endBlock
	}
	return nil
}

// GetLastBlock returns the last processed block
func (bp *Processor) GetLastBlock() *models.StarknetBlocks {
	return bp.lastBlockDB
}

// UpdateLastBlock updates the last processed block
func (bp *Processor) UpdateLastBlock(block *models.StarknetBlocks) {
	bp.lastBlockDB = block
}

// processBlockEvents processes all events in a block
func (bp *Processor) processBlockEvents(block *core.Block) error {
	bp.log.Println("Processing block events for block", block.Number)

	for _, receipt := range block.Receipts {
		for _, event := range receipt.Events {
			fromAddress := event.From.String()
			if bp.vaultManager.IsVaultAddress(fromAddress) {
				err := bp.vaultManager.ProcessVaultEvent(receipt.TransactionHash.String(), fromAddress, event, block.Number, *block.Hash)
				if err != nil {
					bp.log.Println("Error processing vault event", err)
					return err
				}
			}
		}
	}

	return nil
}

// sendDriverEvent sends a driver event through the channel and stores it in the database
func (bp *Processor) sendDriverEvent(eventType string, blockNumber uint64, blockHash string) {
	// Store in database for auditing
	err := bp.db.StoreDriverEvent(eventType, blockNumber, blockHash)
	if err != nil {
		bp.log.Printf("Error storing driver event: %v", err)
	}
	
	// Send through channel
	event := models.DriverEvent{
		Type:        eventType,
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		Timestamp:   time.Now(),
	}
	
	// Wait up to 5 seconds for channel space
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	
	select {
	case bp.driverEventChan <- event:
		bp.log.Printf("Sent driver event: %s for block %d", eventType, blockNumber)
	case <-timeout.C:
		bp.log.Printf("ERROR: Timeout waiting to send driver event: %s for block %d - this is a critical failure!", eventType, blockNumber)
		// Note: Event is still stored in DB, but event-processor won't be notified
	}
}

// SendVaultCatchupEvent sends a vault catchup event through the channel and stores it in the database
func (bp *Processor) SendVaultCatchupEvent(vaultAddress string, startBlock, endBlock uint64) {
	// Store in database for auditing
	err := bp.db.StoreVaultCatchupEvent(vaultAddress, startBlock, endBlock)
	if err != nil {
		bp.log.Printf("Error storing vault catchup event: %v", err)
	}
	
	// Send through channel
	event := models.VaultCatchupEvent{
		Type:         "VaultCatchup",
		VaultAddress: vaultAddress,
		StartBlock:   startBlock,
		EndBlock:     endBlock,
		Timestamp:    time.Now(),
	}
	
	// Wait up to 5 seconds for channel space
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	
	select {
	case bp.vaultCatchupChan <- event:
		bp.log.Printf("Sent vault catchup event for vault %s, blocks %d-%d", vaultAddress, startBlock, endBlock)
	case <-timeout.C:
		bp.log.Printf("ERROR: Timeout waiting to send vault catchup event for vault %s - this is a critical failure!", vaultAddress)
		// Note: Event is still stored in DB, but event-processor won't be notified
	}
}

// GetDriverEventChannel returns the driver event channel for external listeners
func (bp *Processor) GetDriverEventChannel() <-chan models.DriverEvent {
	return bp.driverEventChan
}

// GetVaultCatchupEventChannel returns the vault catchup event channel for external listeners
func (bp *Processor) GetVaultCatchupEventChannel() <-chan models.VaultCatchupEvent {
	return bp.vaultCatchupChan
}
