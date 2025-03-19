package db

import (
	"errors"
	"junoplugin/models"
	"log"
	"math/big"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DB struct {
	Conn *gorm.DB
	tx   *gorm.DB
}

func Init(dsn string) (*DB, error) {
	log.Printf("connecting to %s", dsn)
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})

	if err != nil {
		return nil, err
	}

	m, err := migrate.New(
		"file://db/migrations",
		dsn)
	if err != nil {
		log.Printf("FAIlED HERE 1")
		return nil, err
	}
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return nil, err
		}

	}
	m.Close()
	// Automatically migrate your schema
	// err = conn.AutoMigrate(
	// 	&models.Vault{},
	// 	&models.LiquidityProvider{},
	// 	&models.OptionBuyer{},
	// 	&models.OptionRound{},
	// 	&models.VaultState{},
	// 	&models.Bid{},
	// )
	// if err != nil {
	// 	log.Fatalf("Failed to migrate database schema: %v", err)
	// 	return nil, err
	// }

	return &DB{Conn: conn}, nil
}

func (db *DB) Close() error {
	//Close the DB connection
	sqlDB, err := db.Conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *DB) CreateVault(vault *models.VaultState) error {
	if err := db.tx.Create(vault).Error; err != nil {
		return err
	}
	return nil
}

func (db *DB) UpdateVaultBalanceAuctionStart(vaultAddress string, blockNumber uint64) error {
	return db.tx.Model(models.VaultState{}).Where("address=?", vaultAddress).Updates(
		map[string]interface{}{
			"unlocked_balance": 0,
			"locked_balance":   gorm.Expr("unlocked_balance"),
			"latest_block":     blockNumber,
		}).Error
}

func (db *DB) UpdateVaultBalancesAuctionEnd(
	vaultAddress string,
	unsoldLiquidity,
	premiums models.BigInt,
	blockNumber uint64) error {
	return db.tx.Model(models.VaultState{}).Where("address=?", vaultAddress).Updates(
		map[string]interface{}{
			"unlocked_balance": gorm.Expr("unlocked_balance+?+?", unsoldLiquidity, premiums),
			"locked_balance":   gorm.Expr("locked_balance-?", unsoldLiquidity),
			"latest_block":     blockNumber,
		}).Error

}

func (db *DB) UpdateAllLiquidityProvidersBalancesAuctionStart(vaultAddress string, blockNumber uint64) error {
	return db.tx.Model(models.LiquidityProviderState{}).Where("vault_address=? AND unlocked_balance > 0", vaultAddress).Updates(
		map[string]interface{}{
			"locked_balance":   gorm.Expr("unlocked_balance"),
			"unlocked_balance": 0,
			"latest_block":     blockNumber,
		}).Error
}

func (db *DB) UpdateAllLiquidityProvidersBalancesAuctionEnd(
	vaultAddress string,
	startingLiquidity,
	unsoldLiquidity,
	premiums models.BigInt,
	blockNumber uint64) error {

	zero := models.BigInt{
		Int: big.NewInt(0),
	}
	if startingLiquidity.Cmp(zero.Int) == 0 {
		return nil
	}
	return db.tx.Model(models.LiquidityProviderState{}).Where("vault_address=?", vaultAddress).Updates(
		map[string]interface{}{
			"locked_balance":   gorm.Expr("locked_balance-FLOOR((locked_balance*?)/?)", unsoldLiquidity, startingLiquidity),
			"unlocked_balance": gorm.Expr("unlocked_balance+FLOOR((locked_balance*?))/?+FLOOR((?*locked_balance)/?)", unsoldLiquidity, startingLiquidity, premiums, startingLiquidity),
			"latest_block":     blockNumber,
		}).Error
}

