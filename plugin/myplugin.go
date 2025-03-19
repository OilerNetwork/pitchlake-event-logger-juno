package main

import (
	"junoplugin/adaptors"
	"junoplugin/db"
	"junoplugin/models"
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	junoplugin "github.com/NethermindEth/juno/plugin"
)

// Todo: push this stuff to a config file / cmd line

//go:generate go build -buildmode=plugin -o ../../build/plugin.so ./example.go
type pitchlakePlugin struct {
	vaultHash         string
	vaultAddressesMap map[string]struct{}
	deployer          string
	udcAddress        string
	db                *db.DB
	log               *log.Logger
	junoAdaptor       *adaptors.JunoAdaptor
	cursor            uint64
}

// Important: "JunoPluginInstance" needs to be exported for Juno to load the plugin correctly
var JunoPluginInstance = pitchlakePlugin{}

// Ensure the plugin and Juno client follow the same interface
var _ junoplugin.JunoPlugin = (*pitchlakePlugin)(nil)

func (p *pitchlakePlugin) Init() error {
	dbUrl := os.Getenv("DB_URL")
	udcAddress := os.Getenv("UDC_ADDRESS")
	p.udcAddress = udcAddress
	p.vaultAddressesMap = make(map[string]struct{})
	dbClient, err := db.Init(dbUrl)
	if err != nil {
		return err
	}
	p.db = dbClient
	vaultAddresses, err := p.db.GetVaultAddresses()
	if err != nil {
		return err
	}

	//Map
	for _, vaultAddress := range vaultAddresses {
		p.vaultAddressesMap[vaultAddress] = struct{}{}
	}

	p.junoAdaptor = &adaptors.JunoAdaptor{}
	p.vaultHash = os.Getenv("VAULT_HASH")
	p.deployer = os.Getenv("DEPLOYER")
	cursor := os.Getenv("CURSOR")
	if cursor != "" {
		p.cursor, err = strconv.ParseUint(cursor, 10, 64)
		if err != nil {
			return err
		}
	}
	p.log = log.Default()

	//Add function to catch up on vaults/rounds that are not synced to currentBlock
	return nil
}

func (p *pitchlakePlugin) Shutdown() error {
	p.log.Println("Calling Shutdown() in plugin")
	p.db.Close()
	return nil
}

func (p *pitchlakePlugin) NewBlock(
	block *core.Block,
	stateUpdate *core.StateUpdate,
	newClasses map[felt.Felt]core.Class,
) error {

	p.db.Begin()
	p.log.Println("ExamplePlugin NewBlock called")
	if block.Number < p.cursor {
		log.Printf("Pre-cursor block")
		p.db.Rollback()
		return nil
	}

	for _, receipt := range block.Receipts {
		for i, event := range receipt.Events {
			fromAddress := event.From.String()
			if fromAddress == p.udcAddress {
				err := p.processUDC(receipt.TransactionHash.String(), receipt.Events, event, i, block.Number, block.Timestamp)
				if err != nil {
					p.db.Rollback()
					return err
				}
			} else {
				//HashMap processing
				if _, exists := p.vaultAddressesMap[fromAddress]; exists {
					err := p.processVaultEvent(receipt.TransactionHash.String(), fromAddress, event, block.Number, block.Timestamp)
					if err != nil {
						p.db.Rollback()
						return err
					}
				}

			}
		}
	}
	p.db.Commit()
	return nil
}

func (p *pitchlakePlugin) RevertBlock(
	from,
	to *junoplugin.BlockAndStateUpdate,
	reverseStateDiff *core.StateDiff,
) error {
	p.db.Begin()
	length := len(from.Block.Receipts)
	for i := length - 1; i >= 0; i-- {
		receipt := from.Block.Receipts[i]
		for _, event := range receipt.Events {
			fromAddress := event.From.String()
			//HashMap
			if _, exists := p.vaultAddressesMap[fromAddress]; exists {
				err := p.revertVaultEvent(fromAddress, event, from.Block.Number)
				if err != nil {
					return err
				}
			}
		}
	}
	p.db.Commit()
	return nil
}

func (p *pitchlakePlugin) processUDC(
	txHash string,
	events []*core.Event,
	event *core.Event,
	index int,
	blockNumber uint64,
	timestamp uint64,
) error {

	eventHash := adaptors.Keccak256("ContractDeployed")
	if eventHash == event.Keys[0].String() {
		address := adaptors.FeltToHexString(event.Data[0].Bytes())
		deployer := adaptors.FeltToHexString(event.Data[1].Bytes())
		classHash := adaptors.FeltToHexString(event.Data[3].Bytes())
		//ClassHash and deployer filter, may use other filters here

		if classHash == p.vaultHash && deployer == p.deployer {
			eventKeys, eventData := adaptors.EventToStringArrays(*event)
			p.db.StoreEvent(txHash, address, blockNumber, timestamp, "ContractDeployed", eventKeys, eventData)
			if err := p.processVaultEvent(txHash, address, events[index-1], blockNumber, timestamp); err != nil {
				return err
			}
		}

	}
	return nil
}

