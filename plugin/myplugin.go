package main

import (
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/network"
	"junoplugin/utils"
	"log"
	"os"
	"sync"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	junoplugin "github.com/NethermindEth/juno/plugin"
)

// Todo: push this stuff to a config file / cmd line

type Vault struct {
	Address    string
	DeployedAt uint64
	LastBlock  uint64
}

//go:generate go build -buildmode=plugin -o ../../build/plugin.so ./example.go
type pitchlakePlugin struct {
	vaultHash         string
	vaultAddressesMap map[string]struct{}
	deployer          string
	udcAddress        string
	db                *db.DB
	log               *log.Logger
	network           *network.Network
	lastBlockDB       uint64
	mu                sync.Mutex
	channel           chan models.VaultRegistry
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

	p.vaultHash = os.Getenv("VAULT_HASH")
	p.deployer = os.Getenv("DEPLOYER")
	p.log = log.Default()
	p.network, err = network.NewNetwork()
	p.lastBlockDB, err = p.db.GetLastBlock()
	p.channel = make(chan models.VaultRegistry)
	if err != nil {
		return err
	}
	go p.Listener()
	//Add function to catch up on vaults/rounds that are not synced to currentBlock
	return nil
}

func (p *pitchlakePlugin) CatchupIndexer(latestBlock uint64) error {

	for vault := range p.vaultAddressesMap {
		p.CatchupVault(vault, latestBlock)
	}
	return nil

}

func (p *pitchlakePlugin) CatchupVault(address string, latestBlock uint64) error {
	lastBlock, err := p.db.GetLastBlockVault(address)
	if err != nil {
		return err
	}
	events, err := p.network.GetEvents(lastBlock, latestBlock, address)
	if err != nil {
		return err
	}

	for _, event := range events.Events {
		coreEvent := core.Event{
			From: event.FromAddress,
			Keys: event.Keys,
			Data: event.Data,
		}
		p.processVaultEvent(event.TransactionHash.String(), address, &coreEvent, event.BlockNumber)
	}
	return nil
}

func (p *pitchlakePlugin) InitializeVault(vault models.VaultRegistry) error {
	p.mu.Lock()
	err := p.db.InsertVaultRegistry(vault)
	if err != nil {
		return err
	}
	err = p.CatchupVault(vault.Address, p.lastBlockDB)
	if err != nil {
		return err
	}
	p.vaultAddressesMap[vault.Address] = struct{}{}
	p.mu.Unlock()
	return nil
}

func (p *pitchlakePlugin) Shutdown() error {
	p.log.Println("Calling Shutdown() in plugin")
	p.db.Shutdown()
	return nil
}

func (p *pitchlakePlugin) Listener() error {

	p.db.ListenerNewVault(p.channel)
	for {
		select {
		case vault := <-p.channel:
			p.InitializeVault(vault)
		}
	}

}

func (p *pitchlakePlugin) NewBlock(
	block *core.Block,
	stateUpdate *core.StateUpdate,
	newClasses map[felt.Felt]core.Class,
) error {

	p.mu.Lock()
	if p.lastBlockDB != 0 && p.lastBlockDB < block.Number-1 {
		p.CatchupIndexer(block.Number)
	}

	p.db.BeginTx()
	p.log.Println("ExamplePlugin NewBlock called")

	for _, receipt := range block.Receipts {
		for _, event := range receipt.Events {
			fromAddress := event.From.String()
			if _, exists := p.vaultAddressesMap[fromAddress]; exists {
				err := p.processVaultEvent(receipt.TransactionHash.String(), fromAddress, event, block.Number)
				if err != nil {
					p.db.RollbackTx()
					return err
				}
			}
		}
	}
	p.db.CommitTx()
	p.mu.Unlock()
	return nil
}

func (p *pitchlakePlugin) RevertBlock(
	from,
	to *junoplugin.BlockAndStateUpdate,
	reverseStateDiff *core.StateDiff,
) error {
	err := p.db.RevertBlock(from.Block.Number, from.Block.Hash.String())
	if err != nil {
		return err
	}

	// p.db.Begin()
	// length := len(from.Block.Receipts)
	// for i := length - 1; i >= 0; i-- {
	// 	receipt := from.Block.Receipts[i]
	// 	for _, event := range receipt.Events {
	// 		fromAddress := event.From.String()
	// 		//HashMap
	// 		if _, exists := p.vaultAddressesMap[fromAddress]; exists {
	// 			err := p.revertVaultEvent(fromAddress, event, from.Block.Number)
	// 			if err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	// }
	// p.db.Commit()
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

	eventHash := utils.Keccak256("ContractDeployed")
	if eventHash == event.Keys[0].String() {
		address := utils.FeltToHexString(event.Data[0].Bytes())
		deployer := utils.FeltToHexString(event.Data[1].Bytes())
		classHash := utils.FeltToHexString(event.Data[3].Bytes())
		//ClassHash and deployer filter, may use other filters here

		if classHash == p.vaultHash && deployer == p.deployer {
			eventKeys, eventData := utils.EventToStringArrays(*event)
			p.db.StoreEvent(txHash, address, blockNumber, "ContractDeployed", eventKeys, eventData)
			if err := p.processVaultEvent(txHash, address, events[index-1], blockNumber); err != nil {
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
) error {

	// Store the event in the database
	eventName, err := utils.DecodeEventNameVault(event.Keys[0].String())
	if err != nil {
		log.Printf("Unknown Event")
		return nil
	}

	// Store the event in the database
	eventKeys, eventData := utils.EventToStringArrays(*event)
	if err := p.db.StoreEvent(txHash, vaultAddress, blockNumber, eventName, eventKeys, eventData); err != nil {
		return err
	}
	return nil
}
