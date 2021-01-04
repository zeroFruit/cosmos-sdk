package rosetta

import (
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
	"strings"
)

func operationsToSdkMsgs(handlers map[string]func(ops []*types.Operation) (sdk.Msg, error), ops []*types.Operation) ([]sdk.Msg, sdk.Coins, error) {
	var feeAmnt []*types.Amount
	var newOps []*types.Operation

	for _, op := range ops {
		switch op.Type {
		case OperationFee:
			amount := op.Amount
			feeAmnt = append(feeAmnt, amount)
		default:
			newOps = append(newOps, op)
		}
	}

	msgs, err := convertOpsToMsgs(handlers, newOps)
	if err != nil {
		return nil, nil, err
	}

	return msgs, amountsToCoins(feeAmnt), nil
}

func convertOpsToMsgs(opHandlers map[string]func(ops []*types.Operation) (sdk.Msg, error), ops []*types.Operation) (msgs []sdk.Msg, err error) {
	opsForHandler := make(map[string][]*types.Operation)
	for _, op := range ops {
		opsForHandler[op.Type] = append(opsForHandler[op.Type], op)
	}

	for k, v := range opsForHandler {
		handler, ok := opHandlers[k]
		if !ok {
			return nil, fmt.Errorf("handler not found for operation: %s", k)
		}
		msg, err := handler(v)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

type payloadReqMeta struct {
	ChainID       string
	Sequence      uint64
	AccountNumber uint64
	Gas           uint64
	Memo          string
}

// GetMetadataFromPayloadReq obtains the metadata from the request to /construction/payloads endpoint.
func GetMetadataFromPayloadReq(metadata map[string]interface{}) (*payloadReqMeta, error) {
	chainID, ok := metadata[OptionChainID].(string)
	if !ok {
		return nil, fmt.Errorf("chain_id metadata was not provided")
	}

	sequence, ok := metadata[OptionSequence]
	if !ok {
		return nil, fmt.Errorf("sequence metadata was not provided")
	}

	seqNum, ok := sequence.(float64)
	if !ok {
		return nil, fmt.Errorf("invalid sequence value")
	}

	accountNum, ok := metadata[OptionAccountNumber]
	if !ok {
		return nil, fmt.Errorf("account_number metadata was not provided")
	}
	accNum, ok := accountNum.(float64)
	if !ok {
		return nil, fmt.Errorf("invalid account_number value")
	}

	gasNum, ok := metadata[OptionGas]
	if !ok {
		return nil, fmt.Errorf("gas metadata was not provided")
	}
	gasF64, ok := gasNum.(float64)
	if !ok {
		return nil, fmt.Errorf("invalid gas value")
	}

	memo, ok := metadata["memo"]
	if !ok {
		memo = ""
	}
	memoStr, ok := memo.(string)
	if !ok {
		return nil, fmt.Errorf("invalid account_number value")
	}

	return &payloadReqMeta{
		ChainID:       chainID,
		Sequence:      uint64(seqNum),
		AccountNumber: uint64(accNum),
		Gas:           uint64(gasF64),
		Memo:          memoStr,
	}, nil
}

// amountsToCoins converts rosetta amounts to sdk coins
func amountsToCoins(amounts []*types.Amount) sdk.Coins {
	var feeCoins sdk.Coins

	for _, amount := range amounts {
		absValue := strings.Trim(amount.Value, "-")
		value, err := strconv.ParseInt(absValue, 10, 64)
		if err != nil {
			return nil
		}
		coin := sdk.NewCoin(amount.Currency.Symbol, sdk.NewInt(value))
		feeCoins = append(feeCoins, coin)
	}

	return feeCoins
}
