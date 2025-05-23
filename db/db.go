package db

import (
	"context"
	"encoding/json"
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
		last_block_indexed 
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
		if err := rows.Scan(&vault.Address, &vault.DeployedAt, &vault.LastBlockIndexed, &vault.LastBlockIndexed); err != nil {
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
	_, err := db.tx.Exec(context.Background(), query, block.BlockNumber, hash, parentHash, block.Timestamp)
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

	err := db.Conn.QueryRow(context.Background(), query, address).Scan(
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
	SELECT * FROM starknet_blocks 
	WHERE parent_hash = $1`
	row, err := db.Conn.Query(context.Background(), query, hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer row.Close()
	for row.Next() {
		err = row.Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp)
		if err != nil {
			return nil, err
		}
	}
	return &block, nil
}
func (db *DB) GetBlock(hash string) (models.StarknetBlocks, error) {
	var block models.StarknetBlocks
	query := `
	SELECT * FROM starknet_blocks 
	WHERE block_hash = $1`
	err := db.tx.QueryRow(context.Background(), query, hash).Scan(&block)
	return block, err
}
func (db *DB) GetLastIndexedBlockVault(address string) (uint64, error) {
	var lastBlock uint64
	query := `
	SELECT last_block_indexed FROM vault_registry 
	WHERE vault_address = $1`
	err := db.tx.QueryRow(context.Background(), query, address).Scan(&lastBlock)
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
		err := db.tx.QueryRow(context.Background(), query).Scan(&lastBlock.BlockNumber, &lastBlock.BlockHash, &lastBlock.ParentHash, &lastBlock.Timestamp)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
	}
	return &lastBlock, nil
}

func (db *DB) StoreEvent(txHash, vaultAddress string, blockNumber uint64, eventName string, eventKeys []string, eventData []string) error {
	log.Printf("Storing event %s %s %d %s %v %v", txHash, vaultAddress, blockNumber, eventName, eventKeys, eventData)
	query := `    
	INSERT INTO events 
	(transaction_hash, vault_address, block_number, event_name, event_keys, event_data, event_count) 
	VALUES ($1, $2::varchar, $3, $4, $5, $6, 
		(SELECT COALESCE(MAX(event_count), 0) + 1 
		 FROM events 
		 WHERE vault_address = $2::varchar))`
	_, err := db.tx.Exec(context.Background(), query, txHash, vaultAddress, blockNumber, eventName, eventKeys, eventData)
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
func (db *DB) ListenerNewVault(channel chan<- models.VaultRegistry) {
	db.Conn.Exec(context.Background(), "LISTEN new_vault")

	for {
		notification, err := db.Conn.WaitForNotification(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		var vault models.VaultRegistry
		json.Unmarshal([]byte(notification.Payload), &vault)
		channel <- vault
	}
}