func (db *DB) UpdateOptionRoundAuctionEnd(
	address string,
	clearingPrice,
	optionsSold, unsoldLiquidity, premiums models.BigInt) error {
	err := db.UpdateOptionRoundFields(
		address,
		map[string]interface{}{
			"clearing_price":   clearingPrice,
			"sold_options":     optionsSold,
			"state":            "Running",
			"unsold_liquidity": unsoldLiquidity,
			"premiums":         premiums,
		})
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) UpdateBiddersAuctionEnd(
	roundAddress string,
	clearingPrice,
	clearingOptionsSold models.BigInt,
	clearingNonce uint64) error {
	bidsAbove, err := db.GetBidsAboveClearingForRound(roundAddress, clearingPrice, clearingNonce)
	if err != nil {
		return err
	}

	optionsLeft := models.NewBigInt(clearingOptionsSold.String())
	for _, bid := range bidsAbove {
		if clearingNonce == bid.TreeNonce {
			log.Printf("optionsLeft %v", optionsLeft)
			refundableAmount := models.BigInt{Int: new(big.Int).Mul(new(big.Int).Sub(bid.Amount.Int, optionsLeft.Int), clearingPrice.Int)}
			log.Printf("REFUNDABLEAMOUNT %v", refundableAmount)
			err := db.UpdateOptionBuyerFields(bid.BuyerAddress, roundAddress, map[string]interface{}{
				"refundable_amount": gorm.Expr("refundable_amount+?", refundableAmount),
				"mintable_options":  gorm.Expr("mintable_options+?", optionsLeft),
			})
			if err != nil {
				return err
			}

		} else {
			refundableAmount := models.BigInt{Int: new(big.Int).Mul(new(big.Int).Sub(bid.Price.Int, clearingPrice.Int), bid.Amount.Int)}
			log.Printf("REFUNDABLEAMOUNT %v", refundableAmount)
			err := db.UpdateOptionBuyerFields(bid.BuyerAddress, roundAddress, map[string]interface{}{
				"mintable_options":  gorm.Expr("mintable_options+?", bid.Amount),
				"refundable_amount": gorm.Expr("refundable_amount+?", refundableAmount),
			})
			if err != nil {
				return err
			}
			optionsLeft.Sub(optionsLeft.Int, bid.Amount.Int)
		}

	}
	bidsBelow, err := db.GetBidsBelowClearingForRound(roundAddress, clearingPrice, clearingNonce)
	if err != nil {
		return err
	}
	for _, bid := range bidsBelow {
		refundableAmount := models.BigInt{Int: new(big.Int).Mul(bid.Amount.Int, bid.Price.Int)}
		log.Printf("REFUNDABLEAMOUNT %v", refundableAmount)
		err := db.UpdateOptionBuyerFields(bid.BuyerAddress, roundAddress, map[string]interface{}{

			"refundable_amount": gorm.Expr("refundable_amount+?", refundableAmount),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) UpdateVaultBalancesOptionSettle(
	vaultAddress string,
	remainingLiquidityStashed,
	remainingLiquidityNotStashed models.BigInt,
	blockNumber uint64,
) error {
	return db.tx.Model(models.VaultState{}).Where("address=?", vaultAddress).Updates(map[string]interface{}{

		"stashed_balance":  gorm.Expr("stashed_balance+ ? ", remainingLiquidityStashed),
		"unlocked_balance": gorm.Expr("unlocked_balance+?", remainingLiquidityNotStashed),
		"locked_balance":   0,
		"latest_block":     blockNumber,
	}).Error

}
func (db *DB) UpdateAllLiquidityProvidersBalancesOptionSettle(
	vaultAddress,
	roundAddress string,
	startingLiquidity,
	remainingLiquidty,
	remainingLiquidtyNotStashed,
	unsoldLiquidity,
	payoutPerOption,
	optionsSold models.BigInt,
	blockNumber uint64,
) error {

	//	totalPayout := models.BigInt{Int: new(big.Int).Mul(optionsSold.Int, payoutPerOption.Int)}
	db.tx.Model(models.LiquidityProviderState{}).Where("vault_address = ? AND locked_balance > 0", vaultAddress).Updates(map[string]interface{}{
		"locked_balance": 0,
		"unlocked_balance": gorm.Expr(`
			unlocked_balance + FLOOR(
				CASE 
					WHEN ?::numeric - ?::numeric <> 0 
					THEN locked_balance * ? / (?::numeric - ?::numeric) 
					ELSE locked_balance
				END
			)`, remainingLiquidty, startingLiquidity, remainingLiquidty, startingLiquidity, unsoldLiquidity),
		"latest_block": blockNumber,
	})
	queuedAmounts, err := db.GetAllQueuedLiquidityForRound(roundAddress)
	if err != nil {
		return err
	}
	for _, queuedAmount := range queuedAmounts {

		amountToAdd := &models.BigInt{Int: new(big.Int).Div(new(big.Int).Mul(remainingLiquidty.Int, queuedAmount.QueuedAmount.Int), (startingLiquidity.Int))}
		db.tx.Model(models.LiquidityProviderState{}).Where("vault_address=? AND address = ?", vaultAddress, queuedAmount.Address).
			Updates(map[string]interface{}{
				"stashed_balance":  gorm.Expr("stashed_balance + ?", amountToAdd),
				"unlocked_balance": gorm.Expr("unlocked_balance - ?", amountToAdd),
			})
	}

	/* Use this JOIN query to update this without creating 2 entries on the historic table
	// Perform the update in a single query using JOINs and subqueries
	err := db.Conn.Exec(`
		UPDATE liquidity_provider_states lps
		JOIN (
			SELECT
				address,
				queued_amount,
				remaining_liquidity * queued_amount / ? AS amount_to_add
			FROM queued_liquidity
			WHERE round_id = ?
		) ql ON lps.address = ql.address AND lps.round_id = ?
		SET
			lps.locked_balance = 0,
			lps.unlocked_balance = lps.unlocked_balance + lps.locked_balance - lps.locked_balance * ? / ? - ql.amount_to_add,
			lps.stashed_balance = lps.stashed_balance + ql.amount_to_add
	`, startingLiquidity, roundID, roundID, totalPayout, startingLiquidity).Error
	*/
	return nil
}
func (db *DB) GetVaultByAddress(address string) (*models.VaultState, error) {
	var vault models.VaultState
	if err := db.tx.Where("address = ?", address).First(&vault).Error; err != nil {
		return nil, err
	}
	return &vault, nil
}

func (db *DB) GetVaultAddresses() ([]string, error) {
	var addresses []string

	// Use Pluck to retrieve only the "address" field from the VaultState model
	err := db.Conn.Model(&models.VaultState{}).Pluck("address", &addresses).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]string, 0), nil
		} else {
			return nil, err
		}
	}
	return addresses, nil
}

