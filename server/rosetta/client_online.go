package rosetta

import (
	"context"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"

	"github.com/cosmos/cosmos-sdk/x/supply"
	crgerrs "github.com/tendermint/cosmos-rosetta-gateway/errors"
	crgtypes "github.com/tendermint/cosmos-rosetta-gateway/types"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/rpc/client/http"
)

func (c *Client) Bootstrap() error {
	tmRPC, err := http.New(c.tmEndpoint, "/websocket")
	if err != nil {
		return err
	}
	c.tm = tmRPC
	return nil
}

func (c *Client) Ready() error {
	_, err := c.tm.Health()
	return err
}

func (c *Client) Balances(ctx context.Context, address string, height *int64) (amounts []*types.Amount, err error) {
	balance, err := c.balance(ctx, address, height)
	if err != nil {
		return
	}

	amounts = make([]*types.Amount, len(balance))
	for i, coin := range balance {
		amounts[i] = &types.Amount{
			Value: coin.Amount.String(),
			Currency: &types.Currency{
				Symbol: coin.Denom,
			},
		}
	}
	return
}

func (c *Client) doABCI(ctx context.Context, height *int64, path string, req interface{}, resp interface{}) error {
	abciQuery := client.ABCIQueryOptions{
		Prove: true,
	}
	if height != nil {
		abciQuery.Height = *height
	}

	b, err := c.cdc.MarshalJSON(req)
	if err != nil {
		return crgerrs.WrapError(crgerrs.ErrCodec, err.Error())
	}

	result, err := c.tm.ABCIQueryWithOptions(path, b, abciQuery)
	if err != nil {
		return crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}

	if !result.Response.IsOK() {
		return crgerrs.WrapError(crgerrs.ErrUnknown, result.Response.Log)
	}
	err = c.cdc.UnmarshalJSON(result.Response.Value, resp)
	if err != nil {
		return crgerrs.WrapError(crgerrs.ErrCodec, err.Error())
	}

	return nil
}

func (c *Client) getAccount(ctx context.Context, height *int64, address string) (authexported.Account, error) {
	const path = "custom/" + auth.QuerierRoute + "/" + auth.QueryAccount
	sdkAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidAddress, err.Error())
	}
	params := auth.NewQueryAccountParams(sdkAddr)
	var acc authexported.Account
	err = c.doABCI(ctx, height, path, params, &acc)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	return acc, nil
}

func (c *Client) balance(ctx context.Context, address string, height *int64) (coins sdk.Coins, err error) {
	acc, err := c.getAccount(ctx, height, address)
	if err != nil {
		return nil, err
	}
	return acc.GetCoins(), nil
}

func (c *Client) supply(ctx context.Context, height *int64) (coins sdk.Coins, err error) {
	const path = "custom/" + supply.QuerierRoute + "/total_supply"
	supplyReq := struct {
		Page, Limit int
	}{
		Page:  1,
		Limit: 0,
	}
	err = c.doABCI(ctx, height, path, supplyReq, &coins)
	return
}

func (c *Client) BlockByHash(ctx context.Context, hash string) (block crgtypes.BlockResponse, err error) {
	return block, crgerrs.WrapError(crgerrs.ErrNotImplemented, "unable to get block by hash")
}

