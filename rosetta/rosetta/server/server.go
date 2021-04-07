package server

import "github.com/coinbase/rosetta-sdk-go/server"

// Server defines the rosetta implementer server
type Server interface {
	server.BlockAPIServicer
	server.AccountAPIServicer
	server.CallAPIServicer
	server.NetworkAPIServicer
	server.ConstructionAPIServicer
}

type option struct {
}

type Options interface {
	apply(o *option)
}

func NewServer() (Server, error) {
	panic("implement plz")
}