func (db *DB) GetRoundAddressess(vaultAddress string) (*[]string, error) {
	var addresses []string

	// Use Pluck to retrieve only the "address" field from the VaultState model
	err := db.Conn.Model(&models.OptionRound{}).Where("vault_address = ?", vaultAddress).Pluck("address", &addresses).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &[]string{}, nil
		} else {
			return nil, err
		}
	}
	return &addresses, nil
}
func (db *DB) CreateOptionBuyer(buyer *models.OptionBuyer) error {
	err := db.tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}, {Name: "round_address"}}, // Composite primary key columns
		DoNothing: true,                                                        // Do nothing if a conflict occurs
	}).Create(buyer).Error
	return err
}

func (db *DB) UpsertQueuedLiquidity(queuedLiquidity *models.QueuedLiquidity) error {
	// Log the input for debugging
	// Log the input for debugging

	// Perform upsert using GORM's Clauses with the transaction object
	err := db.tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}, {Name: "round_address"}},
		DoUpdates: clause.AssignmentColumns([]string{"bps", "queued_liquidity"}),
	}).Create(queuedLiquidity).Error

	if err != nil {
		log.Printf("Upsert error: %v", err)
		return err
	}

	return nil

}

func (db *DB) UpsertLiquidityProviderState(lp *models.LiquidityProviderState, blockNumber uint64) error {
	// Log the input for debugging
	// Log the input for debugging
	log.Printf("Upserting LP: %+v, Block Number: %d", lp, blockNumber)

	// Perform upsert using GORM's Clauses with the transaction object
	err := db.tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}, {Name: "vault_address"}},
		DoUpdates: clause.AssignmentColumns([]string{"unlocked_balance", "latest_block"}),
	}).Create(lp).Error

	if err != nil {
		log.Printf("Upsert error: %v", err)
		return err
	}

	return nil

}

func (db *DB) UpdateOptionBuyerFields(
	address string,
	roundAddress string,
	updates map[string]interface{},
) error {
	return db.tx.Model(models.OptionBuyer{}).Where("address = ? AND round_address = ?", address, roundAddress).Updates(updates).Error
}

