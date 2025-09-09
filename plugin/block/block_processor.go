package block

import (
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/network"
	"junoplugin/plugin/vault"
	"log"
	"sync"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	junoplugin "github.com/NethermindEth/juno/plugin"
)

// Processor handles block processing logic
type Processor struct {
	db           *db.DB
	network      *network.Network
	vaultManager *vault.Manager
	lastBlockDB  *models.StarknetBlocks
	cursor       uint64
	mu           sync.Mutex
	log          *log.Logger
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
		db:           db,
		network:      network,
		vaultManager: vaultManager,
		lastBlockDB:  lastBlockDB,
		cursor:       cursor,
		log:          log.Default(),
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
	if bp.lastBlockDB != nil && bp.lastBlockDB.BlockNumber < block.Number-1 {
		err := bp.CatchupIndexer(block.Number)
		if err != nil {
			bp.log.Println("Error catching up indexer", err)
			return err
		}
	}

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
	starknetBlock := models.StarknetBlocks{
		BlockNumber: block.Number,
		BlockHash:   block.Hash.String(),
		ParentHash:  block.ParentHash.String(),
		Timestamp:   block.Timestamp,
		Status:      "MINED",
	}

	err = bp.db.InsertBlock(&starknetBlock)
	if err != nil {
		bp.db.RollbackTx()
		bp.log.Println("Error inserting block", err)
		return err
	}

	bp.lastBlockDB = &starknetBlock
	bp.db.CommitTx()

	return nil
}

// RevertBlock reverts a block
func (bp *Processor) RevertBlock(
	from,
	to *junoplugin.BlockAndStateUpdate,
	reverseStateDiff *core.StateDiff,
) error {
	err := bp.db.RevertBlock(from.Block.Number, from.Block.Hash.String())
	if err != nil {
		return err
	}

	// TODO: Implement vault event reversion if needed
	// This was commented out in the original code

	return nil
}

// CatchupIndexer catches up the indexer to a specific block
func (bp *Processor) CatchupIndexer(latestBlock uint64) error {
	startBlock := bp.lastBlockDB.BlockNumber
	for startBlock < latestBlock {
		endBlock := startBlock + 1000
		if endBlock > latestBlock {
			endBlock = latestBlock
		}

		// Catch up all vaults
		for vaultAddr := range bp.vaultManager.GetVaultAddresses() {
			err := bp.vaultManager.CatchupVault(vaultAddr, endBlock)
			if err != nil {
				bp.log.Println("Error catching up vault", err)
				return err
			}
		}

		bp.log.Println("Catching up indexer from", startBlock, "to", endBlock)
		blocks, err := bp.network.GetBlocks(startBlock, endBlock)
		if err != nil {
			bp.log.Println("Error getting blocks", err)
			return err
		}

		for _, block := range blocks {
			err := bp.db.InsertBlock(block)
			if err != nil {
				bp.log.Println("Error inserting block", err)
				return err
			}
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
				err := bp.vaultManager.ProcessVaultEvent(receipt.TransactionHash.String(), fromAddress, event, block.Number)
				if err != nil {
					bp.log.Println("Error processing vault event", err)
					return err
				}
			}
		}
	}

	return nil
}
