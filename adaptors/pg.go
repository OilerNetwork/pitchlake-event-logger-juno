package adaptors

import (
	"junoplugin/models"
	"log"
	"math/big"

	"github.com/NethermindEth/juno/core"
)

type JunoAdaptor struct {
}

func (p *JunoAdaptor) ContractDeployed(event core.Event) (string, string, string, models.BigInt, models.BigInt, uint64, uint64, uint64) {

	fossilClientAddress := FeltToHexString(event.Data[5].Bytes())
	ethAddress := FeltToHexString(event.Data[6].Bytes())
	optionRoundClassHash := FeltToHexString(event.Data[7].Bytes())
	alpha := FeltToBigInt(event.Data[8].Bytes())
	strikeLevel := FeltToBigInt(event.Data[9].Bytes())
	roundTransitionDuration := event.Data[10].Uint64()
	auctionDuration := event.Data[11].Uint64()
	roundDuration := event.Data[12].Uint64()
	return fossilClientAddress,
		ethAddress,
		optionRoundClassHash,
		alpha,
		strikeLevel,
		roundTransitionDuration,
		auctionDuration,
		roundDuration
}
func (p *JunoAdaptor) PricingDataSet(event core.Event) (models.BigInt, models.BigInt, models.BigInt, string) {
	strikePrice := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	capLevel := FeltToBigInt(event.Data[2].Bytes())
	reservePrice := CombineFeltToBigInt(event.Data[4].Bytes(), event.Data[3].Bytes())
	roundAddress := FeltToHexString(event.Data[5].Bytes())
	return strikePrice, capLevel, reservePrice, roundAddress
}
func (p *JunoAdaptor) DepositOrWithdraw(event core.Event) (string, models.BigInt, models.BigInt) {
	lpAddress := FeltToHexString(event.Keys[1].Bytes())
	lpUnlocked := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	vaultUnlocked := CombineFeltToBigInt(event.Data[5].Bytes(), event.Data[4].Bytes())
	return lpAddress, lpUnlocked, vaultUnlocked
}

func (p *JunoAdaptor) WithdrawalQueued(event core.Event) (string, models.BigInt, uint64, models.BigInt, models.BigInt, models.BigInt) {
	lpAddress := FeltToHexString(event.Keys[1].Bytes())
	bps := FeltToBigInt(event.Data[0].Bytes())
	roundId := event.Data[1].Uint64()
	accountQueuedNow := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	vaultQueuedNow := CombineFeltToBigInt(event.Data[5].Bytes(), event.Data[4].Bytes())

	//Change this when using new cont
	accountQueuedBefore := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())

	return lpAddress, bps, roundId, accountQueuedBefore, accountQueuedNow, vaultQueuedNow
}

func (p *JunoAdaptor) StashWithdrawn(event core.Event) (string, models.BigInt, models.BigInt) {
	lpAddress := FeltToHexString(event.Keys[1].Bytes())
	amount := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	vaultStashed := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	return lpAddress, amount, vaultStashed
}

func (p *JunoAdaptor) RoundDeployed(event core.Event) models.OptionRound {

	log.Printf("event %v", event)
	vaultAddress :=
		event.From.String()
	roundId := FeltToBigInt(event.Data[0].Bytes())
	roundAddress := FeltToHexString(event.Data[1].Bytes())
	startingBlock := event.Data[2].Uint64()
	endingBlock := event.Data[3].Uint64()
	settlementDate := event.Data[4].Uint64()
	strikePrice := CombineFeltToBigInt(event.Data[6].Bytes(), event.Data[5].Bytes())
	capLevel := FeltToBigInt(event.Data[7].Bytes())
	reservePrice := CombineFeltToBigInt(event.Data[9].Bytes(), event.Data[8].Bytes())
	optionRound := models.OptionRound{
		RoundID:        roundId,
		Address:        roundAddress,
		VaultAddress:   vaultAddress,
		StartDate:      startingBlock,
		EndDate:        endingBlock,
		SettlementDate: settlementDate,
		StrikePrice:    strikePrice,
		CapLevel:       capLevel,
		ReservePrice:   reservePrice,
		State:          "Open",
	}
	return optionRound

}

