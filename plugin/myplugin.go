package main

import (
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/network"
	"junoplugin/utils"
	"log"
	"os"
	"strconv"
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
	vaultAddressesMap map[string]struct{}
	udcAddress        string
	db                *db.DB
	log               *log.Logger
	network           *network.Network
	lastBlockDB       *models.StarknetBlocks
	mu                sync.Mutex
	channel           chan models.VaultRegistry
	cursor            uint64
}

// Important: "JunoPluginInstance" needs to be exported for Juno to load the plugin correctly
var JunoPluginInstance = pitchlakePlugin{}

// Ensure the plugin and Juno client follow the same interface
var _ junoplugin.JunoPlugin = (*pitchlakePlugin)(nil)

func (p *pitchlakePlugin) Init() error {
	dbUrl := os.Getenv("DB_URL")
	udcAddress := os.Getenv("UDC_ADDRESS")
	cursor := os.Getenv("CURSOR")

	var err error
	dbClient, err := db.Init(dbUrl)
	if err != nil {
		return err
	}
	p.db = dbClient

	p.network, err = network.NewNetwork()
	if err != nil {
		return err
	}
	if cursor != "" {
		p.cursor, err = strconv.ParseUint(cursor, 10, 64)
		if err != nil {
			return err
		}
	}
	p.lastBlockDB, err = p.db.GetLastBlock()
	if err != nil {
		return err
	}
	p.udcAddress = udcAddress
	p.vaultAddressesMap = make(map[string]struct{})

	vaultRegistry, err := p.db.GetVaultRegistry()
	if err != nil {
		return err
	}

	//Map
	if len(vaultRegistry) > 0 {
		for _, vault := range vaultRegistry {
			if vault.LastBlockIndexed != &p.lastBlockDB.BlockHash {
				p.CatchupVault(vault.Address, p.lastBlockDB.BlockNumber)
				p.vaultAddressesMap[vault.Address] = struct{}{}
			}
		}
	}

	p.log = log.Default()
	p.channel = make(chan models.VaultRegistry)
	if err != nil {
		return err
	}
	log.Printf("Vault addresses: %v", p.vaultAddressesMap)
	log.Printf("Last block: %v", p.lastBlockDB)
	log.Printf("Cursor: %v", p.cursor)
	go p.Listener()
	return nil
}

func (p *pitchlakePlugin) CatchupIndexer(latestBlock uint64) error {

	startBlock := p.lastBlockDB.BlockNumber
	for startBlock < latestBlock {
		endBlock := startBlock + 1000
		endBlock = min(endBlock, latestBlock)
		for vault := range p.vaultAddressesMap {
			err := p.CatchupVault(vault, endBlock)
			if err != nil {
				p.log.Println("Error catching up vault", err)
				return err
			}
		}
		p.log.Println("Catching up indexer from", startBlock, "to", endBlock)
		blocks, err := p.network.GetBlocks(startBlock, endBlock)
		if err != nil {
			p.log.Println("Error getting blocks", err)
			return err
		}
		for _, block := range blocks {
			err := p.db.InsertBlock(block)
			if err != nil {
				p.log.Println("Error inserting block", err)
				return err
			}
		}
		startBlock = endBlock
	}

	return nil
}

func (p *pitchlakePlugin) CatchupVault(address string, toBlock uint64) error {
	vaultRegistry, err := p.db.GetVaultRegistryByAddress(address)
	if err != nil {
		log.Println("Error getting vault registry", err)
		return err
	}
	log.Printf("Vault registry: %v", vaultRegistry)

	var fromBlock rpc.BlockID
	if vaultRegistry.LastBlockIndexed == nil {
		err = p.InitializeVault(vaultRegistry)
		if err != nil {
			log.Println("Error initializing vault", err)
			return err

		}
	} else {
		nextBlock, err := p.db.GetNextBlock(*vaultRegistry.LastBlockIndexed)
		if err != nil {
			log.Println("Error getting block", err)
			return err
		}
		fromBlock.Number = &nextBlock.BlockNumber
	}
	log.Printf("From block: %v", fromBlock)

	events, err := p.network.GetEvents(fromBlock, rpc.BlockID{Number: &toBlock}, &address)
	if err != nil {
		log.Println("Error getting events", err)
		return err
	}
	for _, event := range events.Events {
		coreEvent := core.Event{
			From: event.FromAddress,
			Keys: event.Keys,
			Data: event.Data,
		}
		err := p.processVaultEvent(event.TransactionHash.String(), address, &coreEvent, event.BlockNumber)
		if err != nil {
			p.log.Println("Error processing vault event", err)
			return err
		}
	}
	return nil
}

