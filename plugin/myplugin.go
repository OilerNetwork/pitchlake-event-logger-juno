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
	"github.com/NethermindEth/starknet.go/rpc"
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
	lastBlockDB       *models.StarknetBlocks
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
	if vaultAddresses != nil {
		for _, vaultAddress := range vaultAddresses {
			p.vaultAddressesMap[*vaultAddress] = struct{}{}
		}
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

	startBlock := p.lastBlockDB.BlockNumber
	for startBlock < latestBlock {
		endBlock := startBlock + 1000
		if endBlock > latestBlock {
			endBlock = latestBlock
		}
		p.db.BeginTx()
		for vault := range p.vaultAddressesMap {
			p.CatchupVault(vault, endBlock)
		}
		blocks, err := p.network.GetBlocks(startBlock, endBlock)
		if err != nil {
			return err
		}
		for _, block := range blocks {
			p.db.InsertBlock(block)
		}
		p.db.CommitTx()
		startBlock = endBlock
	}

	return nil
}

func (p *pitchlakePlugin) CatchupVault(address string, toBlock uint64) error {
	vaultRegistry, err := p.db.GetVaultRegistry(address)
	if err != nil {
		return err
	}
	var fromBlock uint64

	if vaultRegistry.LastBlockIndexed == nil {
		deploymentBlock, err := p.db.GetBlock(vaultRegistry.DeployedAt)
		if err != nil {
			return err
		}
		fromBlock = deploymentBlock.BlockNumber
	} else {
		lastBlockIndexed, err := p.db.GetBlock(*vaultRegistry.LastBlockIndexed)
		if err != nil {
			return err
		}
		fromBlock = lastBlockIndexed.BlockNumber + 1
	}
	events, err := p.network.GetEvents(rpc.BlockID{Number: &fromBlock}, rpc.BlockID{Number: &toBlock}, &address)
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

	hash := felt.Felt{}
	hash.SetString(vault.DeployedAt)
	deployBlock := rpc.BlockID{
		Hash: &hash,
	}
	events, err := p.network.GetEvents(deployBlock, deployBlock, nil)
	if err != nil {
		return err
	}
	err = p.processDeploymentEvents(events, vault)
	if err != nil {
		return err
	}
	err = p.CatchupVault(vault.Address, p.lastBlockDB.BlockNumber)
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
	if p.lastBlockDB != nil && p.lastBlockDB.BlockNumber < block.Number-1 {
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
	starknetBlock := models.StarknetBlocks{
		BlockNumber: block.Number,
		BlockHash:   block.Hash.String(),
		ParentHash:  block.ParentHash.String(),
		Status:      "MINED",
	}
	err := p.db.InsertBlock(&starknetBlock)
	if err != nil {
		p.db.RollbackTx()
		return err
	}
	p.lastBlockDB = &starknetBlock
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

func (p *pitchlakePlugin) processDeploymentEvents(
	events *rpc.EventChunk,
	vault models.VaultRegistry,
) error {

	for index, event := range events.Events {
		eventNameHash := utils.Keccak256("ContractDeployed")
		if eventNameHash == event.Keys[0].String() {
			address := utils.FeltToHexString(event.Data[0].Bytes())
			if address == vault.Address {
				txHash := utils.FeltToHexString(event.TransactionHash.Bytes())

				eventKeys := utils.FeltArrayToStringArrays(event.Keys)
				eventData := utils.FeltArrayToStringArrays(event.Data)

				p.db.StoreEvent(txHash, address, event.BlockNumber, "ContractDeployed", eventKeys, eventData)

				//Process the round deployed event
				junoEvent := core.Event{
					From: events.Events[index-1].FromAddress,
					Keys: events.Events[index-1].Keys,
					Data: events.Events[index-1].Data,
				}
				if err := p.processVaultEvent(txHash, address, &junoEvent, event.BlockNumber); err != nil {
					return err
				}
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
