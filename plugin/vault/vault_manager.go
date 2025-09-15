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
	db               *db.DB
	network          *network.Network
	vaultRegistryMap map[string]*models.VaultRegistry
	udcAddress       string
	log              *log.Logger
}

// NewManager creates a new vault manager
func NewManager(db *db.DB, network *network.Network, udcAddress string) *Manager {
	return &Manager{
		db:               db,
		network:          network,
		vaultRegistryMap: make(map[string]*models.VaultRegistry),
		udcAddress:       udcAddress,
		log:              log.Default(),
	}
}

// InitializeVaults initializes existing vaults from the database
func (vm *Manager) LoadVaultsFromRegistry(latestBlock *models.StarknetBlocks) error {
	vaultRegistry, err := vm.db.GetVaultRegistry()
	if err != nil {
		return fmt.Errorf("failed to get vault registry: %w", err)
	}

	// Catchup vaults while loading in mem to avoid reiterating later with SyncVaults call
	if len(vaultRegistry) > 0 {
		for _, vault := range vaultRegistry {
			if vault.LastBlockIndexed == nil {
				vm.InitializeVault(vault)
			}

			//Do this before the lastBlock escape
			vm.vaultRegistryMap[vault.Address] = vault

			//Escape if we don't have the latest block, we shouldn't need this if used only after initialization
			if latestBlock == nil {
				return nil
			}

			lastBlockIndexed, err := vm.db.GetBlock(*vault.LastBlockIndexed)
			if err != nil {
				return err
			}

			if lastBlockIndexed == nil || lastBlockIndexed.BlockNumber < latestBlock.BlockNumber-1 {
				if err := vm.CatchupVault(*vault, latestBlock.BlockNumber); err != nil {
					return fmt.Errorf("failed to catchup vault %s: %w", vault.Address, err)
				}

			}
		}
	}

	vm.log.Printf("Vault addresses: %v", vm.vaultRegistryMap)
	vm.log.Printf("Last block: %v", latestBlock)

	return nil
}

func (vm *Manager) SyncVaults(head *models.StarknetBlocks) error {
	vaultRegistry := &vm.vaultRegistryMap
	// if err != nil {
	// 	return fmt.Errorf("failed to get vault registry: %w", err)
	// }
	for _, vault := range *vaultRegistry {
		if vault.LastBlockIndexed == nil {
			vm.InitializeVault(vault)
		}
		if head == nil {
			log.Printf("No last block found, starting node to find current block")
			return nil
		}
		if *vault.LastBlockIndexed != head.BlockHash {
			if err := vm.CatchupVault(*vault, head.BlockNumber); err != nil {
				return fmt.Errorf("failed to catchup vault %s: %w", vault.Address, err)
			}
		}
	}

	return nil
}

// InitializeVault initializes a new vault
func (vm *Manager) InitializeVault(vault *models.VaultRegistry) error {
	deployBlockHash, err := utils.HexStringToFelt(vault.DeployedAt)
	if err != nil {
		vm.log.Println("Error getting felt", err)
		return err
	}

	hash := felt.FromBytes(deployBlockHash)

	// hash.SetString(vault.DeployedAt)
	deployBlock := rpc.BlockID{
		Hash: &hash,
	}
	vm.log.Printf("Deploy block: %v", deployBlock)

	events, err := vm.network.GetEvents(deployBlock, deployBlock, nil)
	if err != nil {
		vm.log.Println("Error getting events", err)
		return err
	}
	log.Printf("events list %v", len(events.Events))

	vm.db.BeginTx()
	err = vm.processDeploymentBlockEvents(events, vault)
	if err != nil {
		vm.log.Println("Error processing deployment events", err)
		vm.db.RollbackTx()
		return err
	}

	// Save the block as well in db if it doesn't exist
	// block, err := vm.db.GetBlock(vault.DeployedAt)
	// if err != nil {
	// 	return err
	// }

	// if block == nil {
	// 	//get block from network
	// 	networkBlock, err := vm.network.GetBlockByHash(vault.DeployedAt)

	// 	if err != nil {
	// 		return nil
	// 	}

	// 	starknetBlock := models.RPCBlockToStarknetBlock(networkBlock)

	// 	err = vm.db.InsertBlock(starknetBlock)
	// 	if err != nil {
	// 		vm.db.RollbackTx()
	// 		vm.log.Println("Error inserting block", err)
	// 		return err
	// 	}
	// }
	vm.db.CommitTx()
	return nil
}

