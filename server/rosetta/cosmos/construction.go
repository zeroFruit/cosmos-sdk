package cosmos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/cosmos/cosmos-sdk/server/rosetta"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	secp256k1 "github.com/tendermint/btcd/btcec"
	"github.com/tendermint/tendermint/crypto"
	tmsecp256k1 "github.com/tendermint/tendermint/crypto/secp256k1"
	"strings"
)

func (d Client) AccountIdentifierFromPubKeyBytes(curveType string, pkBytes []byte) (*types.AccountIdentifier, error) {
	switch curveType {
	case "secp256k1":

		pubKey, err := secp256k1.ParsePubKey(pkBytes, secp256k1.S256())
		if err != nil {
			return nil, rosetta.WrapError(rosetta.ErrInvalidPubkey, err.Error())
		}

		var pubkeyBytes tmsecp256k1.PubKeySecp256k1
		copy(pubkeyBytes[:], pubKey.SerializeCompressed())

		account := &types.AccountIdentifier{
			Address: sdk.AccAddress(pubkeyBytes.Address().Bytes()).String(),
		}

		return account, nil

	default:
		return nil, rosetta.WrapError(rosetta.ErrUnsupportedCurve, curveType)
	}
}

func (d Client) TransactionIdentifierFromHexBytes(hexBytes []byte) (txIdentifier *types.TransactionIdentifier, err error) {
	var stdTx auth.StdTx
	err = d.cdc.UnmarshalJSON(hexBytes, &stdTx)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidTransaction, err.Error())
	}

	txBytes, err := d.cdc.MarshalBinaryLengthPrefixed(stdTx)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidTransaction, err.Error())
	}

	hash := sha256.Sum256(txBytes)
	bzHash := hash[:]
	hashString := hex.EncodeToString(bzHash)

	return &types.TransactionIdentifier{Hash: strings.ToUpper(hashString)}, nil
}

func (d Client) TxOperationsAndSignersAccountIdentifiers(signed bool, txBytes []byte) (ops []*types.Operation, signers []*types.AccountIdentifier, err error) {
	var stdTx auth.StdTx
	err = d.cdc.UnmarshalJSON(txBytes, &stdTx)
	if err != nil {
		return nil, nil, rosetta.WrapError(rosetta.ErrInvalidTransaction, err.Error())
	}
	ops = sdkTxToOperations(stdTx, false, false)

	signers = make([]*types.AccountIdentifier, len(stdTx.Signatures))
	if signed {
		for i, sig := range stdTx.Signatures {
			addr, err := sdk.AccAddressFromHex(sig.PubKey.Address().String())
			if err != nil {
				return nil, nil, rosetta.WrapError(rosetta.ErrInvalidAddress, err.Error())
			}
			signers[i] = &types.AccountIdentifier{
				Address: addr.String(),
			}
		}
	}

	return
}

func (d Client) PostTxBytes(_ context.Context, txBytes []byte) (txResp *types.TransactionIdentifier, meta map[string]interface{}, err error) {
	resp, err := d.tm.BroadcastTxSync(txBytes)
	if err != nil {
		return nil, nil, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}
	txResp = &types.TransactionIdentifier{Hash: fmt.Sprintf("%X", resp.Hash)}
	meta = map[string]interface{}{
		"log": resp.Log,
	}

	return
}

func (d Client) ConstructionMetadataFromOptions(ctx context.Context, options map[string]interface{}) (meta map[string]interface{}, err error) {
	if len(options) == 0 {
		return nil, rosetta.WrapError(rosetta.ErrBadArgument, "no option provided")
	}

	addr, ok := options[OptionAddress]
	if !ok {
		return nil, rosetta.WrapError(rosetta.ErrBadArgument, "bad address")
	}
	addrString := addr.(string)
	accRes, err := d.getAccount(ctx, nil, addrString)
	if err != nil {
		return nil, err
	}

	gas, ok := options[GasKey]
	if !ok {
		return nil, rosetta.WrapError(rosetta.ErrBadArgument, "bad gas")
	}

	memo, ok := options[OptionMemo]
	if !ok {
		return nil, rosetta.WrapError(rosetta.ErrBadArgument, "bad memo")
	}

	statusRes, err := d.tm.Status()
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrUnknown, err.Error())
	}

	return map[string]interface{}{
		AccountNumberKey: accRes.GetAccountNumber(),
		SequenceKey:      accRes.GetSequence(),
		ChainIDKey:       statusRes.NodeInfo.Network,
		GasKey:           gas,
		OptionMemo:       memo,
	}, nil
}

