package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/cosmos/cosmos-sdk/rosetta/dynamic/codec/protohelpers"
)

type live struct {
	files            map[string][]byte
	symbolToFilepath map[protoreflect.FullName]string
	c                grpc_reflection_v1alpha.ServerReflectionClient
}

func newLive(ctx context.Context, endpoint string) (*live, error) {
	anypb.MarshalFrom()
	cc, err := grpc.DialContext(ctx, endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &live{
		files:            make(map[string][]byte),
		symbolToFilepath: make(map[protoreflect.FullName]string),
		c:                grpc_reflection_v1alpha.NewServerReflectionClient(cc),
	}, nil
}

func (l live) FilepathFromFullName(ctx context.Context, fullname protoreflect.FullName) (path string, err error) {
	// check if it exists in cache
	path, exists := l.symbolToFilepath[fullname]
	if exists {
		return path, nil
	}
	// else fetch it live
	stub, err := l.c.ServerReflectionInfo(ctx)
	if err != nil {
		return "", err
	}

	err = stub.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: string(fullname)},
	})
	if err != nil {
		return "", err
	}
	// get resp
	resp, err := stub.Recv()
	if err != nil {
		return "", err
	}

	fileContainingSymbol, ok := resp.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return "", fmt.Errorf("unexpected response: %T", resp.MessageResponse)
	}

	files := fileContainingSymbol.FileDescriptorResponse.FileDescriptorProto
	for _, file := range files {
		filepath, symbols, err := processFile(file)
		if err != nil {
			return "", err
		}
		l.files[filepath] = file
		for _, symbol := range symbols {
			l.symbolToFilepath[symbol] = filepath
		}
	}

	// now try to see if it's there
	path, exists = l.symbolToFilepath[fullname]
	if !exists {
		return "", fmt.Errorf("despite fetching dependencies fullname %s was not found", fullname)
	}

	return path, nil
}

func (l live) FileDescriptorBytes(ctx context.Context, filepath string) (rawDesc []byte, err error) {
	// check if we got it cached
	rawDesc, exists := l.files[filepath]
	if exists {
		return rawDesc, nil
	}

	// otherwise we will need to find it from the reflection server
}

func processFile(desc []byte) (path string, symbols []protoreflect.FullName, err error) {
	// unzip desc
	desc, err = protohelpers.Unzip(desc)
	if err != nil {
		return "", nil, err
	}

	fd, err := protohelpers.BuildFileDescriptor(nil, nil, desc)
	if err != nil {
		return "", nil, err
	}

	path = fd.Path()

	symbols = protohelpers.AllSymbolsFromFileDescriptor(fd)
	return path, symbols, nil
}
