package models

import (
	"database/sql/driver"
	"fmt"
	"math/big"

	"github.com/NethermindEth/juno/core"
)

type BigInt struct {
	*big.Int
}

// NewBigInt creates a new BigInt from a string
func NewBigInt(s string) *BigInt {
	i := new(big.Int)
	i.SetString(s, 10)
	return &BigInt{i}
}

// Scan implements the sql.Scanner interface for BigInt
func (b *BigInt) Scan(value interface{}) error {
	if b.Int == nil {
		b.Int = new(big.Int)
	}
	switch v := value.(type) {
	case string:
		_, ok := b.Int.SetString(v, 10)
		if !ok {
			return fmt.Errorf("failed to scan BigInt: invalid string %s", v)
		}
	case []byte:
		_, ok := b.Int.SetString(string(v), 10)
		if !ok {
			return fmt.Errorf("failed to scan BigInt: invalid bytes %s", v)
		}
	case int64:
		b.Int.SetInt64(v)
	default:
		return fmt.Errorf("unsupported scan type for BigInt: %T", value)
	}
	return nil
}

// Value implements the driver.Valuer interface for BigInt
func (b BigInt) Value() (driver.Value, error) {
	if b.Int == nil {
		return "0", nil
	}
	return b.Int.String(), nil
}

func CoreToStarknetBlock(block core.Block) StarknetBlocks {
	starknetBlock := StarknetBlocks{
		BlockNumber: block.Number,
		BlockHash:   block.Hash.String(),
		ParentHash:  block.ParentHash.String(),
		Timestamp:   block.Timestamp,
		Status:      "MINED",
	}
	return starknetBlock
}

type Event struct {
	ID              uint     `json:"id"`
	TransactionHash string   `json:"transaction_hash"`
	BlockNumber     uint64   `json:"block_number"`
	VaultAddress    string   `json:"vault_address"`
	Timestamp       uint64   `json:"timestamp"`
	EventName       string   `json:"event_name"`
	EventKeys       []string `json:"event_keys"`
	EventData       []string `json:"event_data"`
	EventNonce      int      `json:"event_nonce"`
}

type StarknetBlocks struct {
	BlockNumber uint64 `json:"block_number"`
	Timestamp   uint64 `json:"timestamp"`
	BlockHash   string `json:"block_hash"`
	ParentHash  string `json:"parent_hash"`
	Status      string `json:"status"`
}

type VaultRegistry struct {
	ID                 uint    `json:"id"`
	Address            string  `json:"address"`
	DeployedAt         string  `json:"deployed_at"`
	LastBlockIndexed   *string `json:"last_block_indexed"`
	LastBlockProcessed *string `json:"last_block_processed"`
}