func (d Client) SignedTx(ctx context.Context, txBytes []byte, sigs []*types.Signature) (signedTxBytes []byte, err error) {
	var stdTx auth.StdTx
	err = d.cdc.UnmarshalJSON(txBytes, &stdTx)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidTransaction, err.Error())
	}
	sdkSig := make([]auth.StdSignature, len(sigs))
	for i, signature := range sigs {
		if signature.PublicKey.CurveType != "secp256k1" {
			return nil, rosetta.WrapError(rosetta.ErrInvalidPubkey, "invalid curve "+(string)(signature.PublicKey.CurveType))
		}

		pubKey, err := secp256k1.ParsePubKey(signature.PublicKey.Bytes, secp256k1.S256())
		if err != nil {
			return nil, rosetta.WrapError(rosetta.ErrInvalidPubkey, err.Error())
		}

		var compressedPublicKey tmsecp256k1.PubKeySecp256k1
		copy(compressedPublicKey[:], pubKey.SerializeCompressed())

		sign := auth.StdSignature{
			PubKey:    compressedPublicKey,
			Signature: signature.Bytes,
		}
		sdkSig[i] = sign
	}

	stdTx.Signatures = sdkSig
	signedTxBytes, err = d.cdc.MarshalJSON(stdTx)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrCodec, "unable to marshal signed tx: "+err.Error())
	}
	return
}

func (d Client) PreprocessOperationsToOptions(ctx context.Context, req *types.ConstructionPreprocessRequest) (options map[string]interface{}, err error) {
	operations := req.Operations
	if len(operations) > 3 {
		return nil, rosetta.WrapError(rosetta.ErrBadArgument, "expected at maximum 3 operations")
	}

	msgs, _, err := operationsToSdkMsgs(d.rosMsgs, operations)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidAddress, err.Error())
	}

	memo, ok := req.Metadata["memo"]
	if !ok {
		memo = ""
	}

	defaultGas := float64(200000)

	gas := req.SuggestedFeeMultiplier
	if gas == nil {
		gas = &defaultGas
	}

	return map[string]interface{}{
		OptionAddress: msgs[0].GetSigners()[0],
		OptionMemo:    memo,
		OptionGas:     gas,
	}, nil
}

func (d Client) ConstructionPayload(ctx context.Context, req *types.ConstructionPayloadsRequest) (resp *types.ConstructionPayloadsResponse, err error) {
	if len(req.Operations) > 3 {
		return nil, rosetta.WrapError(rosetta.ErrInvalidOperation, "operations must be at least 3")
	}

	msgs, fee, err := operationsToSdkMsgs(d.rosMsgs, req.Operations)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidOperation, err.Error())
	}

	metadata, err := GetMetadataFromPayloadReq(req.Metadata)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrInvalidOperation, err.Error())
	}

	tx := auth.NewStdTx(msgs, auth.StdFee{
		Amount: fee,
		Gas:    metadata.Gas,
	}, nil, metadata.Memo)
	signBytes := auth.StdSignBytes(
		metadata.ChainID, metadata.AccountNumber, metadata.Sequence, tx.Fee, tx.Msgs, tx.Memo,
	)
	txBytes, err := d.cdc.MarshalJSON(tx)
	if err != nil {
		return nil, rosetta.WrapError(rosetta.ErrBadArgument, err.Error())
	}

	accIdentifiers := getAccountIdentifiersByMsgs(msgs)
	payloads := make([]*types.SigningPayload, len(accIdentifiers))
	for i, accID := range accIdentifiers {
		payloads[i] = &types.SigningPayload{
			AccountIdentifier: accID,
			Bytes:             crypto.Sha256(signBytes),
			SignatureType:     "ecdsa",
		}
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: hex.EncodeToString(txBytes),
		Payloads:            payloads,
	}, nil
}

func getAccountIdentifiersByMsgs(msgs []sdk.Msg) []*types.AccountIdentifier {
	var accIdentifiers []*types.AccountIdentifier
	for _, msg := range msgs {
		for _, signer := range msg.GetSigners() {
			accIdentifiers = append(accIdentifiers, &types.AccountIdentifier{Address: signer.String()})
		}
	}

	return accIdentifiers
}
