package network

import (
	"context"
	"fmt"
	"junoplugin/models"
	"junoplugin/utils"
	"log"
	"os"
	"sync"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/rpc"
)

type Network struct {
	provider *rpc.Provider
	ctx      context.Context
}

func (n *Network) GetBlockByHash(hash string) (*rpc.BlockTxHashes, error) {
	feltString, err := utils.HexStringToFelt(hash)
	if err != nil {
		return nil, err
	}
	hashFelt := felt.FromBytes(feltString)
	block, err := n.provider.BlockWithTxHashes(n.ctx, rpc.BlockID{Hash: &hashFelt})
	if err != nil {
		return nil, err
	}
	blockTxHashes, ok := block.(*rpc.BlockTxHashes)
	if !ok {
		return nil, fmt.Errorf("unexpected block type for block %v", hash)
	}
	return blockTxHashes, nil
}
func NewNetwork() (*Network, error) {
	provider, err := rpc.NewProvider(os.Getenv("RPC_URL"))
	if err != nil {
		return nil, err
	}
	return &Network{
		provider: provider,
		ctx:      context.Background(),
	}, nil
}

func (n *Network) GetEvents(fromBlock rpc.BlockID, toBlock rpc.BlockID, address *string) (*rpc.EventChunk, error) {
	var addressFelt felt.Felt
	filter := rpc.EventFilter{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
	}
	if address != nil {
		addreshHash, err := utils.HexStringToFelt(*address)
		if err != nil {
			log.Printf("Error getting felt", err)
			return nil, err
		}
		addressFelt = felt.FromBytes(addreshHash)
		filter.Address = &addressFelt
	}

	log.Printf("Filter: %v", filter)
	input := rpc.EventsInput{
		EventFilter: filter,
		ResultPageRequest: rpc.ResultPageRequest{
			ChunkSize: 10,
		},
	}
	events, err := n.provider.Events(n.ctx, input)
	if err != nil {
		log.Printf("Error getting events", err)
		return nil, err
	}
	return events, nil
}

func (n *Network) GetBlocks(fromBlock uint64, toBlock uint64) ([]*models.StarknetBlocks, error) {
	numBlocks := toBlock - fromBlock + 1
	blocks := make([]*models.StarknetBlocks, numBlocks)

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	// Channel for errors
	errCh := make(chan error, numBlocks)
	// Semaphore to limit concurrency
	maxConcurrent := 100 // Adjust based on your needs
	sem := make(chan struct{}, maxConcurrent)

	for i := fromBlock; i <= toBlock; i++ {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(blockNum uint64, index uint64) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			log.Printf("Getting block %v", blockNum)
			block, err := n.provider.BlockWithTxHashes(n.ctx, rpc.BlockID{Number: &blockNum})
			if err != nil {
				log.Printf("Error getting block %v: %v", blockNum, err)
				errCh <- err
				return
			}

			blockTxHashes, ok := block.(*rpc.BlockTxHashes)
			if !ok {
				errCh <- fmt.Errorf("unexpected block type for block %v", blockNum)
				return
			}

			blockHeader := blockTxHashes.BlockHeader
			starknetBlock := &models.StarknetBlocks{
				BlockNumber: blockNum,
				BlockHash:   blockHeader.BlockHash.String(),
				ParentHash:  blockHeader.ParentHash.String(),
				Timestamp:   blockHeader.Timestamp,
			}

			// Store in the correct position in the result slice
			blocks[index-fromBlock] = starknetBlock
			log.Printf("Processed block %v", blockNum)
		}(i, i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errCh)

	// Check for any errors
	select {
	case err := <-errCh:
		return nil, err
	default:
		return blocks, nil
	}
}
