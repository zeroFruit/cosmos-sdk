package rosetta

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/go-amino"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	"reflect"
	"unsafe"
)

// statuses
const (
	StatusSuccess = "Success"
	StageSynced   = "synced"
	StageSyncing  = "syncing"
)

// misc
const (
	Log = "log"
)

// operations
const (
	OperationFee = "fee"
)

// options
const (
	OptionAccountNumber = "account_number"
	OptionAddress       = "address"
	OptionChainID       = "chain_id"
	OptionSequence      = "sequence"
	OptionMemo          = "memo"
	OptionGas           = "gas"
)

type Client struct {
	tmEndpoint string

	tm        rpcclient.Client
	cdc       *amino.Codec
	txDecoder sdk.TxDecoder
	txEncoder sdk.TxEncoder

	rosMsgs map[string]func(ops []*types.Operation) (sdk.Msg, error)
}

func (c *Client) SupportedOperations() []string {
	ops := make([]string, 0, len(c.rosMsgs)+1)
	ops = append(ops, OperationFee)
	for k := range c.rosMsgs {
		ops = append(ops, k)
	}
	return ops
}

func (c *Client) OperationStatuses() []*types.OperationStatus {
	return []*types.OperationStatus{
		{
			Status:     StatusSuccess,
			Successful: true,
		},
	}
}

func (c *Client) Version() string {
	return "cosmos:0.39.x"
}

func NewClient(tmEndpoint string, cdc *amino.Codec) (*Client, error) {
	msgs := getOperationResolvers(cdc)
	dc := &Client{
		tmEndpoint: tmEndpoint,
		cdc:        cdc,
		txDecoder:  auth.DefaultTxDecoder(cdc),
		txEncoder:  auth.DefaultTxEncoder(cdc),
		rosMsgs:    msgs,
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
