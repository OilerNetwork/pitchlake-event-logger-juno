package network

import (
	"context"
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

func (n *Network) GetEvents(fromBlock uint64, toBlock uint64, address string) (*rpc.EventChunk, error) {
	addressFelt := new(felt.Felt)
	addressFelt.SetString(address)
	filter := rpc.EventFilter{
		FromBlock: rpc.BlockID{Number: &fromBlock},
		ToBlock:   rpc.BlockID{Number: &toBlock},
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

func (n *Network) GetUDCEventsAt(blockNumber uint64) (*rpc.EventChunk, error) {
	events, err := n.GetEvents(blockNumber, blockNumber, os.Getenv("UDC_ADDRESS"))
	if err != nil {
		return nil, err
	}
	return events, nil
}
