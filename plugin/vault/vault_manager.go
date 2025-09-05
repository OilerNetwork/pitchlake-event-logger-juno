package vault

import (
	"fmt"
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/network"
	"junoplugin/utils"
	"log"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/rpc"
)

// Manager handles vault-related operations
type Manager struct {
	db                *db.DB
	network           *network.Network
	vaultAddressesMap map[string]struct{}
	udcAddress        string
	log               *log.Logger
}

// NewManager creates a new vault manager
func NewManager(db *db.DB, network *network.Network, udcAddress string) *Manager {
	return &Manager{
		db:                db,
		network:           network,
		vaultAddressesMap: make(map[string]struct{}),
		udcAddress:        udcAddress,
		log:               log.Default(),
	}
}

// InitializeVaults initializes existing vaults from the database
func (vm *Manager) InitializeVaults(lastBlock *models.StarknetBlocks) error {
	vaultRegistry, err := vm.db.GetVaultRegistry()
	if err != nil {
		return fmt.Errorf("failed to get vault registry: %w", err)
	}

	// Initialize existing vaults
	if len(vaultRegistry) > 0 {
		for _, vault := range vaultRegistry {
			if vault.LastBlockIndexed != &lastBlock.BlockHash {
				if err := vm.CatchupVault(vault.Address, lastBlock.BlockNumber); err != nil {
					return fmt.Errorf("failed to catchup vault %s: %w", vault.Address, err)
				}
				vm.vaultAddressesMap[vault.Address] = struct{}{}
			}
		}
	}

	vm.log.Printf("Vault addresses: %v", vm.vaultAddressesMap)
	vm.log.Printf("Last block: %v", lastBlock)

	return nil
}

// InitializeVault initializes a new vault
func (vm *Manager) InitializeVault(vault models.VaultRegistry) error {
	deployBlockHash, err := utils.HexStringToFelt(vault.DeployedAt)
	if err != nil {
		vm.log.Println("Error getting felt", err)
		return err
	}
	hash := felt.FromBytes(deployBlockHash)

	hash.SetString(vault.DeployedAt)
	deployBlock := rpc.BlockID{
		Hash: &hash,
	}
	vm.log.Printf("Deploy block: %v", deployBlock)

	events, err := vm.network.GetEvents(deployBlock, deployBlock, nil)
	if err != nil {
		vm.log.Println("Error getting events", err)
		return err
	}

	vm.db.BeginTx()
	err = vm.processDeploymentBlockEvents(events, vault)
	if err != nil {
		vm.log.Println("Error processing deployment events", err)
		vm.db.RollbackTx()
		return err
	}
	vm.db.CommitTx()

	vm.log.Printf("Processing catchup vault")
	err = vm.CatchupVault(vault.Address, 0) // Will be set to proper block number by caller
	if err != nil {
		return err
	}
	vm.vaultAddressesMap[vault.Address] = struct{}{}
	return nil
}

// CatchupVault catches up a vault to a specific block
func (vm *Manager) CatchupVault(address string, toBlock uint64) error {
	vaultRegistry, err := vm.db.GetVaultRegistryByAddress(address)
	if err != nil {
		vm.log.Println("Error getting vault registry", err)
		return err
	}
	vm.log.Printf("Vault registry: %v", vaultRegistry)

	var fromBlock *rpc.BlockID
	vm.log.Printf("Vault registry: %v", vaultRegistry.LastBlockIndexed)
	if vaultRegistry.LastBlockIndexed == nil {
		err = vm.InitializeVault(vaultRegistry)
		if err != nil {
			vm.log.Println("Error initializing vault", err)
			return err
		}
		deployBlock, err := vm.network.GetBlockByHash(vaultRegistry.DeployedAt)
		if err != nil {
			vm.log.Println("Error getting deploy block", err)
			return err
		}
		nextBlockNumber := deployBlock.Number + 1
		fromBlock = &rpc.BlockID{Number: &nextBlockNumber}
	} else {
		vm.log.Printf("Last block indexed: %v", vaultRegistry)
		hash := *vaultRegistry.LastBlockIndexed
		nextBlock, err := vm.db.GetNextBlock(hash)
		if err != nil {
			vm.log.Println("Block not found, wait to catch up", err)
			return nil
		}
		fromBlock = &rpc.BlockID{Number: &nextBlock.BlockNumber}
	}
	vm.log.Printf("From block: %v", fromBlock)

	if *fromBlock.Number > toBlock {
		vm.log.Println("From block is greater than to block, wait to catch up")
		return nil
	}

	events, err := vm.network.GetEvents(*fromBlock, rpc.BlockID{Number: &toBlock}, &address)
	if err != nil {
		vm.log.Println("Error getting events", err)
		return err
	}

	for _, event := range events.Events {
		coreEvent := core.Event{
			From: event.FromAddress,
			Keys: event.Keys,
			Data: event.Data,
		}
		err := vm.ProcessVaultEvent(event.TransactionHash.String(), address, &coreEvent, event.BlockNumber)
		if err != nil {
			vm.log.Println("Error processing vault event", err)
			return err
		}
	}
	return nil
}

