package cosmos

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/go-amino"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	"reflect"
	"unsafe"
)

const (
	OptionAddress = "address"
	OptionGas     = "gas"
	operationFee  = "fee"
	OptionMemo    = "memo"
)
const (
	// Metadata Keys
	ChainIDKey       = "chain_id"
	SequenceKey      = "sequence"
	AccountNumberKey = "account_number"
	GasKey           = "gas"
)

type Client struct {
	tm        rpcclient.Client
	cdc       *amino.Codec
	txDecoder sdk.TxDecoder
	txEncoder sdk.TxEncoder

	rosMsgs map[string]func(ops []*types.Operation) (sdk.Msg, error)
}

func (d Client) SupportedOperations() []string {
	ops := make([]string, 0, len(d.rosMsgs)+1)
	ops = append(ops, operationFee)
	for k := range d.rosMsgs {
		ops = append(ops, k)
	}
	return ops
}

func (d Client) NodeVersion() string {
	return "0.37.12"
}

func NewDataClient(tmEndpoint string, cdc *amino.Codec) (Client, error) {
	msgs := getOperationResolvers(cdc)
	tmClient := rpcclient.NewHTTP(tmEndpoint, "/websocket")
	// test it works
	_, err := tmClient.Health()
	if err != nil {
		return Client{}, err
	}
	dc := Client{
		tm:        tmClient,
		cdc:       cdc,
		txDecoder: auth.DefaultTxDecoder(cdc),
		txEncoder: auth.DefaultTxEncoder(cdc),
		rosMsgs:   msgs,
	}
	return dc, nil
}

// Msg interface is the interface that Cosmos SDK messages should implement if they want to
// be supported by the Rosetta service.
type Msg interface {
	// ToOperations converts the message to rosetta operations
	// the name to use must be equal to the name used to register
	// the concrete type via amino.Codec
	ToOperations(withStatus bool, hasError bool) []*types.Operation
	// FromOperations converts the operations to sdk.Msg
	FromOperations(ops []*types.Operation) (sdk.Msg, error)
}

// getOperationResolvers gets all the messages types from amino codec
func getOperationResolvers(cdc *amino.Codec) map[string]func(ops []*types.Operation) (sdk.Msg, error) {
	rosMsgType := reflect.TypeOf((*Msg)(nil)).Elem()
	resp := make(map[string]func(ops []*types.Operation) (sdk.Msg, error))
	cv := reflect.ValueOf(cdc).Elem()
	field := cv.FieldByName("typeInfos")
	// need to access the unexported field
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	// cast it to its concrete type
	typesMap := field.Interface().(map[reflect.Type]*amino.TypeInfo)
	// iterate and register types that implement rosetta.Msg
	for k, v := range typesMap {
		if k.Implements(rosMsgType) {
			handlerValue := reflect.New(v.Type)
			handler := handlerValue.Interface().(Msg)
			resp[v.Name] = handler.FromOperations
		}
	}
	return resp

}