func (db *DB) UpdateAllOptionBuyerFields(roundAddress string, updates map[string]interface{}) error {
	return db.tx.Model(models.OptionRound{}).Where("round_address=?", roundAddress).Updates(updates).Error
}

func (db *DB) GetOptionRoundByAddress(address string) (*models.OptionRound, error) {
	var or models.OptionRound
	if err := db.tx.Where("address = ?", address).First(&or).Error; err != nil {
		return nil, err
	}
	return &or, nil
}

func (db *DB) UpdateOptionRoundFields(address string, updates map[string]interface{}) error {
	return db.tx.Model(models.OptionRound{}).Where("address = ?", address).Updates(updates).Error
}

func (db *DB) UpdateVaultFields(address string, updates map[string]interface{}) error {
	return db.tx.Model(models.VaultState{}).Where("address = ?", address).Updates(updates).Error
}
func (db *DB) UpdateLiquidityProviderFields(vaultAddress, address string, updates map[string]interface{}) error {
	return db.tx.Model(models.LiquidityProviderState{}).Where("vault_address = ? AND address = ?", vaultAddress, address).Updates(updates).Error
}

// DeleteOptionRound deletes an OptionRound record by its ID
func (db *DB) DeleteOptionRound(roundAddress string) error {
	if err := db.tx.Where("address = ?", roundAddress).Delete(&models.OptionRound{}).Error; err != nil {
		return err
	}
	return nil
}

// CreateBid creates a new Bid record in the database
func (db *DB) CreateBid(bid *models.Bid) error {
	if err := db.tx.Create(bid).Error; err != nil {
		return err
	}
	return nil
}
func (db *DB) CreateOptionRound(round *models.OptionRound) error {
	if err := db.tx.Create(round).Error; err != nil {
		return err
	}
	return nil
}

// DeleteBid deletes a Bid record by its ID
func (db *DB) DeleteBid(bidID string, roundAddress string) error {
	if err := db.tx.Model(&models.Bid{}).Where("round_address=? AND bid_id=?", roundAddress, bidID).Error; err != nil {
		return err
	}
	return nil
}
func (db *DB) GetBidsForRound(roundAddress string) ([]models.Bid, error) {
	var bids []models.Bid
	if err := db.Conn.Where("round_address = ?", roundAddress).Order("price DESC,tree_nonce ASC").Find(&bids).Error; err != nil {
		return nil, err
	}
	return bids, nil
}

func (db *DB) GetBidsAboveClearingForRound(
	roundAddress string,
	clearingPrice models.BigInt,
	clearingNonce uint64,
) ([]models.Bid, error) {
	var bids []models.Bid
	if err := db.Conn.
		Where("round_address = ?", roundAddress).
		Where("price > ? OR (price = ? AND tree_nonce <= ?)", clearingPrice, clearingPrice, clearingNonce).
		Order("price DESC, tree_nonce ASC").
		Find(&bids).Error; err != nil {
		return nil, err
	}
	log.Printf("BIDS ABOVE %v", bids)
	return bids, nil
}

func (db *DB) GetBidsBelowClearingForRound(
	roundAddress string,
	clearingPrice models.BigInt,
	clearingNonce uint64,
) ([]models.Bid, error) {
	var bids []models.Bid

	if err := db.Conn.Where("round_address = ?", roundAddress).
		Where("price < ? OR ( price = ? AND tree_nonce >?) ", clearingPrice, clearingPrice, clearingNonce).
		Find(&bids).Error; err != nil {
		return nil, err
	}
	log.Printf("BIDS ABOVE %v", bids)
	return bids, nil
}

func (db *DB) GetAllQueuedLiquidityForRound(roundAddress string) ([]models.QueuedLiquidity, error) {

	var queuedAmounts []models.QueuedLiquidity
	if err := db.Conn.Where("round_address=?", roundAddress).Find(&queuedAmounts).Error; err != nil {
		return nil, err
	}
	return queuedAmounts, nil
}

