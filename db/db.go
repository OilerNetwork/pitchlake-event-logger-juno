package db

import (
	"context"
	"encoding/json"
	"junoplugin/models"
	"junoplugin/utils"
	"log"

	"github.com/NethermindEth/juno/core"
)

func (db *DB) GetVaultAddresses() ([]string, error) {
	var vaultAddresses []string
	if err := db.tx.QueryRow(context.Background(), "SELECT address FROM vault_registry").Scan(&vaultAddresses); err != nil {
		return nil, err
	}
	return vaultAddresses, nil
}

func (db *DB) InsertVaultRegistry(vault models.VaultRegistry) error {
	query := `
	INSERT INTO vault_registry (address, deployed_at, last_block) 
	VALUES ($1, $2, $3)
	ON CONFLICT (address) 
	DO UPDATE SET 
		deployed_at = $2, 
		last_block = $3	
	`
	_, err := db.tx.Exec(context.Background(), query, vault.Address, vault.DeployedAt, vault.LastBlock)
	return err
}

func (db *DB) InsertBlock(coreBlock *core.Block) error {
	hash := utils.FeltToHexString(coreBlock.Hash.Bytes())
	parentHash := utils.FeltToHexString(coreBlock.ParentHash.Bytes())
	query := `
	INSERT INTO starknet_blocks 
	(block_number, 
	block_hash, 
	parent_hash, 
	status, 
	is_processed) 
	VALUES ($1, $2, $3, 'MINED', FALSE)
	`
	_, err := db.tx.Exec(context.Background(), query, coreBlock.Number, hash, parentHash)
	return err
}

func (db *DB) RevertBlock(blockNumber uint64, blockHash string) error {
	query := `	
	UPDATE starknet_blocks 
	SET status = 'REVERTED' AND is_processed = FALSE 
	WHERE block_number = $1 and block_hash = $2`
	_, err := db.tx.Exec(context.Background(), query, blockNumber, blockHash)
	return err
}

func (db *DB) GetLastBlockVault(address string) (uint64, error) {
	var lastBlock uint64
	query := `
	SELECT last_block FROM vault_registry 
	WHERE address = $1`
	err := db.tx.QueryRow(context.Background(), query, address).Scan(&lastBlock)
	return lastBlock, err
}
func (db *DB) GetLastBlock() (uint64, error) {
	var lastBlock uint64
	query := `
	SELECT block_number FROM starknet_blocks 
	ORDER BY block_number DESC 
	LIMIT 1`
	if db.tx == nil {
		err := db.Conn.QueryRow(context.Background(), query).Scan(&lastBlock)
		if err != nil {
			return 0, err
		}
	} else {
		err := db.tx.QueryRow(context.Background(), query).Scan(&lastBlock)
		if err != nil {
			return 0, err
		}
	}
	return lastBlock, nil
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
