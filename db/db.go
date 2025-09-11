package db

import (
	"context"
	"errors"
	"junoplugin/models"
	"log"

	"github.com/jackc/pgx/v5"
)

func (db *DB) GetVaultRegistry() ([]*models.VaultRegistry, error) {
	var vaultRegistry []*models.VaultRegistry
	query := `
	SELECT 
		vault_address, 
		deployed_at, 
		last_block_indexed, 
		last_block_processed 
	FROM vault_registry`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var vault models.VaultRegistry
		if err := rows.Scan(&vault.Address, &vault.DeployedAt, &vault.LastBlockIndexed, &vault.LastBlockProcessed); err != nil {
			return nil, err
		}
		vaultRegistry = append(vaultRegistry, &vault)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return vaultRegistry, nil
}

func (db *DB) InsertBlock(block *models.StarknetBlocks) error {
	hash := block.BlockHash
	parentHash := block.ParentHash
	query := `
	INSERT INTO starknet_blocks 
	(block_number, 
	block_hash, 
	parent_hash, 
	timestamp,
	status) 
	VALUES ($1, $2, $3, $4, 'MINED')
	`
	res, err := db.tx.Exec(context.Background(), query, block.BlockNumber, hash, parentHash, block.Timestamp)

	log.Printf("STORAGE RESULT %v %v", res, err)
	return err
}

func (db *DB) RevertBlock(blockNumber uint64, blockHash string) error {
	query := `	
	UPDATE starknet_blocks 
	SET status = 'REVERTED' 	
	WHERE block_number = $1 and block_hash = $2`
	_, err := db.tx.Exec(context.Background(), query, blockNumber, blockHash)
	return err
}

func (db *DB) GetVaultRegistryByAddress(address string) (models.VaultRegistry, error) {
	var vaultRegistry models.VaultRegistry
	query := `
	SELECT 
		vault_address, 
		deployed_at, 
		last_block_indexed, 
		last_block_processed
	FROM vault_registry 
	WHERE vault_address = $1`

	err := db.Pool.QueryRow(context.Background(), query, address).Scan(
		&vaultRegistry.Address,
		&vaultRegistry.DeployedAt,
		&vaultRegistry.LastBlockIndexed,
		&vaultRegistry.LastBlockProcessed,
	)
	return vaultRegistry, err
}

func (db *DB) GetNextBlock(hash string) (*models.StarknetBlocks, error) {
	var block models.StarknetBlocks

	log.Printf("Getting next block: %v", hash)
	query := `
	SELECT block_number, block_hash, parent_hash, timestamp, status FROM starknet_blocks 
	WHERE parent_hash = $1`
	err := db.Pool.QueryRow(context.Background(), query, hash).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp, &block.Status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &block, nil
}
func (db *DB) GetBlock(hash string) (*models.StarknetBlocks, error) {
	var block models.StarknetBlocks
	query := `
	SELECT * FROM starknet_blocks 
	WHERE block_hash = $1`
	err := db.Pool.QueryRow(context.Background(), query, hash).Scan(&block)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &block, err
}
func (db *DB) GetLastIndexedBlockVault(address string) (uint64, error) {
	var lastBlock uint64
	query := `
	SELECT last_block_indexed FROM vault_registry 
	WHERE vault_address = $1`
	err := db.Pool.QueryRow(context.Background(), query, address).Scan(&lastBlock)
	return lastBlock, err
}
func (db *DB) GetLastBlock() (*models.StarknetBlocks, error) {
	var lastBlock models.StarknetBlocks
	query := `
	SELECT block_number, block_hash, parent_hash, timestamp FROM starknet_blocks 
	WHERE STATUS = 'MINED'
	ORDER BY block_number DESC 
	LIMIT 1`
	if db.tx == nil {
		err := db.Pool.QueryRow(context.Background(), query).Scan(&lastBlock.BlockNumber, &lastBlock.BlockHash, &lastBlock.ParentHash, &lastBlock.Timestamp)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
	} else {
		err := db.Pool.QueryRow(context.Background(), query).Scan(&lastBlock.BlockNumber, &lastBlock.BlockHash, &lastBlock.ParentHash, &lastBlock.Timestamp)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
	}
	return &lastBlock, nil
}

func (db *DB) StoreEvent(txHash, vaultAddress string, blockNumber uint64, blockHash string, eventName string, eventKeys []string, eventData []string) error {

	if db.tx == nil {
		return errors.New("No transaction found")
	}
	log.Printf("Storing event %s %s %d %s %v %v", txHash, vaultAddress, blockNumber, eventName, eventKeys, eventData)
	query := `    
	INSERT INTO events 
	(transaction_hash, vault_address, block_number, block_hash, event_name, event_keys, event_data, event_nonce) 
	VALUES ($1, $2::varchar, $3, $4::varchar, $5, $6, $7,
		(SELECT COUNT(*) + 1 
		 FROM events 
		 WHERE vault_address = $2::varchar))`
	_, err := db.tx.Exec(context.Background(), query, txHash, vaultAddress, blockNumber, blockHash, eventName, eventKeys, eventData)
	if err != nil {
		log.Printf("WTHELLY")
		log.Printf("%v", err)
	}
	return err
}

