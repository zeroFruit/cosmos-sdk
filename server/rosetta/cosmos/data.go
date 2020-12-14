package cosmos

import (
	"context"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/cosmos/cosmos-sdk/server/rosetta"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/supply"
	"github.com/tendermint/tendermint/rpc/client"
)

func (d Client) Balances(ctx context.Context, address string, height *int64) (amounts []*types.Amount, err error) {
	balance, err := d.balance(ctx, address, height)
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

func (d Client) doABCI(ctx context.Context, height *int64, path string, req interface{}, resp interface{}) error {
	abciQuery := client.ABCIQueryOptions{
		Prove: true,
	}
	if height != nil {
		abciQuery.Height = *height
	}

	b, err := d.cdc.MarshalJSON(req)
	if err != nil {
		return rosetta.WrapError(rosetta.ErrCodec, err.Error())
	}

	result, err := d.tm.ABCIQueryWithOptions(path, b, abciQuery)
	if err != nil {
		return rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}

	if !result.Response.IsOK() {
		return rosetta.WrapError(rosetta.ErrUnknown, result.Response.Log)
	}
	err = d.cdc.UnmarshalJSON(result.Response.Value, resp)
	if err != nil {
		return rosetta.WrapError(rosetta.ErrCodec, err.Error())
	}

	return nil
}

func (d Client) getAccount(ctx context.Context, height *int64, address string) (auth.Account, error) {
	const path = "custom/" + auth.QuerierRoute + "/" + auth.QueryAccount
	sdkAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidAddress, err.Error())
	}
	params := auth.NewQueryAccountParams(sdkAddr)
	var acc auth.Account
	err = d.doABCI(ctx, height, path, params, &acc)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	return acc, nil
}

func (d Client) balance(ctx context.Context, address string, height *int64) (coins sdk.Coins, err error) {
	acc, err := d.getAccount(ctx, height, address)
	if err != nil {
		return nil, err
	}
	return acc.GetCoins(), nil
}

func (d Client) supply(ctx context.Context, height *int64) (coins sdk.Coins, err error) {
	const path = "custom/" + supply.QuerierRoute + "/total_supply"
	supplyReq := struct {
		Page, Limit int
	}{
		Page:  1,
		Limit: 0,
	}
	err = d.doABCI(ctx, height, path, supplyReq, &coins)
	return
}

func (d Client) BlockByHeight(_ context.Context, height *int64) (block rosetta.BlockResponse, err error) {
	tmBlock, err := d.tm.Block(height)
	if err != nil {
		return block, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	block = rosetta.BlockResponse{
		Block: &types.BlockIdentifier{
			Index: tmBlock.BlockMeta.Header.Height,
			Hash:  fmt.Sprintf("%X", tmBlock.Block.Hash()),
		},
		ParentBlock: &types.BlockIdentifier{
			Index: tmBlock.BlockMeta.Header.Height - 1,
			Hash:  fmt.Sprintf("%X", tmBlock.BlockMeta.Header.LastBlockID.Hash),
		},
		MillisecondTimestamp: timeToMilliseconds(tmBlock.Block.Time),
		TxCount:              tmBlock.Block.NumTxs,
	}
	return block, nil
}

func (d Client) BlockByHash(_ context.Context, _ string) (block rosetta.BlockResponse, err error) {
	return block, rosetta.WrapError(rosetta.ErrNotImplemented, "unable to get block by hash")
}

func (d Client) BlockTransactionsByHeight(ctx context.Context, height *int64) (block rosetta.BlockTransactionsResponse, err error) {
	tmBlock, err := d.BlockByHeight(ctx, height)
	if err != nil {
		return block, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}

	// set up block
	block.BlockResponse = tmBlock
	// if the txs in the block are 0 then return
	if block.TxCount == 0 {
		return block, nil
	}
	// otherwise fetch transactions and add them to block
	tmTxs, err := d.tm.TxSearch(fmt.Sprintf("tx.height=%d", tmBlock.Block.Index), true, 0, 0)
	if err != nil {
		return block, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}

	block.Transactions = make([]*types.Transaction, tmBlock.TxCount)
	for i, tmTx := range tmTxs.Txs {
		decodedTx, err := d.txDecoder(tmTx.Tx)
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

func (d Client) BlockTransactionsByHash(_ context.Context, _ string) (block rosetta.BlockTransactionsResponse, err error) {
	return block, rosetta.WrapError(rosetta.ErrNotImplemented, "unable to get block transactions given a block hash")
}

func (d Client) GetTransaction(_ context.Context, hash string) (tx *types.Transaction, err error) {
	tmTx, err := d.tm.Tx([]byte(hash), false)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	var cosmosTx sdk.TxResponse
	err = d.cdc.UnmarshalBinaryBare(tmTx.TxResult.Data, &cosmosTx)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrCodec, err.Error())
	}
	tx = &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: hash},
		Operations:            sdkTxToOperations(cosmosTx.Tx, true, cosmosTx.Code != 0),
	}
	return tx, nil
}

func (d Client) GetMempoolTransactions(_ context.Context) (txs []*types.TransactionIdentifier, err error) {
	tmTxs, err := d.tm.UnconfirmedTxs(0)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	txs = make([]*types.TransactionIdentifier, len(tmTxs.Txs))
	for i, tmTx := range tmTxs.Txs {
		txs[i] = &types.TransactionIdentifier{Hash: fmt.Sprintf("%X", tmTx.Hash())}
	}
	return
}

func (d Client) GetMempoolTransaction(_ context.Context, _ string) (tx *types.Transaction, err error) {
	return nil, rosetta.ErrNotImplemented
}

func (d Client) Peers(_ context.Context) (peers []*types.Peer, err error) {
	netInfo, err := d.tm.NetInfo()
	if err != nil {
		return peers, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	peers = make([]*types.Peer, len(netInfo.Peers))
	for i, tmPeer := range netInfo.Peers {
		peers[i] = &types.Peer{
			PeerID: (string)(tmPeer.NodeInfo.ID()),
		}
	}
	return
}

func (d Client) Status(_ context.Context) (status *types.SyncStatus, err error) {
	tmStatus, err := d.tm.Status()
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	status = &types.SyncStatus{
		CurrentIndex: tmStatus.SyncInfo.LatestBlockHeight,
		TargetIndex:  nil,
		Stage:        nil,
	}
	return
}
