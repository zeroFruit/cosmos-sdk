package client

import (
	"context"
	"fmt"
	"sync"

	cosmosreflectionv2alpha1 "github.com/cosmos/cosmos-sdk/server/grpc/reflection/v2alpha1"

	"github.com/cosmos/cosmos-sdk/rosetta/dynamic/codec"
	"github.com/cosmos/cosmos-sdk/rosetta/dynamic/codec/protohelpers"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func newOnlineReflectionInfoProvider(ctx context.Context, endpoint string) (*onlineReflectionInfoProvider, error) {
	cc, err := grpc.DialContext(ctx, endpoint, grpc.WithInsecure(), grpc.WithResolvers())
	if err != nil {
		return nil, err
	}
	return &onlineReflectionInfoProvider{
		files:                 make(map[string][]byte),
		symbolToFilepath:      make(map[protoreflect.FullName]string),
		interfaceImplementers: make(map[string][]codec.InterfaceImplementer),
		reflection:            grpc_reflection_v1alpha.NewServerReflectionClient(cc),
		cosmosReflection:      cosmosreflectionv2alpha1.NewReflectionServiceClient(cc),
		initCosmosOnce:        new(sync.Once),
	}, nil
}

type onlineReflectionInfoProvider struct {
	files                 map[string][]byte
	symbolToFilepath      map[protoreflect.FullName]string
	services              []protoreflect.FullName
	messages              []protoreflect.FullName
	interfaces            []codec.InterfaceDescriptor
	interfaceImplementers map[string][]codec.InterfaceImplementer

	appDesc *cosmosreflectionv2alpha1.AppDescriptor

	reflection       grpc_reflection_v1alpha.ServerReflectionClient
	cosmosReflection cosmosreflectionv2alpha1.ReflectionServiceClient
	initCosmosOnce   *sync.Once
}

func (l *onlineReflectionInfoProvider) ListMessages(ctx context.Context) ([]protoreflect.FullName, error) {
	err := l.initCosmosSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to init symbols: %w", err)
	}
	return l.messages, nil
}

func (l *onlineReflectionInfoProvider) ListServices(ctx context.Context) ([]protoreflect.FullName, error) {
	err := l.initCosmosSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to init symbols: %w", err)
	}
	return l.services, nil
}

func (l *onlineReflectionInfoProvider) ListInterfaces(ctx context.Context) ([]codec.InterfaceDescriptor, error) {
	err := l.initCosmosSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to init symbols: %w", err)
	}
	return l.interfaces, nil
}

func (l *onlineReflectionInfoProvider) ListInterfaceImplementations(ctx context.Context, interfaceName string) ([]codec.InterfaceImplementer, error) {
	err := l.initCosmosSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to init symbols: %w", err)
	}
	impls, exist := l.interfaceImplementers[interfaceName]
	if !exist {
		return nil, fmt.Errorf("no interface implementers for %s", interfaceName)
	}
	return impls, nil
}

func (l *onlineReflectionInfoProvider) FilepathFromFullName(ctx context.Context, fullname protoreflect.FullName) (path string, err error) {
	// check if it exists in cache
	path, exists := l.symbolToFilepath[fullname]
	if exists {
		return path, nil
	}

	err = l.processRequest(ctx, &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: (string)(fullname),
		}})
	if err != nil {
		return "", fmt.Errorf("unable to fetch symbol %s from server: %w", fullname, err)
	}
	path, exists = l.symbolToFilepath[fullname]
	if !exists {
		return "", fmt.Errorf("server did not return file descriptor for symbol %s", fullname)
	}
	return path, nil
}

