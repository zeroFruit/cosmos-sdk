package client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/rosetta/dynamic/codec"
)

// Dial creates a *Client instance given gRPC and tendermint endpoint
func Dial(ctx context.Context, tmEndpoint string, gRPCEndpoint string) (*Client, error) {
	rip, err := newOnlineReflectionInfoProvider(ctx, gRPCEndpoint)
	if err != nil {
		return nil, err
	}
	cdcBuilder := codec.NewBuilder(rip, rip)
	cdc, err := cdcBuilder.Build(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{cdc: cdc}, nil
}

type Client struct {
	cdc *codec.Codec
	cc  grpc.ClientConnInterface
}

func (c *Client) Query(ctx context.Context, request proto.Message, response proto.Message) error {
	err := c.cc.Invoke(ctx, "", request, response)
	if err != nil {
		return err
	}
	return err
}