func (db *DB) InsertVault(vault *models.VaultRegistry) error {
	query := `
	INSERT INTO vault_registry 
	(vault_address, deployed_at, last_block_indexed, last_block_processed) 
	VALUES ($1, $2, $3, $4)`
	_, err := db.tx.Exec(context.Background(), query, vault.Address, vault.DeployedAt, vault.LastBlockIndexed, vault.LastBlockProcessed)
	return err
}

func (db *DB) UpdateVaultRegistry(address string, blockHash string) error {
	query := `
	UPDATE vault_registry 
	SET last_block_indexed = $1 
	WHERE vault_address = $2`
	_, err := db.tx.Exec(context.Background(), query, blockHash, address)
	return err
}

// StoreDriverEvent stores a basic driver event (StartBlock/RevertBlock) and triggers PostgreSQL NOTIFY
func (db *DB) StoreDriverEvent(eventType string, blockHash string) error {
	if db.tx == nil {
		return errors.New("No transaction found")
	}
	
	// Store event in database with sequence index (triggers NOTIFY automatically)
	query := `
	INSERT INTO driver_events 
	(sequence_index, type, block_hash, timestamp) 
	VALUES (nextval('driver_events_sequence'), $1, $2, NOW())`
	_, err := db.tx.Exec(context.Background(), query, eventType, blockHash)
	return err
}

// StoreCatchupBlockEvent stores a catchup block event with start/end block hashes
func (db *DB) StoreCatchupBlockEvent(startBlockHash, endBlockHash string) error {
	if db.tx == nil {
		return errors.New("No transaction found")
	}
	
	// Store event in database with sequence index (triggers NOTIFY automatically)
	query := `
	INSERT INTO driver_events 
	(sequence_index, type, start_block_hash, end_block_hash, timestamp) 
	VALUES (nextval('driver_events_sequence'), $1, $2, $3, NOW())`
	_, err := db.tx.Exec(context.Background(), query, "CatchupBlock", startBlockHash, endBlockHash)
	return err
}

// StoreVaultCatchupEvent stores a vault catchup event and triggers PostgreSQL NOTIFY
func (db *DB) StoreVaultCatchupEvent(vaultAddress string, startBlockHash, endBlockHash string) error {
	if db.tx == nil {
		return errors.New("No transaction found")
	}
	
	// Store event in database with sequence index (triggers NOTIFY automatically)
	query := `
	INSERT INTO driver_events 
	(sequence_index, type, vault_address, start_block_hash, end_block_hash, timestamp) 
	VALUES (nextval('driver_events_sequence'), $1, $2, $3, $4, NOW())`
	_, err := db.tx.Exec(context.Background(), query, "CatchupVault", vaultAddress, startBlockHash, endBlockHash)
	return err
}

// GetUnprocessedDriverEvents returns unprocessed driver events for catchup
func (db *DB) GetUnprocessedDriverEvents(limit int) ([]*models.DriverEvent, error) {
	query := `
	SELECT id, sequence_index, type, block_hash, start_block_hash, end_block_hash, vault_address, timestamp, is_processed
	FROM driver_events 
	WHERE is_processed = FALSE 
	ORDER BY sequence_index ASC 
	LIMIT $1`
	
	rows, err := db.Pool.Query(context.Background(), query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var events []*models.DriverEvent
	for rows.Next() {
		var event models.DriverEvent
		var id int
		var blockHash, startBlockHash, endBlockHash, vaultAddress *string
		
		err := rows.Scan(&id, &event.SequenceIndex, &event.Type, &blockHash, &startBlockHash, &endBlockHash, &vaultAddress, &event.Timestamp, &event.IsProcessed)
		if err != nil {
			return nil, err
		}
		
		event.ID = id
		
		// Set fields based on event type
		if blockHash != nil {
			event.BlockHash = *blockHash
		}
		if startBlockHash != nil {
			event.StartBlockHash = *startBlockHash
		}
		if endBlockHash != nil {
			event.EndBlockHash = *endBlockHash
		}
		if vaultAddress != nil {
			event.VaultAddress = *vaultAddress
		}
		
		events = append(events, &event)
	}
	
	return events, nil
}

// MarkDriverEventProcessed marks a driver event as processed
func (db *DB) MarkDriverEventProcessed(id int) error {
	query := `UPDATE driver_events SET is_processed = TRUE WHERE id = $1`
	_, err := db.Pool.Exec(context.Background(), query, id)
	return err
}