func (l *onlineReflectionInfoProvider) FileDescriptorBytes(ctx context.Context, filepath string) (rawDesc []byte, err error) {
	// check if we got it cached
	rawDesc, exists := l.files[filepath]
	if exists {
		return rawDesc, nil
	}

	err = l.processRequest(ctx, &grpc_reflection_v1alpha.ServerReflectionRequest{
		Host:           "",
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename{FileByFilename: filepath},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to fetch %s from server: %w", filepath, err)
	}

	rawDesc, exists = l.files[filepath]
	if !exists {
		return nil, fmt.Errorf("server did not return descriptor for %s", filepath)
	}

	return rawDesc, nil
}

func (l *onlineReflectionInfoProvider) processRequest(ctx context.Context, req *grpc_reflection_v1alpha.ServerReflectionRequest) error {
	switch req.MessageRequest.(type) {
	case *grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename:
	case *grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol:
	case *grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingExtension:
	default:
		panic(fmt.Sprintf("invalid request: %T", req.MessageRequest))
	}

	// send req
	stub, err := l.reflection.ServerReflectionInfo(ctx)
	if err != nil {
		return fmt.Errorf("unable to create stub: %w", err)
	}

	err = stub.Send(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}
	srvResp, err := stub.Recv()
	if err != nil {
		return fmt.Errorf("unable to receive response: %w", err)
	}
	var fdResp *grpc_reflection_v1alpha.FileDescriptorResponse
	switch t := srvResp.MessageResponse.(type) {
	case *grpc_reflection_v1alpha.ServerReflectionResponse_ErrorResponse:
		return fmt.Errorf("unable to get file: %d %s", t.ErrorResponse.ErrorCode, t.ErrorResponse.ErrorMessage)
	case *grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse:
		fdResp = t.FileDescriptorResponse
	default:
		return fmt.Errorf("unexpected response %T", t)
	}
	// now we iterate through file descriptors bytes and register its path
	// and map symbols of the given file descriptor to its file path
	for _, fdBytes := range fdResp.FileDescriptorProto {
		// get symbols and the file descriptor
		symbols, fd, err := protohelpers.AllSymbolsFromRawDescBytes(fdBytes)
		if err != nil {
			return fmt.Errorf("unable to build file descriptor: %w", err)
		}
		l.files[fd.Path()] = fdBytes
		for _, symbol := range symbols {
			l.symbolToFilepath[symbol] = fd.Path()
		}
	}

	return nil
}

func (l *onlineReflectionInfoProvider) initCosmosSymbols(ctx context.Context) error {
	var err error
	l.initCosmosOnce.Do(func() {
		var cdc *cosmosreflectionv2alpha1.GetCodecDescriptorResponse
		cdc, err = l.cosmosReflection.GetCodecDescriptor(ctx, &cosmosreflectionv2alpha1.GetCodecDescriptorRequest{})
		if err != nil {
			return
		}

		var authn *cosmosreflectionv2alpha1.GetAuthnDescriptorResponse
		authn, err = l.cosmosReflection.GetAuthnDescriptor(ctx, &cosmosreflectionv2alpha1.GetAuthnDescriptorRequest{})
		if err != nil {
			return
		}

		var services *cosmosreflectionv2alpha1.GetQueryServicesDescriptorResponse
		services, err = l.cosmosReflection.GetQueryServicesDescriptor(ctx, &cosmosreflectionv2alpha1.GetQueryServicesDescriptorRequest{})
		if err != nil {
			return
		}

		var txd *cosmosreflectionv2alpha1.GetTxDescriptorResponse
		txd, err = l.cosmosReflection.GetTxDescriptor(ctx, &cosmosreflectionv2alpha1.GetTxDescriptorRequest{})
		if err != nil {
			return
		}

		var conf *cosmosreflectionv2alpha1.GetConfigurationDescriptorResponse
		conf, err = l.cosmosReflection.GetConfigurationDescriptor(ctx, &cosmosreflectionv2alpha1.GetConfigurationDescriptorRequest{})
		if err != nil {
			return
		}

		var chain *cosmosreflectionv2alpha1.GetChainDescriptorResponse
		chain, err = l.cosmosReflection.GetChainDescriptor(ctx, &cosmosreflectionv2alpha1.GetChainDescriptorRequest{})
		if err != nil {
			return
		}
		// handle services
		l.services = make([]protoreflect.FullName, len(services.Queries.QueryServices))
		for i, svc := range services.Queries.QueryServices {
			l.services[i] = (protoreflect.FullName)(svc.Fullname)
		}
		// handle interfaces
		l.interfaces = make([]codec.InterfaceDescriptor, 0, len(cdc.Codec.Interfaces))
		for _, ifd := range cdc.Codec.Interfaces {
			l.interfaces = append(l.interfaces, codec.InterfaceDescriptor{Name: ifd.Fullname})
			l.interfaceImplementers[ifd.Fullname] = nil
			for _, impl := range ifd.InterfaceImplementers {
				l.interfaceImplementers[ifd.Fullname] = append(l.interfaceImplementers[ifd.Fullname], codec.InterfaceImplementer{
					FullName: (protoreflect.FullName)(impl.Fullname),
					TypeURL:  impl.TypeUrl,
				})
			}
		}
		// handle tx messages
		l.messages = make([]protoreflect.FullName, len(txd.Tx.Msgs))
		for i, msg := range txd.Tx.Msgs {
			switch m := msg.Msg.(type) {
			case *cosmosreflectionv2alpha1.MsgDescriptor_ServiceMsg:
				l.messages[i] = (protoreflect.FullName)(m.ServiceMsg.RequestFullname)
			default:
				err = fmt.Errorf("unrecognized tx.Msg type: %T", m)
				return
			}
		}
		// build the app descriptor
		l.appDesc = &cosmosreflectionv2alpha1.AppDescriptor{
			Authn:         authn.Authn,
			Chain:         chain.Chain,
			Codec:         cdc.Codec,
			Configuration: conf.Config,
			QueryServices: services.Queries,
			Tx:            txd.Tx,
		}
	})

	if err != nil {
		return fmt.Errorf("unable to init symbols: %w", err)
	}
	return nil
}