// Revert Functions
func (db *DB) RevertVaultState(address string, blockNumber uint64) error {
	var vaultState models.VaultState
	var vaultHistoric, postRevert models.Vault
	if err := db.tx.Where("address = ? AND last_block = ?", address, blockNumber).First(&vaultState).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	}

	if err := db.tx.Where("address = ? AND block_number = ?", address, blockNumber).First(&vaultHistoric).Error; err != nil {
		return err
	}

	if err := db.tx.Delete(&vaultHistoric).Error; err != nil {
		return err
	}

	if err := db.tx.Where("address = ?", address).
		Order("latest_block DESC").
		First(&postRevert).Error; err != nil {
		return nil
	}

	if err := db.tx.Where("address = ?").Updates(map[string]interface{}{
		"unlocked_balance": postRevert.UnlockedBalance,
		"locked_balance":   postRevert.LockedBalance,
		"stashed_balance":  postRevert.StashedBalance,
		"latest_block":     postRevert.BlockNumber,
	}).Error; err != nil {
		return err
	}

	return nil
}

func (db *DB) RevertAllLPState(vaultAddress string, blockNumber uint64) error {
	var lpStates []models.LiquidityProviderState
	var lpHistoric, postRevert models.LiquidityProvider
	if err := db.tx.Model(models.LiquidityProviderState{}).Where("vault_address = ? AND last_block = ?", vaultAddress, blockNumber).Find(&lpStates).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	}

	for _, lpState := range lpStates {
		if err := db.tx.Model(models.LiquidityProvider{}).Where("vault_address = ? AND address = ? AND block_number = ?", vaultAddress, lpState.Address, blockNumber).First(&lpHistoric).Error; err != nil {
			return err
		}

		if err := db.tx.Delete(&lpHistoric).Error; err != nil {
			return err
		}

		if err := db.tx.Where("vault_address = ? AND address = ?", vaultAddress, lpState.Address).
			Order("latest_block DESC").
			First(&postRevert).Error; err != nil {
			return nil
		}

		if err := db.tx.Where("vault_address = ? AND address = ?").Updates(map[string]interface{}{
			"unlocked_balance": postRevert.UnlockedBalance,
			"locked_balance":   postRevert.LockedBalance,
			"stashed_balance":  postRevert.StashedBalance,
			"latest_block":     postRevert.BlockNumber,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) RevertLPState(vaultAddress, address string, blockNumber uint64) error {
	var lpState models.LiquidityProviderState
	var lpHistoric, postRevert models.LiquidityProvider
	if err := db.tx.Where("vault_address = ? AND address = ? AND last_block = ?", vaultAddress, address, blockNumber).First(&lpState).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	}

	if err := db.tx.Model(models.LiquidityProvider{}).Where("vault_address = ? AND address = ? AND block_number = ?", vaultAddress, address, blockNumber).First(&lpHistoric).Error; err != nil {
		return err
	}

	if err := db.tx.Delete(&lpHistoric).Error; err != nil {
		return err
	}

	if err := db.tx.Model(models.LiquidityProvider{}).Where("vault_address = ? AND address = ?", vaultAddress, address).
		Order("latest_block DESC").
		First(&postRevert).Error; err != nil {
		return nil
	}

	if err := db.tx.Model(models.LiquidityProviderState{}).Where("vault_address = ? AND address = ?", vaultAddress, address).Updates(map[string]interface{}{
		"unlocked_balance": postRevert.UnlockedBalance,
		"locked_balance":   postRevert.LockedBalance,
		"stashed_balance":  postRevert.StashedBalance,
		"latest_block":     postRevert.BlockNumber,
	}).Error; err != nil {
		return err
	}

	return nil
}

func (db *DB) Begin() {
	db.tx = db.Conn.Begin()
}

func (db *DB) Commit() {
	if db.tx != nil {
		db.tx.Commit()
	}
}

func (db *DB) Rollback() {
	if db.tx != nil {
		db.tx.Rollback()
		db.tx = nil
	}
}

func (db *DB) Tx(tx *gorm.DB) {
	db.tx = tx
}

func (db *DB) StoreEvent(txHash, vaultAddress string, blockNumber uint64, timestamp uint64, eventName string, eventKeys []string, eventData []string) error {
	event := models.Event{
		TransactionHash: txHash,
		VaultAddress:    vaultAddress,
		BlockNumber:     blockNumber,
		Timestamp:       timestamp,
		EventName:       eventName,
		EventKeys:       pq.StringArray(eventKeys),
		EventData:       pq.StringArray(eventData),
	}
	return db.tx.Create(&event).Error
}
