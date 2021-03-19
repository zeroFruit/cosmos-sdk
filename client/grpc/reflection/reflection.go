package reflection

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/codec/types"
)

type reflectionServiceServer struct {
	interfaceRegistry types.InterfaceRegistry

	initAppDescOnce *sync.Once     // inits appDesc once
	appDesc         *AppDescriptor // static app descriptor, changes in the information don't happen during runtime
}

// NewReflectionServiceServer creates a new reflectionServiceServer.
func NewReflectionServiceServer(interfaceRegistry types.InterfaceRegistry) ReflectionServiceServer {
	return &reflectionServiceServer{
		interfaceRegistry: interfaceRegistry,
		initAppDescOnce:   new(sync.Once),
		appDesc:           new(AppDescriptor),
	}
}

var _ ReflectionServiceServer = (*reflectionServiceServer)(nil)

// GetAppDescriptor implements GetAppDescriptor method of the ReflectionServiceServer interface
func (r reflectionServiceServer) GetAppDescriptor(_ context.Context, _ *GetAppDescriptorRequest) (*GetAppDescriptorResponse, error) {
	r.initAppDescOnce.Do(r.initAppDescriptor)
	return &GetAppDescriptorResponse{AppDescriptor: r.appDesc}, nil
}

// ListAllInterfaces implements the ListAllInterfaces method of the
// ReflectionServiceServer interface.
func (r reflectionServiceServer) ListAllInterfaces(_ context.Context, _ *ListAllInterfacesRequest) (*ListAllInterfacesResponse, error) {
	ifaces := r.interfaceRegistry.ListAllInterfaces()

	return &ListAllInterfacesResponse{InterfaceNames: ifaces}, nil
}

// ListImplementations implements the ListImplementations method of the
// ReflectionServiceServer interface.
func (r reflectionServiceServer) ListImplementations(_ context.Context, req *ListImplementationsRequest) (*ListImplementationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.InterfaceName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid interface name")
	}

	impls := r.interfaceRegistry.ListImplementations(req.InterfaceName)

	return &ListImplementationsResponse{ImplementationMessageNames: impls}, nil
}

// initAppDescriptor creates the app descriptor
func (r reflectionServiceServer) initAppDescriptor() {
	sdkConf := sdk.GetConfig()
	// set configuration
	confDesc := Config{
		Bech32AccAddressPrefix: sdkConf.GetBech32AccountAddrPrefix(),
		Bech32AccPubPrefix:     sdkConf.GetBech32AccountPubPrefix(),
		Bech32ValAddrPrefix:    sdkConf.GetBech32ValidatorAddrPrefix(),
		Bech32ValPubPrefix:     sdkConf.GetBech32ValidatorPubPrefix(),
		Bech32ConsAddrPrefix:   sdkConf.GetBech32ConsensusAddrPrefix(),
		Bech32ConsPubPrefix:    sdkConf.GetBech32ConsensusPubPrefix(),
		Purpose:                sdkConf.GetPurpose(),
		CoinType:               sdkConf.GetCoinType(),
	}

	r.appDesc.Config = &confDesc
	// set messages info
	msgImpls := r.interfaceRegistry.ListImplementations(sdk.MsgInterfaceProtoName)
	svcMsgImpls := r.interfaceRegistry.ListImplementations(sdk.ServiceMsgInterfaceProtoName)
	messages := make([]*Message, 0, len(msgImpls)+len(svcMsgImpls))

	for _, msg := range msgImpls {
		pb, err := r.interfaceRegistry.Resolve(msg)
		if err != nil {
			panic(fmt.Sprintf("unexpected interface registry error when trying to resolve type %s: %s", msg, err))
		}
		pbName := proto.MessageName(pb)
		if pbName == "" {
			panic(fmt.Sprintf("proto.MessageName for type %s returned an empty string", msg))
		}
		messages = append(messages, &Message{Message: &Message_Msg{Msg: &Msg{FullName: proto.MessageName(pb)}}})
	}

	for _, methodName := range svcMsgImpls {
		pb, err := r.interfaceRegistry.Resolve(methodName)
		if err != nil {
			panic(fmt.Sprintf("unexpected interface registry error when trying to resolve type %s: %s", methodName, err))
		}
		pbName := proto.MessageName(pb)
		if pbName == "" {
			panic(fmt.Sprintf("proto.MessageName for type %s returned an empty string", methodName))
		}
		messages = append(messages, &Message{
			Message: &Message_ServiceMsg{
				ServiceMsg: &ServiceMsg{
					FullName:   pbName,
					MethodName: methodName,
				},
			},
		})
	}

	r.appDesc.Messages = messages

	// set codec info
	interfaces := r.interfaceRegistry.ListAllInterfaces()
	codec := Codec{
		InterfacesDescriptors: make([]*InterfaceDescriptor, len(interfaces)),
	}

	for i, iface := range interfaces {
		implementers := r.interfaceRegistry.ListImplementations(iface)
		interfaceImplementers := make([]*InterfaceImplementer, len(implementers))
		for j, implementer := range implementers {
			pb, err := r.interfaceRegistry.Resolve(implementer)
			if err != nil {
				panic(fmt.Sprintf("unexpected interface registry error when trying to resolve type %s: %s", implementer, err))
			}
			pbName := proto.MessageName(pb)
			if pbName == "" {
				panic(fmt.Sprintf("proto.MessageName for type %s returned an empty string", implementer))
			}
			interfaceImplementers[j] = &InterfaceImplementer{FullName: pbName}
		}

		codec.InterfacesDescriptors[i] = &InterfaceDescriptor{
			Name:                  iface,
			InterfaceImplementers: interfaceImplementers,
		}
	}

	r.appDesc.Codec = &codec

	// set query services

}
