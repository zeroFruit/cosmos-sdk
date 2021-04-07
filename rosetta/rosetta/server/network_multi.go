package server

import (
	"context"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/cosmos/cosmos-sdk/rosetta/rosetta/errors"
)

var _ Server = (*MultiNetwork)(nil)

// MultiNetwork defines a Server which can support
// multiple networks at the same time
type MultiNetwork struct {
	// networks defines a set of multiple Server instances
	// and each of them can handle a specific network
	networks map[string]Server
	// networkIdentifiers contains a list of all the networks
	networkIdentifiers []*types.NetworkIdentifier
}

func (m MultiNetwork) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.Block(ctx, request)
}

func (m MultiNetwork) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.BlockTransaction(ctx, request)
}

func (m MultiNetwork) AccountBalance(ctx context.Context, request *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.AccountBalance(ctx, request)
}

func (m MultiNetwork) AccountCoins(ctx context.Context, request *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.AccountCoins(ctx, request)
}

func (m MultiNetwork) Call(ctx context.Context, request *types.CallRequest) (*types.CallResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.Call(ctx, request)
}

func (m MultiNetwork) NetworkList(ctx context.Context, request *types.MetadataRequest) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{NetworkIdentifiers: m.networkIdentifiers}, nil
}

func (m MultiNetwork) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.NetworkOptions(ctx, request)
}

func (m MultiNetwork) NetworkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.NetworkStatus(ctx, request)
}

func (m MultiNetwork) ConstructionCombine(ctx context.Context, request *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionCombine(ctx, request)
}

func (m MultiNetwork) ConstructionDerive(ctx context.Context, request *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionDerive(ctx, request)
}

func (m MultiNetwork) ConstructionHash(ctx context.Context, request *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionHash(ctx, request)
}

func (m MultiNetwork) ConstructionMetadata(ctx context.Context, request *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionMetadata(ctx, request)
}

func (m MultiNetwork) ConstructionParse(ctx context.Context, request *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionParse(ctx, request)
}

func (m MultiNetwork) ConstructionPayloads(ctx context.Context, request *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionPayloads(ctx, request)
}

func (m MultiNetwork) ConstructionPreprocess(ctx context.Context, request *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionPreprocess(ctx, request)
}

func (m MultiNetwork) ConstructionSubmit(ctx context.Context, request *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	network, err := m.networkFor(request.NetworkIdentifier)
	if err != nil {
		return nil, err
	}
	return network.ConstructionSubmit(ctx, request)
}

func (m MultiNetwork) networkFor(networkIdentifier *types.NetworkIdentifier) (Server, *types.Error) {
	strNetID := m.hashNetwork(networkIdentifier)
	network, exists := m.networks[strNetID]
	if !exists {
		return nil, errors.NotFound
	}
	return network, nil
}

func (m MultiNetwork) hashNetwork(networkIdentifier *types.NetworkIdentifier) string {
	return fmt.Sprintf("%s/%s", networkIdentifier.Network, networkIdentifier.Blockchain)
}
