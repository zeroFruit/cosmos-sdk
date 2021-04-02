package client

import (
	"context"

	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/cosmos/cosmos-sdk/server/grpc/appreflection"
)

// Builder is going to build a client
type Builder struct {
	ctx context.Context

	appDesc *appreflection.AppDescriptor
	grpc    grpc_reflection_v1alpha.ServerReflectionClient
}
