package models

import (
	"database/sql/driver"
	"fmt"
	"math/big"

	"github.com/lib/pq"
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

type Vault struct {
	BlockNumber     uint64 `gorm:"column:block_number;type:numeric(78,0);not null"`
	UnlockedBalance BigInt `gorm:"column:unlocked_balance;not null"`
	LockedBalance   BigInt `gorm:"column:locked_balance;not null"`
	StashedBalance  BigInt `gorm:"column:stashed_balance;not null"`
}

type LiquidityProvider struct {
	VaultAddress    string `gorm:"column:vault_address;not null"`
	Address         string `gorm:"column:address;not null"`
	UnlockedBalance BigInt `gorm:"column:unlocked_balance;not null"`
	LockedBalance   BigInt `gorm:"column:locked_balance;not null"`
	StashedBalance  BigInt `gorm:"column:stashed_balance;not null"`
	BlockNumber     uint64 `gorm:"column:block_number;not null"`
}

type OptionBuyer struct {
	Address string `gorm:"column:address;not null"`
	//Maybe this is not required and can be directly fetched as a view/index on the bids table
	//Bids       string `gorm:"column:bids;type:jsonb"` // Store bids as JSON in PostgreSQL
	RoundAddress      string `gorm:"column:round_address;not null"`
	MintableOptions   BigInt `gorm:"column:mintable_options;"`
	HasMinted         bool   `gorm:"column:has_minted;"`
	RefundableOptions BigInt `gorm:"column:refundable_amount;"`
	HasRefunded       bool   `gorm:"column:has_refunded;"`
}

type OptionRound struct {
	VaultAddress       string `gorm:"column:vault_address;"`
	Address            string `gorm:"column:address"`
	RoundID            BigInt `gorm:"column:round_id;"` // Store bids as JSON in PostgreSQL
	CapLevel           BigInt `gorm:"column:cap_level"`
	StartDate          uint64 `gorm:"column:start_date;"`
	EndDate            uint64 `gorm:"column:end_date;"`
	SettlementDate     uint64 `gorm:"column:settlement_date;"`
	StartingLiquidity  BigInt `gorm:"column:starting_liquidity;"`
	QueuedLiquidity    BigInt `gorm:"column:queued_liquidity;"`
	RemainingLiquidity BigInt `gorm:"column:remaining_liquidity;"`
	AvailableOptions   BigInt `gorm:"column:available_options;"`
	SettlementPrice    BigInt `gorm:"column:settlement_price;"`
	StrikePrice        BigInt `gorm:"column:strike_price;"`
	UnsoldLiquidity    BigInt `gorm:"column:unsold_liquidity;"`
	SoldOptions        BigInt `gorm:"column:sold_options;"`
	ReservePrice       BigInt `gorm:"column:reserve_price"`
	ClearingPrice      BigInt `gorm:"column:clearing_price"`
	State              string `gorm:"column:state;"`
	Premiums           BigInt `gorm:"column:premiums;"`
	PayoutPerOption    BigInt `gorm:"column:payout_per_option;"`
	DeploymentDate     uint64 `gorm:"column:deployment_date;"`
}

type VaultState struct {
	CurrentRound          BigInt `gorm:"column:current_round;not null;"`
	CurrentRoundAddress   string `gorm:"column:current_round_address;"`
	UnlockedBalance       BigInt `gorm:"column:unlocked_balance;"`
	LockedBalance         BigInt `gorm:"column:locked_balance;"`
	StashedBalance        BigInt `gorm:"column:stashed_balance;"`
	Address               string `gorm:"column:address;not null;"`
	LatestBlock           uint64 `gorm:"column:latest_block;"`
	FossilClientAddress   string `gorm:"column:fossil_client_address;"`
	EthAddress            string `gorm:"column:eth_address;"`
	OptionRoundClassHash  string `gorm:"column:option_round_class_hash;"`
	Alpha                 BigInt `gorm:"column:alpha;"`
	StrikeLevel           BigInt `gorm:"column:strike_level;"`
	RoundTransitionPeriod uint64 `gorm:"column:round_transition_period;"`
	AuctionDuration       uint64 `gorm:"column:auction_duration;"`
	RoundDuration         uint64 `gorm:"column:round_duration;"`
	DeploymentDate        uint64 `gorm:"column:deployment_date;"`
}

type LiquidityProviderState struct {
	VaultAddress    string `gorm:"column:vault_address;not null"`
	Address         string `gorm:"column:address;not null;primaryKey"`
	UnlockedBalance BigInt `gorm:"column:unlocked_balance;not null"`
	LockedBalance   BigInt `gorm:"column:locked_balance;not null"`
	StashedBalance  BigInt `gorm:"column:stashed_balance;"`
	LatestBlock     uint64 `gorm:"column:latest_block;"`
}

type QueuedLiquidity struct {
	Address      string `gorm:"column:address;not null"`
	RoundAddress string `gorm:"column:round_address;not null"`
	Bps          BigInt `gorm:"column:bps;not null"`
	QueuedAmount BigInt `gorm:"column:queued_liquidity;not null"`
}
type Bid struct {
	BuyerAddress string `gorm:"column:buyer_address;not null"`
	RoundAddress string `gorm:"column:round_address;not null"`
	BidID        string `gorm:"column:bid_id;not null"`
	TreeNonce    uint64 `gorm:"column:tree_nonce;not null"`
	Amount       BigInt `gorm:"column:amount;not null"`
	Price        BigInt `gorm:"column:price;not null"`
}

type Event struct {
	ID              uint           `gorm:"primaryKey;column:id;autoIncrement"`
	TransactionHash string         `gorm:"column:transaction_hash;not null"`
	BlockNumber     uint64         `gorm:"column:block_number;not null"`
	VaultAddress    string         `gorm:"column:vault_address;not null"`
	Timestamp       uint64         `gorm:"column:timestamp;not null"`
	EventName       string         `gorm:"column:event_name;not null"`
	EventKeys       pq.StringArray `gorm:"column:event_keys;type:varchar(256)[];not null"`
	EventData       pq.StringArray `gorm:"column:event_data;type:varchar(256)[];not null"`
	EventIndex      int            `gorm:"column:event_index;not null"`
}

func (VaultState) TableName() string {
	return "VaultStates"
}
func (LiquidityProviderState) TableName() string {
	return "Liquidity_Providers"
}

func (OptionRound) TableName() string {
	return "Option_Rounds"
}

func (QueuedLiquidity) TableName() string {
	return "Queued_Liquidity"
}

func (Bid) TableName() string {
	return "Bids"
}

func (OptionBuyer) TableName() string {
	return "Option_Buyers"
}

func (Event) TableName() string {
	return "Events"
}