func (c *Client) BlockByHeight(ctx context.Context, height *int64) (block crgtypes.BlockResponse, err error) {
	tmBlock, err := c.tm.Block(height)
	if err != nil {
		return block, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	block = crgtypes.BlockResponse{
		Block: &types.BlockIdentifier{
			Index: tmBlock.Block.Height,
			Hash:  fmt.Sprintf("%X", tmBlock.Block.Hash()),
		},
		ParentBlock: &types.BlockIdentifier{
			Index: tmBlock.Block.Height - 1,
			Hash:  fmt.Sprintf("%X", tmBlock.Block.LastBlockID.Hash),
		},
		MillisecondTimestamp: timeToMilliseconds(tmBlock.Block.Time),
		TxCount:              int64(len(tmBlock.Block.Txs)),
	}
	return block, nil
}

func (c *Client) BlockTransactionsByHash(ctx context.Context, hash string) (block crgtypes.BlockTransactionsResponse, err error) {
	return block, crgerrs.WrapError(crgerrs.ErrNotImplemented, "unable to get block by hash")
}

func (c *Client) BlockTransactionsByHeight(ctx context.Context, height *int64) (block crgtypes.BlockTransactionsResponse, err error) {
	tmBlock, err := c.BlockByHeight(ctx, height)
	if err != nil {
		return block, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}

	// set up block
	block.BlockResponse = tmBlock
	// if the txs in the block are 0 then return
	if block.TxCount == 0 {
		return block, nil
	}
	// otherwise fetch transactions and add them to block
	tmTxs, err := c.tm.TxSearch(fmt.Sprintf("tx.height=%d", tmBlock.Block.Index), true, 0, 0, "")
	if err != nil {
		return block, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}

	block.Transactions = make([]*types.Transaction, tmBlock.TxCount)
	for i, tmTx := range tmTxs.Txs {
		decodedTx, err := c.txDecoder(tmTx.Tx)
		if err != nil {
			return block, err
		}
		block.Transactions[i] = &types.Transaction{
			TransactionIdentifier: &types.TransactionIdentifier{Hash: fmt.Sprintf("%X", tmTx.Hash)},
			Operations:            sdkTxToOperations(decodedTx, true, tmTx.TxResult.Code != 0),
		}

	}
	return block, nil
}

func (c *Client) GetTx(ctx context.Context, hash string) (tx *types.Transaction, err error) {
	tmTx, err := c.tm.Tx([]byte(hash), false)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	var cosmosTx sdk.TxResponse
	err = c.cdc.UnmarshalBinaryBare(tmTx.TxResult.Data, &cosmosTx)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrCodec, err.Error())
	}
	tx = &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: hash},
		Operations:            sdkTxToOperations(cosmosTx.Tx, true, cosmosTx.Code != 0),
	}
	return tx, nil
}

func (c *Client) GetUnconfirmedTx(ctx context.Context, hash string) (*types.Transaction, error) {
	return nil, crgerrs.ErrNotImplemented
}

func (c *Client) Mempool(ctx context.Context) (txs []*types.TransactionIdentifier, err error) {
	tmTxs, err := c.tm.UnconfirmedTxs(0)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	txs = make([]*types.TransactionIdentifier, len(tmTxs.Txs))
	for i, tmTx := range tmTxs.Txs {
		txs[i] = &types.TransactionIdentifier{Hash: fmt.Sprintf("%X", tmTx.Hash())}
	}
	return
}

func (c *Client) Peers(ctx context.Context) (peers []*types.Peer, err error) {
	netInfo, err := c.tm.NetInfo()
	if err != nil {
		return peers, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	peers = make([]*types.Peer, len(netInfo.Peers))
	for i, tmPeer := range netInfo.Peers {
		peers[i] = &types.Peer{
			PeerID: (string)(tmPeer.NodeInfo.ID()),
		}
	}
	return
}

func (c *Client) Status(ctx context.Context) (status *types.SyncStatus, err error) {
	tmStatus, err := c.tm.Status()
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	status = &types.SyncStatus{
		CurrentIndex: tmStatus.SyncInfo.LatestBlockHeight,
		TargetIndex:  nil,
		Stage:        nil,
	}
	return
}

func (c *Client) PostTx(txBytes []byte) (res *types.TransactionIdentifier, meta map[string]interface{}, err error) {
	resp, err := c.tm.BroadcastTxSync(txBytes)
	if err != nil {
		return nil, nil, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}
	res = &types.TransactionIdentifier{Hash: fmt.Sprintf("%X", resp.Hash)}
	meta = map[string]interface{}{
		"log": resp.Log,
	}

	return
}

func (c *Client) ConstructionMetadataFromOptions(ctx context.Context, options map[string]interface{}) (meta map[string]interface{}, err error) {
	if len(options) == 0 {
		return nil, crgerrs.WrapError(crgerrs.ErrBadArgument, "no option provided")
	}

	addr, ok := options[OptionAddress]
	if !ok {
		return nil, crgerrs.WrapError(crgerrs.ErrBadArgument, "bad address")
	}
	addrString := addr.(string)
	accRes, err := c.getAccount(ctx, nil, addrString)
	if err != nil {
		return nil, err
	}

	gas, ok := options[OptionGas]
	if !ok {
		return nil, crgerrs.WrapError(crgerrs.ErrBadArgument, "bad gas")
	}

	memo, ok := options[OptionMemo]
	if !ok {
		return nil, crgerrs.WrapError(crgerrs.ErrBadArgument, "bad memo")
	}

	statusRes, err := c.tm.Status()
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrUnknown, err.Error())
	}

	return map[string]interface{}{
		OptionAccountNumber: accRes.GetAccountNumber(),
		OptionSequence:      accRes.GetSequence(),
		OptionChainID:       statusRes.NodeInfo.Network,
		OptionGas:           gas,
		OptionMemo:          memo,
	}, nil
}
