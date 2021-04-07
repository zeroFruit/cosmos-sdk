package server

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"
)

var _ Server = (*requestSanitizer)(nil)

// requestSanitizer is meant to be wrapped around a Server
// and acts as a request checker middleware that asserts
// that requests are correctly formed to avoid, for example,
// nil dereferences.
type requestSanitizer struct {
	s Server
}

func (r requestSanitizer) Block(ctx context.Context, request *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	return r.s.Block(ctx, request)
}

func (r requestSanitizer) BlockTransaction(ctx context.Context, request *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	return r.s.BlockTransaction(ctx, request)
}

func (r requestSanitizer) AccountBalance(ctx context.Context, request *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	return r.s.AccountBalance(ctx, request)
}

func (r requestSanitizer) AccountCoins(ctx context.Context, request *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	return r.s.AccountCoins(ctx, request)
}

func (r requestSanitizer) Call(ctx context.Context, request *types.CallRequest) (*types.CallResponse, *types.Error) {
	return r.s.Call(ctx, request)
}

func (r requestSanitizer) NetworkList(ctx context.Context, request *types.MetadataRequest) (*types.NetworkListResponse, *types.Error) {
	return r.s.NetworkList(ctx, request)
}

func (r requestSanitizer) NetworkOptions(ctx context.Context, request *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	return r.s.NetworkOptions(ctx, request)
}

func (r requestSanitizer) NetworkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	return r.s.NetworkStatus(ctx, request)
}

func (r requestSanitizer) ConstructionCombine(ctx context.Context, request *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	return r.s.ConstructionCombine(ctx, request)
}

func (r requestSanitizer) ConstructionDerive(ctx context.Context, request *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return r.s.ConstructionDerive(ctx, request)
}

func (r requestSanitizer) ConstructionHash(ctx context.Context, request *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	return r.s.ConstructionHash(ctx, request)
}

func (r requestSanitizer) ConstructionMetadata(ctx context.Context, request *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error) {
	return r.s.ConstructionMetadata(ctx, request)
}

func (r requestSanitizer) ConstructionParse(ctx context.Context, request *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	return r.s.ConstructionParse(ctx, request)
}

func (r requestSanitizer) ConstructionPayloads(ctx context.Context, request *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	return r.s.ConstructionPayloads(ctx, request)
}

func (r requestSanitizer) ConstructionPreprocess(ctx context.Context, request *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	return r.s.ConstructionPreprocess(ctx, request)
}

func (r requestSanitizer) ConstructionSubmit(ctx context.Context, request *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	return r.s.ConstructionSubmit(ctx, request)
}