func (p *JunoAdaptor) AuctionStarted(event core.Event) (models.BigInt, models.BigInt, string) {

	startingLiquidity := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	availableOptions := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	roundAddress := FeltToHexString(event.Data[5].Bytes())
	return availableOptions, startingLiquidity, roundAddress
}

func (p *JunoAdaptor) AuctionEnded(event core.Event) (models.BigInt, models.BigInt, models.BigInt, uint64, models.BigInt, string) {
	optionsSold := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	clearingPrice := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	unsoldLiquidity := CombineFeltToBigInt(event.Data[5].Bytes(), event.Data[4].Bytes())
	clearingNonce := event.Data[6].Uint64()
	premiums := models.BigInt{Int: new(big.Int).Mul(optionsSold.Int, clearingPrice.Int)}
	roundAddress := FeltToHexString(event.Data[7].Bytes())
	return optionsSold, clearingPrice, unsoldLiquidity, clearingNonce, premiums, roundAddress
}

func (p *JunoAdaptor) OptionRoundSettled(event core.Event) (models.BigInt, models.BigInt, string) {
	settlementPrice := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	payoutPerOption := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	roundAddress := FeltToHexString(event.Data[5].Bytes())
	return settlementPrice, payoutPerOption, roundAddress
}

func (p *JunoAdaptor) BidPlaced(event core.Event) (models.Bid, models.OptionBuyer) {
	bidId := event.Data[0].String()
	bidAmount := CombineFeltToBigInt(event.Data[2].Bytes(), event.Data[1].Bytes())
	bidPrice := CombineFeltToBigInt(event.Data[4].Bytes(), event.Data[3].Bytes())
	treeNonce := event.Data[6].Uint64() - 1
	roundAddress := FeltToHexString(event.Data[6].Bytes())

	bid := models.Bid{
		BuyerAddress: FeltToHexString(event.Keys[1].Bytes()),
		BidID:        bidId,
		RoundAddress: roundAddress,
		Amount:       bidAmount,
		Price:        bidPrice,
		TreeNonce:    treeNonce,
	}

	buyer := models.OptionBuyer{
		Address:      FeltToHexString(event.Keys[1].Bytes()),
		RoundAddress: event.From.String(),
	}

	return bid, buyer
}

func (p *JunoAdaptor) BidUpdated(event core.Event) (string, models.BigInt, uint64, uint64, string) {
	bidId := event.Data[0].String()
	price := CombineFeltToBigInt(event.Data[2].Bytes(), event.Data[1].Bytes())
	treeNonceOld := event.Data[3].Uint64()
	treeNonceNew := event.Data[4].Uint64()
	roundAddress := FeltToHexString(event.Data[5].Bytes())
	return bidId, price, treeNonceOld, treeNonceNew, roundAddress
}

func (p *JunoAdaptor) OptionsMinted(event core.Event) (string, models.BigInt, string) {
	buyerAddress := FeltToHexString(event.Keys[1].Bytes())
	mintedAmount := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	roundAddress := FeltToHexString(event.Data[2].Bytes())
	return buyerAddress, mintedAmount, roundAddress
}

func (p *JunoAdaptor) OptionsExercised(event core.Event) (string, models.BigInt, models.BigInt, models.BigInt, string) {
	buyerAddress := FeltToHexString(event.Keys[1].Bytes())
	totalOptionsExercised := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	mintableOptionsExercised := CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())
	exercisedAmount := CombineFeltToBigInt(event.Data[5].Bytes(), event.Data[4].Bytes())
	roundAddress := FeltToHexString(event.Data[7].Bytes())
	return buyerAddress, totalOptionsExercised, mintableOptionsExercised, exercisedAmount, roundAddress
}

func (p *JunoAdaptor) UnusedBidsRefunded(event core.Event) (string, models.BigInt, string) {
	buyerAddress := FeltToHexString(event.Keys[1].Bytes())
	refundedAmount := CombineFeltToBigInt(event.Data[1].Bytes(), event.Data[0].Bytes())
	roundAddress := FeltToHexString(event.Data[2].Bytes())
	return buyerAddress, refundedAmount, roundAddress
}
