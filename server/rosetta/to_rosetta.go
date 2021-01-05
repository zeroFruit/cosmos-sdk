package rosetta

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// sdkTxToOperations converts an sdk.Tx to rosetta operations
func sdkTxToOperations(tx sdk.Tx, withStatus, hasError bool) []*types.Operation {
	var operations []*types.Operation

	feeTx := tx.(auth.StdTx)
	feeCoins := feeTx.Fee.Amount

	msgOps := sdkMsgsToRosettaOperations(tx.GetMsgs(), withStatus, hasError)
	operations = append(operations, msgOps...)

	var feeOps = rosettaFeeOperationsFromCoins(feeCoins, feeTx.GetSigners()[0].String(), withStatus, len(msgOps))
	operations = append(operations, feeOps...)

	return operations
}

// rosettaFeeOperationsFromCoins returns the list of rosetta fee operations given sdk coins
func rosettaFeeOperationsFromCoins(coins sdk.Coins, account string, withStatus bool, previousOps int) []*types.Operation {
	feeOps := make([]*types.Operation, 0)
	var status string
	if withStatus {
		status = StatusSuccess
	}

	for i, coin := range coins {
		op := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(i + previousOps),
			},
			Type:   OperationFee,
			Status: status,
			Account: &types.AccountIdentifier{
				Address: account,
			},
			Amount: &types.Amount{
				Value: "-" + coin.Amount.String(),
				Currency: &types.Currency{
					Symbol: coin.Denom,
				},
			},
		}

		feeOps = append(feeOps, op)
	}

	return feeOps
}

// sdkMsgsToRosettaOperations converts sdk messages to rosetta operations
func sdkMsgsToRosettaOperations(msgs []sdk.Msg, withStatus bool, hasError bool) []*types.Operation {
	var operations []*types.Operation
	for _, msg := range msgs {
		if rosettaMsg, ok := msg.(Msg); ok {
			operations = append(operations, rosettaMsg.ToOperations(withStatus, hasError)...)
		}
	}

	return operations
}
