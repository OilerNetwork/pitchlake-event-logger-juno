package db

import (
	"junoplugin/models"
	"log"
	"math/big"

	"gorm.io/gorm"
)

func (db *DB) DepositIndex(
	vaultAddress,
	lpAddress string,
	lpUnlocked, vaultUnlocked models.BigInt,
	blockNumber uint64) error {
	//Map the other parameters as well
	var newLPState = &(models.LiquidityProviderState{
		VaultAddress:    vaultAddress,
		Address:         lpAddress,
		UnlockedBalance: lpUnlocked,
		LatestBlock:     blockNumber,
	})
	if err := db.UpsertLiquidityProviderState(newLPState, blockNumber); err != nil {
		return err
	}
	if err := db.UpdateVaultFields(vaultAddress, map[string]interface{}{
		"unlocked_balance": vaultUnlocked,
		"latest_block":     blockNumber,
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) WithdrawIndex(
	vaultAddress,
	lpAddress string,
	lpUnlocked, vaultUnlocked models.BigInt,
	blockNumber uint64) error {
	//Map the other parameters as well
	if err := db.UpdateLiquidityProviderFields(vaultAddress, lpAddress, map[string]interface{}{
		"unlocked_balance": lpUnlocked,
		"latest_block":     blockNumber,
	}); err != nil {
		return err
	}
	if err := db.UpdateVaultFields(vaultAddress, map[string]interface{}{
		"unlocked_balance": vaultUnlocked,
		"latest_block":     blockNumber,
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) WithdrawalQueuedIndex(
	lpAddress, vaultAddress string,
	roundId uint64,
	bps, accountQueuedBefore, accountQueuedNow, vaultQueuedNow models.BigInt,
) error {
	vault, err := db.GetVaultByAddress(vaultAddress)
	if err != nil {
		return err
	}
	queuedLiquidity := models.QueuedLiquidity{
		Address:      lpAddress,
		RoundAddress: vault.CurrentRoundAddress,
		Bps:          bps,
		QueuedAmount: accountQueuedNow,
	}
	if err := db.UpsertQueuedLiquidity(&queuedLiquidity); err != nil {
		return err
	}
	if err := db.UpdateOptionRoundFields(vault.CurrentRoundAddress, map[string]interface{}{
		"queued_liquidity": vaultQueuedNow,
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) StashWithdrawnIndex(
	vaultAddress, lpAddress string,
	amount, vaultBalanceNow models.BigInt,
	blockNumber uint64) error {
	if err := db.UpdateLiquidityProviderFields(vaultAddress, lpAddress, map[string]interface{}{
		"stashed_balance": 0,
		"latest_block":    blockNumber,
	}); err != nil {

		return err
	}
	if err := db.UpdateVaultFields(vaultAddress, map[string]interface{}{
		"stashed_balance": vaultBalanceNow,
		"latest_block":    blockNumber,
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) RoundDeployedIndex(optionRound models.OptionRound) error {

	if err := db.CreateOptionRound(&optionRound); err != nil {
		return err
	}
	if err := db.UpdateVaultFields(optionRound.VaultAddress, map[string]interface{}{
		"current_round":         optionRound.RoundID,
		"current_round_address": optionRound.Address,
	}); err != nil {
		return err
	}
	return nil
}

func (dbc *DB) PricingDataSetIndex(
	roundAddress string,
	strikePrice, capLevel, reservePrice models.BigInt) error {
	err := dbc.UpdateOptionRoundFields(roundAddress, map[string]interface{}{
		"strike_price":  strikePrice,
		"cap_level":     capLevel,
		"reserve_price": reservePrice,
	})
	if err != nil {
		return err
	}
	return nil

}
func (dbc *DB) AuctionStartedIndex(
	vaultAddress, roundAddress string,
	blockNumber uint64,
	availableOptions, startingLiquidity models.BigInt) error {
	if err := dbc.UpdateOptionRoundFields(roundAddress, map[string]interface{}{
		"available_options":  availableOptions,
		"starting_liquidity": startingLiquidity,
		"state":              "Auctioning",
	}); err != nil {
		return err
	}
	if err := dbc.UpdateAllLiquidityProvidersBalancesAuctionStart(vaultAddress, blockNumber); err != nil {
		return err
	}
	if err := dbc.UpdateVaultBalanceAuctionStart(vaultAddress, blockNumber); err != nil {
		return err
	}
	return nil
}

func (db *DB) AuctionEndedIndex(
	prevStateOptionRound models.OptionRound,
	roundAddress string,
	blockNumber, clearingNonce uint64,
	optionsSold, clearingPrice, premiums, unsoldLiquidity models.BigInt) error {
	log.Printf("DATA %v %v %v ", unsoldLiquidity, unsoldLiquidity, optionsSold)
	if err := db.UpdateAllLiquidityProvidersBalancesAuctionEnd(
		prevStateOptionRound.VaultAddress,
		prevStateOptionRound.StartingLiquidity,
		unsoldLiquidity,
		premiums,
		blockNumber,
	); err != nil {
		return err
	}
	if err := db.UpdateVaultBalancesAuctionEnd(
		prevStateOptionRound.VaultAddress,
		unsoldLiquidity,
		premiums,
		blockNumber,
	); err != nil {
		return err
	}
	if err := db.UpdateBiddersAuctionEnd(
		roundAddress,
		clearingPrice,
		optionsSold,
		clearingNonce,
	); err != nil {
		return err
	}
	if err := db.UpdateOptionRoundAuctionEnd(
		roundAddress,
		clearingPrice,
		optionsSold,
		unsoldLiquidity,
		premiums,
	); err != nil {
		return err
	}
	return nil
}

func (db *DB) RoundSettledIndex(prevStateOptionRound models.OptionRound, roundAddress string, blockNumber uint64, settlementPrice, optionsSold, payoutPerOption models.BigInt) error {
	totalPayout := models.BigInt{Int: new(big.Int).Mul(optionsSold.Int, payoutPerOption.Int)}
	remainingLiquidity := models.BigInt{Int: new(big.Int).Sub(new(big.Int).Sub(prevStateOptionRound.StartingLiquidity.Int, prevStateOptionRound.UnsoldLiquidity.Int), totalPayout.Int)}
	remainingLiquidityStashed := *models.NewBigInt("0")
	remainingLiquidityNotStashed := models.BigInt{Int: new(big.Int).Sub(remainingLiquidity.Int, remainingLiquidityStashed.Int)}
	if prevStateOptionRound.StartingLiquidity.Cmp(remainingLiquidity.Int) != 0 && prevStateOptionRound.StartingLiquidity.Cmp(prevStateOptionRound.QueuedLiquidity.Int) != 0 {
		remainingLiquidityStashed = models.BigInt{Int: new(big.Int).Div(new(big.Int).Mul(remainingLiquidity.Int, prevStateOptionRound.QueuedLiquidity.Int), prevStateOptionRound.StartingLiquidity.Int)}
	}
	if err := db.UpdateVaultBalancesOptionSettle(
		prevStateOptionRound.VaultAddress,
		remainingLiquidityStashed,
		remainingLiquidityNotStashed,
		blockNumber); err != nil {
		return err
	}

	if err := db.UpdateAllLiquidityProvidersBalancesOptionSettle(
		prevStateOptionRound.VaultAddress,
		roundAddress,
		prevStateOptionRound.StartingLiquidity,
		remainingLiquidity,
		remainingLiquidityNotStashed,
		prevStateOptionRound.UnsoldLiquidity,
		payoutPerOption,
		optionsSold,
		blockNumber); err != nil {
		return err
	}

	if err := db.UpdateOptionRoundFields(prevStateOptionRound.Address, map[string]interface{}{
		"settlement_price":    settlementPrice,
		"payout_per_option":   payoutPerOption,
		"remaining_liquidity": remainingLiquidity,
		"state":               "Settled",
	}); err != nil {
		return err
	}
	return nil
}

func (db *DB) BidPlacedIndex(bid models.Bid, buyer models.OptionBuyer) error {
	log.Printf("NEW BID %v", bid)
	if err := db.CreateOptionBuyer(&buyer); err != nil {
		return err
	}
	if err := db.CreateBid(&bid); err != nil {
		return err
	}
	return nil
}

func (db *DB) BidUpdatedIndex(roundAddress, bidId string, price models.BigInt, treeNonce uint64) error {
	if err := db.tx.Model(models.Bid{}).Where("bid_id = ? AND round_address = ?", bidId, roundAddress).Updates(map[string]interface{}{
		"price":      gorm.Expr("price + ?", price),
		"tree_nonce": treeNonce - 1,
	}).Error; err != nil {
		return err
	}
	return nil
}
