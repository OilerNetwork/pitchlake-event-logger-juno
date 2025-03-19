package db

import (
	"junoplugin/models"

	"gorm.io/gorm"
)

func (db *DB) DepositOrWithdrawRevert(vaultAddress, lpAddress string, blockNumber uint64) error {
	//Map the other parameters as well

	if err := db.RevertVaultState(vaultAddress, blockNumber); err != nil {
		return err
	}

	if err := db.RevertLPState(vaultAddress, lpAddress, blockNumber); err != nil {
		return err
	}
	return nil
}

func (db *DB) WithdrawalQueuedRevertIndex(
	lpAddress, vaultAddress string,
	roundId uint64,
	bps, accountQueuedBefore, accountQueuedNow, vaultQueuedNow models.BigInt,
	blockNumber uint64,
) error {

	db.RevertVaultState(vaultAddress, blockNumber)
	db.RevertLPState(vaultAddress, lpAddress, blockNumber)

	vault, err := db.GetVaultByAddress(vaultAddress)
	if err != nil {
		return err
	}
	queuedLiquidity := models.QueuedLiquidity{
		Address:      lpAddress,
		RoundAddress: vault.CurrentRoundAddress,
		Bps:          bps,
		QueuedAmount: accountQueuedBefore,
	}
	if err := db.UpsertQueuedLiquidity(&queuedLiquidity); err != nil {
		return err
	}

	var vaultQueued, diff models.BigInt
	diff.Sub(accountQueuedBefore.Int, accountQueuedNow.Int)
	diff.Abs(diff.Int)
	if accountQueuedBefore.Cmp(accountQueuedNow.Int) < 0 {
		vaultQueued.Sub(vaultQueuedNow.Int, diff.Int)
	} else {
		vaultQueued.Add(vaultQueuedNow.Int, diff.Int)
	}
	if err := db.UpdateOptionRoundFields(vault.CurrentRoundAddress, map[string]interface{}{
		"queued_liquidity": vaultQueued,
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) RoundDeployedRevert(roundAddress string) {

	db.DeleteOptionRound(roundAddress)
}

func (db *DB) AuctionStartedRevert(vaultAddress, roundAddress string, blockNumber uint64) error {
	if err := db.RevertVaultState(vaultAddress, blockNumber); err != nil {
		return err
	}
	if err := db.RevertAllLPState(vaultAddress, blockNumber); err != nil {
		return err
	}
	if err := db.UpdateOptionRoundFields(roundAddress, map[string]interface{}{
		"available_options":  0,
		"starting_liquidity": 0,
		"state":              "Open",
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) AuctionEndedRevert(vaultAddress, roundAddress string, blockNumber uint64) error {
	if err := db.RevertVaultState(vaultAddress, blockNumber); err != nil {
		return err
	}
	if err := db.RevertAllLPState(vaultAddress, blockNumber); err != nil {
		return err
	}
	if err := db.UpdateOptionRoundFields(roundAddress, map[string]interface{}{
		"clearing_price": nil,
		"options_sold":   nil,
		"state":          "Auctioning",
	}); err != nil {
		return err
	}
	if err := db.UpdateAllOptionBuyerFields(roundAddress, map[string]interface{}{
		"tokenizable_options": 0,
		"refundable_amount":   0,
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) RoundSettledRevert(vaultAddress, roundAddress string, blockNumber uint64) error {
	if err := db.RevertVaultState(vaultAddress, blockNumber); err != nil {
		return err
	}
	if err := db.RevertAllLPState(vaultAddress, blockNumber); err != nil {
		return err
	}
	err := db.UpdateOptionRoundFields(roundAddress, map[string]interface{}{
		"settlement_price": 0,
		"total_payout":     0,
		"state":            "Running",
	})
	return err
}

func (db *DB) BidPlacedRevert(bidId, roundAddress string) error {
	err := db.DeleteBid(bidId, roundAddress)
	return err
}

func (db *DB) BidUpdatedRevert(bidId, roundAddress string, amount models.BigInt, treeNonce uint64) error {
	if err := db.tx.Model(models.Bid{}).Where("bid_id = ?", bidId).Where("round_address = ?", roundAddress).Updates(map[string]interface{}{
		"amount":     gorm.Expr("amount - ?", amount),
		"tree_nonce": treeNonce,
	}); err != nil {
		return err.Error
	}
	return nil

}
