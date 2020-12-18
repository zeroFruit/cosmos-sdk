package rosetta

import (
	"context"
	"github.com/coinbase/rosetta-sdk-go/types"
	crgtypes "github.com/tendermint/cosmos-rosetta-gateway/types"
)

type Client struct {
}

func (c *Client) Balances(ctx context.Context, addr string, height *int64) ([]*types.Amount, error) {
	panic("implement me")
}

func (c *Client) BlockByHash(ctx context.Context, hash string) (crgtypes.BlockResponse, error) {
	panic("implement me")
}

func (c *Client) BlockByHeight(ctx context.Context, height *int64) (crgtypes.BlockResponse, error) {
	panic("implement me")
}

func (c *Client) BlockTransactionsByHash(ctx context.Context, hash string) (crgtypes.BlockTransactionsResponse, error) {
	panic("implement me")
}

func (c *Client) BlockTransactionsByHeight(ctx context.Context, height *int64) (crgtypes.BlockTransactionsResponse, error) {
	panic("implement me")
}

func (c *Client) GetTx(ctx context.Context, hash string) (*types.Transaction, error) {
	panic("implement me")
}

func (c *Client) GetUnconfirmedTx(ctx context.Context, hash string) (*types.Transaction, error) {
	panic("implement me")
}

func (c *Client) Mempool(ctx context.Context) ([]*types.TransactionIdentifier, error) {
	panic("implement me")
}

func (c *Client) Peers(ctx context.Context) ([]*types.Peer, error) {
	panic("implement me")
}

func (c *Client) Status(ctx context.Context) (*types.SyncStatus, error) {
	panic("implement me")
}

func (c *Client) PostTx(txBytes []byte) (res *types.TransactionIdentifier, meta map[string]interface{}, err error) {
	panic("implement me")
}

func (c *Client) SignedTx(ctx context.Context, txBytes []byte, sigs []*types.Signature) (signedTxBytes []byte, err error) {
	panic("implement me")
}

func (c *Client) TxOperationsAndSignersAccountIdentifiers(signed bool, hexBytes []byte) (ops []*types.Operation, signers []*types.AccountIdentifier, err error) {
	panic("implement me")
}

func (c *Client) ConstructionMetadataFromOptions(ctx context.Context, options map[string]interface{}) (meta map[string]interface{}, err error) {
	panic("implement me")
}

func (c *Client) ConstructionPayload(ctx context.Context, req *types.ConstructionPayloadsRequest) (resp *types.ConstructionPayloadsResponse, err error) {
	panic("implement me")
}

func (c *Client) PreprocessOperationsToOptions(ctx context.Context, req *types.ConstructionPreprocessRequest) (options map[string]interface{}, err error) {
	panic("implement me")
}

func (c *Client) SupportedOperations() []string {
	panic("implement me")
}

func (c *Client) OperationStatuses() []*types.OperationStatus {
	panic("implement me")
}

func (c *Client) OperationTypes() []string {
	panic("implement me")
}

func (c *Client) Version() string {
	panic("implement me")
}

func NewClient(tmEndpoint string) (*Client, error) {
	panic("to implement")
}