func (p *pitchlakePlugin) InitializeVault(vault models.VaultRegistry) error {

	deployBlockHash, err := utils.HexStringToFelt(vault.DeployedAt)
	if err != nil {
		log.Println("Error getting felt", err)
		return err
	}
	hash := felt.FromBytes(deployBlockHash)

	hash.SetString(vault.DeployedAt)
	deployBlock := rpc.BlockID{
		Hash: &hash,
	}
	log.Printf("Deploy block: %v", deployBlock)
	events, err := p.network.GetEvents(deployBlock, deployBlock, &p.udcAddress)
	if err != nil {
		log.Println("Error getting events", err)
		return err
	}
	p.db.BeginTx()
	err = p.processDeploymentEvents(events, vault)
	if err != nil {
		log.Println("Error processing deployment events", err)
		p.db.RollbackTx()
		return err
	}
	p.db.CommitTx()
	log.Printf("Processing catchup vault")
	err = p.CatchupVault(vault.Address, p.lastBlockDB.BlockNumber)
	if err != nil {
		return err
	}
	p.vaultAddressesMap[vault.Address] = struct{}{}
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

	if block.Number < p.cursor {
		return nil
	}
	p.mu.Lock()
	if p.lastBlockDB != nil && p.lastBlockDB.BlockNumber < block.Number-1 {
		err := p.CatchupIndexer(block.Number)
		if err != nil {
			p.log.Println("Error catching up indexer", err)
			return err
		}
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
					p.log.Println("Error processing vault event", err)
					return err
				}
			}
		}
	}
	starknetBlock := models.StarknetBlocks{
		BlockNumber: block.Number,
		BlockHash:   block.Hash.String(),
		ParentHash:  block.ParentHash.String(),
		Timestamp:   block.Timestamp,
		Status:      "MINED",
	}
	err := p.db.InsertBlock(&starknetBlock)
	if err != nil {
		p.db.RollbackTx()
		p.log.Println("Error inserting block", err)
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

	eventNameHash := utils.Keccak256("ContractDeployed")
	for index, event := range events.Events {

		log.Printf("index: %v", index)
		if eventNameHash == event.Keys[0].String() {
			log.Printf("Match")
			address := utils.FeltToHexString(event.Data[0].Bytes())
			log.Printf("Address: %v", address)
			log.Printf("Vault address: %v", vault.Address)
			normalizedVaultAddress, err := utils.NormalizeHexAddress(vault.Address)
			if err != nil {
				log.Printf("Error normalizing address", err)
				return err
			}
			if address == normalizedVaultAddress {
				txHash := utils.FeltToHexString(event.TransactionHash.Bytes())

				eventKeys := utils.FeltArrayToStringArrays(event.Keys)
				eventData := utils.FeltArrayToStringArrays(event.Data)

				p.db.StoreEvent(txHash, address, event.BlockNumber, "ContractDeployed", eventKeys, eventData)

				//Process the round deployed event
				vaultEvents, err := p.network.GetEvents(
					rpc.BlockID{Number: &event.BlockNumber},
					rpc.BlockID{Number: &event.BlockNumber},
					&vault.Address,
				)
				if err != nil {
					log.Printf("Error getting events", err)
					return err
				}
				for _, event := range vaultEvents.Events {
					junoEvent := core.Event{
						From: event.FromAddress,
						Keys: event.Keys,
						Data: event.Data,
					}
					err := p.processVaultEvent(txHash, address, &junoEvent, event.BlockNumber)
					if err != nil {
						return err
					}
					p.db.UpdateVaultRegistry(vault.Address, event.BlockHash.String())
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
