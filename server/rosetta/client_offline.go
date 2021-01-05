package rosetta

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	secp256k1 "github.com/tendermint/btcd/btcec"
	crgerrs "github.com/tendermint/cosmos-rosetta-gateway/errors"
	"github.com/tendermint/tendermint/crypto"
	tmsecp256k1 "github.com/tendermint/tendermint/crypto/secp256k1"
)

func (c *Client) SignedTx(ctx context.Context, txBytes []byte, sigs []*types.Signature) (signedTxBytes []byte, err error) {
	rawTx, err := c.txDecoder(txBytes)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidTransaction, err.Error())
	}
	stdTx, ok := rawTx.(auth.StdTx)
	if !ok {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidTransaction, fmt.Sprintf("unexpected transaction of type: %T", rawTx))
	}
	sdkSig := make([]auth.StdSignature, len(sigs))
	for i, signature := range sigs {
		if signature.PublicKey.CurveType != "secp256k1" {
			return nil, crgerrs.WrapError(crgerrs.ErrInvalidPubkey, "invalid curve "+(string)(signature.PublicKey.CurveType))
		}

		pubKey, err := secp256k1.ParsePubKey(signature.PublicKey.Bytes, secp256k1.S256())
		if err != nil {
			return nil, crgerrs.WrapError(crgerrs.ErrInvalidPubkey, err.Error())
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
	signedTxBytes, err = c.txEncoder(stdTx)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrCodec, "unable to marshal signed tx: "+err.Error())
	}
	return
}

func (c *Client) TxOperationsAndSignersAccountIdentifiers(signed bool, hexBytes []byte) (ops []*types.Operation, signers []*types.AccountIdentifier, err error) {
	rawTx, err := c.txDecoder(hexBytes)
	if err != nil {
		return nil, nil, crgerrs.WrapError(crgerrs.ErrInvalidTransaction, err.Error())
	}
	stdTx, ok := rawTx.(auth.StdTx)
	if !ok {
		return nil, nil, crgerrs.WrapError(crgerrs.ErrInvalidTransaction, fmt.Sprintf("unexpected transaction of type: %T", rawTx))
	}
	ops = sdkTxToOperations(stdTx, false, false)

	signers = make([]*types.AccountIdentifier, len(stdTx.Signatures))
	if signed {
		for i, sig := range stdTx.Signatures {
			addr, err := sdk.AccAddressFromHex(sig.PubKey.Address().String())
			if err != nil {
				return nil, nil, crgerrs.WrapError(crgerrs.ErrInvalidAddress, err.Error())
			}
			signers[i] = &types.AccountIdentifier{
				Address: addr.String(),
			}
		}
	}

	return
}

func (c *Client) ConstructionPayload(ctx context.Context, req *types.ConstructionPayloadsRequest) (resp *types.ConstructionPayloadsResponse, err error) {
	if len(req.Operations) > 3 {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidOperation, "operations must be at least 3")
	}

	msgs, fee, err := operationsToSdkMsgs(c.rosMsgs, req.Operations)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidOperation, err.Error())
	}

	metadata, err := GetMetadataFromPayloadReq(req.Metadata)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidOperation, err.Error())
	}

	tx := auth.NewStdTx(msgs, auth.StdFee{
		Amount: fee,
		Gas:    metadata.Gas,
	}, nil, metadata.Memo)
	signBytes := auth.StdSignBytes(
		metadata.ChainID, metadata.AccountNumber, metadata.Sequence, tx.Fee, tx.Msgs, tx.Memo,
	)
	txBytes, err := c.txEncoder(tx)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrBadArgument, err.Error())
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

func (c *Client) PreprocessOperationsToOptions(ctx context.Context, req *types.ConstructionPreprocessRequest) (options map[string]interface{}, err error) {
	operations := req.Operations
	if len(operations) > 3 {
		return nil, crgerrs.WrapError(crgerrs.ErrBadArgument, "expected at maximum 3 operations")
	}

	msgs, _, err := operationsToSdkMsgs(c.rosMsgs, operations)
	if err != nil {
		return nil, crgerrs.WrapError(crgerrs.ErrInvalidAddress, err.Error())
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

func (c *Client) AccountIdentifierFromPublicKey(pubKey *types.PublicKey) (*types.AccountIdentifier, error) {
	switch pubKey.CurveType {
	case "secp256k1":
		pubKey, err := secp256k1.ParsePubKey(pubKey.Bytes, secp256k1.S256())
		if err != nil {
			return nil, crgerrs.WrapError(crgerrs.ErrInvalidPubkey, err.Error())
		}

		var pubkeyBytes tmsecp256k1.PubKeySecp256k1
		copy(pubkeyBytes[:], pubKey.SerializeCompressed())

		account := &types.AccountIdentifier{
			Address: sdk.AccAddress(pubkeyBytes.Address().Bytes()).String(),
		}

		return account, nil

	default:
		return nil, crgerrs.WrapError(crgerrs.ErrUnsupportedCurve, (string)(pubKey.CurveType))
	}
}