func (p *pitchlakePlugin) processVaultEvent(
	txHash string,
	vaultAddress string,
	event *core.Event,
	blockNumber uint64,
	timestamp uint64,
) error {

	// Store the event in the database
	eventName, err := adaptors.DecodeEventNameVault(event.Keys[0].String())
	if err != nil {
		log.Printf("Unknown Event")
		return nil
	}

	// Store the event in the database
	eventKeys, eventData := adaptors.EventToStringArrays(*event)
	if err := p.db.StoreEvent(txHash, vaultAddress, blockNumber, timestamp, eventName, eventKeys, eventData); err != nil {
		return err
	}
	return nil
}

func (p *pitchlakePlugin) revertVaultEvent(vaultAddress string, event *core.Event, blockNumber uint64) error {
	eventName, err := adaptors.DecodeEventNameVault(event.Keys[0].String())
	if err != nil {
		return err
	}
	switch eventName {
	case "Deposit", "Withdraw",
		"StashWithdrawn": //Add withdraw queue

		lpAddress := adaptors.FeltToHexString(event.Keys[1].Bytes())
		err = p.db.DepositOrWithdrawRevert(vaultAddress, lpAddress, blockNumber)
	case "WithdrawalQueued":
		lpAddress,
			bps,
			roundId,
			accountQueuedBefore,
			accountQueuedNow,
			vaultQueuedNow := p.junoAdaptor.WithdrawalQueued(*event)

		err = p.db.WithdrawalQueuedRevertIndex(
			lpAddress,
			vaultAddress,
			roundId,
			bps,
			accountQueuedBefore,
			accountQueuedNow,
			vaultQueuedNow,
			blockNumber,
		)
	case "OptionRoundDeployed":
		roundAddress := adaptors.FeltToHexString(event.Data[2].Bytes())
		err = p.db.DeleteOptionRound(roundAddress)

	case "AuctionStarted":
		_, _, roundAddress := p.junoAdaptor.AuctionStarted(*event)
		prevStateOptionRound, err := p.db.GetOptionRoundByAddress(roundAddress)
		if err != nil {
			return err
		}
		err = p.db.AuctionStartedRevert(prevStateOptionRound.VaultAddress, roundAddress, blockNumber)
	case "AuctionEnded":
		_, _, _, _, _, roundAddress := p.junoAdaptor.AuctionEnded(*event)
		prevStateOptionRound, err := p.db.GetOptionRoundByAddress(roundAddress)
		if err != nil {
			return err
		}
		err = p.db.AuctionEndedRevert(prevStateOptionRound.VaultAddress, roundAddress, blockNumber)

	case "OptionRoundSettled":
		_, _, roundAddress := p.junoAdaptor.OptionRoundSettled(*event)
		prevStateOptionRound, err := p.db.GetOptionRoundByAddress(roundAddress)
		if err != nil {
			return err
		}
		err = p.db.RoundSettledRevert(prevStateOptionRound.VaultAddress, roundAddress, blockNumber)
	case "BidPlaced":
		bid, _ := p.junoAdaptor.BidPlaced(*event)
		err = p.db.BidPlacedRevert(bid.BidID, bid.RoundAddress)
	case "BidUpdated":
		bidId, amount, treeNonceOld, _, roundAddress := p.junoAdaptor.BidUpdated(*event)
		err = p.db.BidUpdatedRevert(bidId, roundAddress, amount, treeNonceOld)
	case "OptionsMinted":
		buyerAddress, _, roundAddress := p.junoAdaptor.OptionsMinted(*event)
		err = p.db.UpdateOptionBuyerFields(
			buyerAddress,
			roundAddress,
			map[string]interface{}{
				"has_minted": false,
			})
	case "OptionsExercised":
		buyerAddress, _, _, _, roundAddress := p.junoAdaptor.OptionsExercised(*event)
		mintableOptionsExercised := adaptors.CombineFeltToBigInt(event.Data[3].Bytes(), event.Data[2].Bytes())

		zero := models.BigInt{
			Int: big.NewInt(0),
		}
		if mintableOptionsExercised.Cmp(zero.Int) == 1 {
			err = p.db.UpdateOptionBuyerFields(
				buyerAddress,
				roundAddress,
				map[string]interface{}{
					"has_minted": false,
				})
		}
	case "UnusedBidsRefunded":
		buyerAddress, _, roundAddress := p.junoAdaptor.UnusedBidsRefunded(*event)
		err = p.db.UpdateOptionBuyerFields(
			buyerAddress,
			roundAddress,
			map[string]interface{}{
				"has_refunded": false,
			})

	}
	if err != nil {
		return err
	}

	return nil
}
