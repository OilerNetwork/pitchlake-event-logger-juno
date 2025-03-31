package db

import (
	"context"
	"encoding/json"
	"junoplugin/models"
	"log"

	"github.com/jackc/pgx/v5"
)

func (db *DB) GetVaultAddresses() ([]*string, error) {
	var vaultAddresses []*string
	rows, err := db.Pool.Query(context.Background(), "SELECT vault_address FROM vault_registry")
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var address string
		if err := rows.Scan(&address); err != nil {
			return nil, err
		}
		vaultAddresses = append(vaultAddresses, &address)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return vaultAddresses, nil
}

func (db *DB) InsertBlock(block *models.StarknetBlocks) error {
	hash := block.BlockHash
	parentHash := block.ParentHash
	query := `
	INSERT INTO starknet_blocks 
	(block_number, 
	block_hash, 
	parent_hash, 
	status) 
	VALUES ($1, $2, $3, 'MINED')
	`
	_, err := db.tx.Exec(context.Background(), query, block.BlockNumber, hash, parentHash)
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

func (db *DB) GetVaultRegistry(address string) (models.VaultRegistry, error) {
	var vaultRegistry models.VaultRegistry
	query := `
	SELECT * FROM vault_registry 
	WHERE vault_address = $1`
	err := db.tx.QueryRow(context.Background(), query, address).Scan(&vaultRegistry)
	return vaultRegistry, err
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
	SELECT * FROM starknet_blocks 
	WHERE STATUS = 'MINED'
	ORDER BY block_number DESC 
	LIMIT 1`
	if db.tx == nil {
		err := db.Pool.QueryRow(context.Background(), query).Scan(&lastBlock)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
	} else {
		err := db.tx.QueryRow(context.Background(), query).Scan(&lastBlock)
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

	query := `	
	INSERT INTO events 
	(transaction_hash, vault_address, block_number, event_name, event_keys, event_data) 
	VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.tx.Exec(context.Background(), query, txHash, vaultAddress, blockNumber, eventName, eventKeys, eventData)
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