// IsVaultAddress checks if an address is a tracked vault
func (vm *Manager) IsVaultAddress(address string) bool {
	_, exists := vm.vaultAddressesMap[address]
	return exists
}

// GetVaultAddresses returns all tracked vault addresses
func (vm *Manager) GetVaultAddresses() map[string]struct{} {
	return vm.vaultAddressesMap
}

// processDeploymentBlockEvents processes events from the deployment block
func (vm *Manager) processDeploymentBlockEvents(events *rpc.EventChunk, vault models.VaultRegistry) error {
	eventNameHash := utils.Keccak256("ContractDeployed")
	for index, event := range events.Events {
		vm.log.Printf("index: %v", index)
		vm.log.Printf("Event from address: %v", event.FromAddress.String())
		vm.log.Printf("UDC address: %v", vm.udcAddress)

		if eventNameHash == event.Keys[0].String() && event.FromAddress.String() == vm.udcAddress {
			vm.log.Printf("Match")
			address := utils.FeltToHexString(event.Data[0].Bytes())
			vm.log.Printf("Address: %v", address)
			vm.log.Printf("Vault address: %v", vault.Address)

			normalizedVaultAddress, err := utils.NormalizeHexAddress(vault.Address)
			if err != nil {
				vm.log.Printf("Error normalizing address: %v", err)
				return err
			}

			if address == normalizedVaultAddress {
				txHash := utils.FeltToHexString(event.TransactionHash.Bytes())
				eventKeys := utils.FeltArrayToStringArrays(event.Keys)
				eventData := utils.FeltArrayToStringArrays(event.Data)

				vm.db.StoreEvent(txHash, address, event.BlockNumber, "ContractDeployed", eventKeys, eventData)
				break
			}
		}
	}

	// Process other vault events in this block
	for _, event := range events.Events {
		junoEvent := core.Event{
			From: event.FromAddress,
			Keys: event.Keys,
			Data: event.Data,
		}
		normalizedVaultAddress, err := utils.NormalizeHexAddress(vault.Address)
		if err != nil {
			vm.log.Printf("Error normalizing address %v", err)
			return err
		}
		if utils.FeltToHexString(event.FromAddress.Bytes()) == normalizedVaultAddress {
			err := vm.ProcessVaultEvent(event.TransactionHash.String(), vault.Address, &junoEvent, event.BlockNumber)
			if err != nil {
				return err
			}
			vm.db.UpdateVaultRegistry(vault.Address, event.BlockHash.String())
		}
	}
	return nil
}

// ProcessVaultEvent processes a vault event
func (vm *Manager) ProcessVaultEvent(txHash string, vaultAddress string, event *core.Event, blockNumber uint64) error {
	// Store the event in the database
	normalizedVaultAddress, err := utils.NormalizeHexAddress(vaultAddress)
	if err != nil {
		vm.log.Printf("Error normalizing address %v", err)
		return err
	}

	eventName, err := utils.DecodeEventNameVault(event.Keys[0].String())
	if err != nil {
		vm.log.Printf("Unknown Event")
		return nil
	}

	// Store the event in the database
	eventKeys, eventData := utils.EventToStringArrays(*event)
	if err := vm.db.StoreEvent(txHash, normalizedVaultAddress, blockNumber, eventName, eventKeys, eventData); err != nil {
		return err
	}
	return nil
}
