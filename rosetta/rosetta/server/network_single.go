package server

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/cosmos/cosmos-sdk/rosetta/dynamic/client"
)

var _ Server = (*Network)(nil)

type Network struct {
	client *client.Client

	allow   *types.Allow
	version *types.Version
}

func (n Network) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	panic("implement me")
}

func (n Network) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	panic("implement me")
}

func (n Network) AccountBalance(ctx context.Context, request *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	panic("implement me")
}

func (n Network) AccountCoins(ctx context.Context, request *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	panic("implement me")
}

func (n Network) Call(ctx context.Context, request *types.CallRequest) (*types.CallResponse, *types.Error) {
	panic("implement me")
}

func (n Network) NetworkList(ctx context.Context, request *types.MetadataRequest) (*types.NetworkListResponse, *types.Error) {
	panic("implement me")
}

func (n Network) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	return &types.NetworkOptionsResponse{
		Version: n.version,
		Allow:   n.allow,
	}, nil
}

func (n Network) NetworkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionCombine(ctx context.Context, request *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionDerive(ctx context.Context, request *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionHash(ctx context.Context, request *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionMetadata(ctx context.Context, request *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionParse(ctx context.Context, request *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionPayloads(ctx context.Context, request *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionPreprocess(ctx context.Context, request *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	panic("implement me")
}

func (n Network) ConstructionSubmit(ctx context.Context, request *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	panic("implement me")
}