// CatchupVault catches up a vault to a specific block
func (vm *Manager) CatchupVault(vault models.VaultRegistry, toBlock uint64) error {

	var fromBlock *rpc.BlockID
	vm.log.Printf("Vault registry: %v", vault.LastBlockIndexed)
	vm.log.Printf("Last block indexed: %v", vault)
	hash := *vault.LastBlockIndexed

	nextBlock, err := vm.db.GetNextBlock(hash)
	if err != nil {
		return err
	}

	if nextBlock == nil {
		//get last block from db
		lastBlock, err := vm.db.GetBlock(hash)
		if err != nil {
			return err
		}

		if lastBlock == nil {
			lastBlockNetwork, err := vm.network.GetBlockByHash(hash)
			if err != nil {
				return err
			}
			nextBlocks, err := vm.network.GetBlocks(lastBlockNetwork.BlockHeader.Number+1, lastBlockNetwork.BlockHeader.Number+1)
			if len(nextBlocks) == 0 {
				vm.log.Printf("No blocks returned for block number: %d", lastBlockNetwork.BlockHeader.Number+1)
				return fmt.Errorf("no block found at number %d", lastBlockNetwork.BlockHeader.Number+1)
			}
			log.Printf("nextBlocks %v", nextBlocks)
			nextBlock = nextBlocks[0]
			//create new int
			fromBlock = &rpc.BlockID{Number: &nextBlock.BlockNumber}
		}

	}

	if *fromBlock.Number > toBlock {
		vm.log.Println("From block is greater than to block, wait to catch up")
		return nil
	}

	events, err := vm.network.GetEvents(*fromBlock, rpc.BlockID{Number: &toBlock}, &vault.Address)
	if err != nil {
		vm.log.Println("Error getting events", err)
		return err
	}

	vm.db.BeginTx()
	for _, event := range events.Events {
		coreEvent := core.Event{
			From: event.FromAddress,
			Keys: event.Keys,
			Data: event.Data,
		}
		err := vm.ProcessVaultEvent(event.TransactionHash.String(), vault.Address, &coreEvent, event.BlockNumber, *event.BlockHash)
		if err != nil {
			vm.log.Println("Error processing vault event", err)
			vm.db.RollbackTx()
			return err

		}
	}

	//Store block as well, nextBlock should never be null by the time we reach here
	if err := vm.db.InsertBlock(nextBlock); err != nil {
		vm.db.RollbackTx()
		return err
	}
	startBlockHash := hash              // fromBlock hash
	endBlockHash := nextBlock.BlockHash // toBlock hash

	err = vm.db.StoreVaultCatchupEvent(vault.Address, startBlockHash, endBlockHash)
	if err != nil {
		vm.log.Printf("Error storing vault catchup event: %v", err)
		vm.db.RollbackTx()
		return err
	}
	vm.db.CommitTx()

	// Send vault catchup event after successful catchup
	vm.log.Printf("Stored and notified vault catchup event for vault %s, blocks %s-%s", vault.Address, startBlockHash, endBlockHash)
	return nil
}

// IsVaultAddress checks if an address is a tracked vault
func (vm *Manager) IsVaultAddress(address string) bool {
	_, exists := vm.vaultRegistryMap[address]
	return exists
}

// GetVaultAddresses returns all tracked vault addresses
func (vm *Manager) GetVaultAddresses() map[string]struct{} {

	//Hacky faster fix, instead update the usage of this function to avoid translating here
	addresses := make(map[string]struct{})
	for _, vault := range vm.vaultRegistryMap {
		addresses[vault.Address] = struct{}{}
	}
	return addresses
}

// processDeploymentBlockEvents processes events from the deployment block
func (vm *Manager) processDeploymentBlockEvents(events *rpc.EventChunk, vault *models.VaultRegistry) error {
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
				blockHash := utils.FeltToHexString(event.BlockHash.Bytes())

				vm.db.StoreEvent(txHash, address, event.BlockNumber, blockHash, "ContractDeployed", eventKeys, eventData)
				vault.LastBlockIndexed = &blockHash
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
			err := vm.ProcessVaultEvent(event.TransactionHash.String(), vault.Address, &junoEvent, event.BlockNumber, *event.BlockHash)
			if err != nil {
				return err
			}
			vm.db.UpdateVaultRegistry(vault.Address, event.BlockHash.String())
		}
	}
	return nil
}

// ProcessVaultEvent processes a vault event
func (vm *Manager) ProcessVaultEvent(txHash string, vaultAddress string, event *core.Event, blockNumber uint64, blockHash felt.Felt) error {
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
	blockHashNormalized := utils.FeltToHexString(blockHash.Bytes())
	if err := vm.db.StoreEvent(txHash, normalizedVaultAddress, blockNumber, blockHashNormalized, eventName, eventKeys, eventData); err != nil {
		return err
	}
	return nil
}
