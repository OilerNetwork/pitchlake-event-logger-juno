package network

import (
	"context"
	"junoplugin/models"
	"os"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/rpc"
)

type Network struct {
	provider *rpc.Provider
	ctx      context.Context
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

func (n *Network) GetDeployerEvents(blockNumber uint64) (*rpc.EventChunk, error) {
	udcAddress := os.Getenv("UDC_ADDRESS")
	var block rpc.BlockID

	block.Number = &blockNumber
	events, err := n.GetEvents(block, block, &udcAddress)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (n *Network) GetEvents(fromBlock rpc.BlockID, toBlock rpc.BlockID, address *string) (*rpc.EventChunk, error) {
	var addressFelt *felt.Felt
	if address != nil {
		addressFelt = new(felt.Felt)
		addressFelt.SetString(*address)
	}
	filter := rpc.EventFilter{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Address:   addressFelt,
	}
	input := rpc.EventsInput{
		EventFilter: filter,
	}
	events, err := n.provider.Events(n.ctx, input)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (n *Network) GetBlocks(fromBlock uint64, toBlock uint64) ([]*models.StarknetBlocks, error) {
	var blocks []*models.StarknetBlocks
	for i := fromBlock; i <= toBlock; i++ {
		block, err := n.provider.BlockWithTxHashes(n.ctx, rpc.BlockID{Number: &i})
		if err != nil {
			return nil, err
		}
		blockData := block.(map[string]any)
		starknetBlock := &models.StarknetBlocks{
			BlockNumber: i,
			BlockHash:   blockData["block_hash"].(string),
			ParentHash:  blockData["parent_hash"].(string),
		}
		blocks = append(blocks, starknetBlock)
	}

	return blocks, nil
}
