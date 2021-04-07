package codec

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/cosmos/cosmos-sdk/rosetta/dynamic/codec/protohelpers"
)

type InterfaceImplementer struct {
	FullName protoreflect.FullName
	TypeURL  string
}

type InterfaceDescriptor struct {
	Name string // it's not protoreflect.FullName because the interface name is a custom sdk thing
}

// ReflectionInfoProvider defines an interface which provides information
// on the application we're trying to build the codec for.
// NOTE: it's fine if ListMessages, ListInterfaceImplementations contain
// messages which have already been parsed while calling ListServices.
type ReflectionInfoProvider interface {
	// ListMessages lists the proto.Messages which may not have been exposed
	// when resolving proto.Messages from ListServices.
	ListMessages(ctx context.Context) ([]protoreflect.FullName, error)
	// ListServices lists the services of the application
	ListServices(ctx context.Context) ([]protoreflect.FullName, error)
	// ListInterfaces lists the interfaces expressed as anypb.Any types
	ListInterfaces(ctx context.Context) ([]InterfaceDescriptor, error)
	// ListInterfaceImplementations lists the interface implementer given an interface name
	ListInterfaceImplementations(ctx context.Context, interfaceName string) ([]InterfaceImplementer, error)
}

// RemoteProtobufRegistry defines an external protobuf registry which we can use
// to fetch files and files associated with a specific protoreflect.FullName
type RemoteProtobufRegistry interface {
	// FilepathFromFullName returns the path of a proto file given a protobuf declaration fullname
	FilepathFromFullName(ctx context.Context, fullname protoreflect.FullName) (path string, err error)
	// FileDescriptorBytes returns the descriptor bytes of a file given its path
	FileDescriptorBytes(ctx context.Context, filepath string) (rawDesc []byte, err error)
}

func NewBuilder(infoProvider ReflectionInfoProvider, externalRegistry RemoteProtobufRegistry) *Builder {
	return &Builder{
		ctx:              nil,
		rip:              infoProvider,
		externalRegistry: externalRegistry,
		files:            new(protoregistry.Files),
		types:            newTypesRegistry(),
		parsedFiles:      make(map[string]struct{}),
	}
}

// Builder builds a Codec
type Builder struct {
	state int32

	ctx context.Context

	rip              ReflectionInfoProvider
	externalRegistry RemoteProtobufRegistry

	files *protoregistry.Files
	types *typesRegistry

	parsedFiles map[string]struct{}
}

func (b *Builder) Build(ctx context.Context) (*Codec, error) {
	const stateNotBuilding int32 = 0
	const stateBuilding int32 = 1
	if !atomic.CompareAndSwapInt32(&b.state, stateNotBuilding, stateBuilding) {
		return nil, fmt.Errorf("codec: Builder.Build can only be called once, and not concurrently")
	}
	b.ctx = ctx
	// parse services
	err := b.parseServices()
	if err != nil {
		return nil, fmt.Errorf("unable to parse services: %w", err)
	}
	// after we parse the service we can parse msgs
	err = b.parseMessages()
	if err != nil {
		return nil, fmt.Errorf("unable to parse msgs: %w", err)
	}
	// then the last step is parsing the Any resolvable types in interfaces
	err = b.parseAnys()
	if err != nil {
		return nil, err
	}
	return &Codec{
		filesRegistry: b.files,
		typeRegistry:  b.types,
		jsonMarshaler: protojson.MarshalOptions{
			Resolver: b.types,
		},
		jsonUnmarshaler: protojson.UnmarshalOptions{
			Resolver: b.types,
		},
		protoMarshaler: proto.MarshalOptions{
			Deterministic: true,
		},
		protoUnmarsahler: proto.UnmarshalOptions{
			Resolver: b.types,
		},
	}, nil
}

func (b *Builder) parseServices() error {
	svcs, err := b.rip.ListServices(b.ctx)
	if err != nil {
		return fmt.Errorf("unable to get services: %s", err)
	}

	for _, svc := range svcs {
		// we get the service file
		filepath, err := b.externalRegistry.FilepathFromFullName(b.ctx, svc)
		if err != nil {
			return fmt.Errorf("unable to get filepath for service: %s", err)
		}
		_, err = b.registerFile(filepath)
		if err != nil {
			return fmt.Errorf("unable to parse descriptor for service %s: %w", svc, err)
		}
	}

	return nil
}

func (b *Builder) parseMessages() error {
	// get the msg descriptors
	msgs, err := b.rip.ListMessages(b.ctx)
	if err != nil {
		return fmt.Errorf("unable to list msgs: %w", err)
	}
	for _, msg := range msgs {
		path, err := b.externalRegistry.FilepathFromFullName(b.ctx, msg)
		if err != nil {
			return fmt.Errorf("unable to get filepath for message %s: %w", msg, err)
		}
		// check if this file was already parsed
		_, err = b.files.FindFileByPath(path)
		if err == nil {
			continue
		}
		if !errors.Is(err, protoregistry.NotFound) {
			return fmt.Errorf("unknown protoregistry error whilst parsing message %s: %w", msg, err)
		}
		// parse the file
		_, err = b.registerFile(path)
		if err != nil {
			return fmt.Errorf("unable to register descriptor for msg %s: %w", msg, err)
		}
	}
	return nil
}

// parseAnys should build the marshallers and unmarshallers for types
// alongside registering type URLs for proto.Messages
func (b *Builder) parseAnys() error {
	interfaces, err := b.rip.ListInterfaces(b.ctx)
	if err != nil {
		return fmt.Errorf("unable to list interfaces: %w", err)
	}

	for _, iface := range interfaces {
		impls, err := b.rip.ListInterfaceImplementations(b.ctx, iface.Name)
		if err != nil {
			return fmt.Errorf("unable to list interface implementations for %s: %w", iface.Name, err)
		}

		for _, impl := range impls {
			// check if protobuf message is already registered
			_, err = b.types.FindMessageByName(impl.FullName)
			if err != nil && !errors.Is(err, protoregistry.NotFound) {
				return fmt.Errorf("unknown types registry error while registering type %s for interface %s: %w", impl.FullName, iface.Name, err)
			}
			if errors.Is(err, protoregistry.NotFound) {
				filepath, err := b.externalRegistry.FilepathFromFullName(b.ctx, impl.FullName)
				if err != nil {
					return fmt.Errorf("unable to find protofile for type %s for interface %s: %w", impl.FullName, iface.Name, err)
				}
				_, err = b.registerFile(filepath)
				if err != nil {
					return fmt.Errorf("unable to register protofile for type %s for interface %s: %w", impl.FullName, iface.Name, err)
				}
			}
			// now register alias
			err = b.types.registerTypeURL(impl.TypeURL, impl.FullName)
			if err != nil {
				return fmt.Errorf("unable to register type URL for %s while registering interface %s: %w", impl.FullName, iface.Name, err)
			}
		}
	}
	// TODO create message scoped proto/protojson.Marshal/UnmarshalOptions for messages which contain *anypb.Any fields
	return nil
}

// registerFile recursively parses descriptors, resolving dependencies
// in a transitive way.
func (b *Builder) registerFile(filepath string) (fd protoreflect.FileDescriptor, err error) {
	if _, exists := b.parsedFiles[filepath]; exists {
		// TODO(fdymylja): we could provide the cycling import
		return nil, fmt.Errorf("bad descriptor sequence, cyclic importing detected in file %s", filepath)
	}

	// add filepath to check for cyclic importing; the execution of Build is atomic: it can either fail or be successful
	// we can't have a partial successful state
	b.parsedFiles[filepath] = struct{}{}

	rawDesc, err := b.externalRegistry.FileDescriptorBytes(b.ctx, filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to get descriptor bytes for file %s: %w", err)
	}
	// try to unzip
	rawDesc, err = protohelpers.Unzip(rawDesc)
	if err != nil {
		return nil, err
	}
	// first we get the dependencies of the descriptor
	dependencies, err := protohelpers.GetDependenciesFromBytesDescriptor(rawDesc)
	if err != nil {
		return nil, fmt.Errorf("unable to get dependencies for descriptor: %w", err)
	}
	// after we've got those we check if all of them are present
	for _, dep := range dependencies {
		// if the file is present then all good, we can move forward
		if _, err = b.files.FindFileByPath(dep); err == nil {
			continue
		}
		// if error is not a NotFound one, then it's something we can't handle
		if !errors.Is(err, protoregistry.NotFound) {
			return nil, fmt.Errorf("unknown protofiles registry error: %w", err)
		}

		_, err = b.registerFile(dep)
		if err != nil {
			return nil, fmt.Errorf("unable to parse dependency %s: %w", dep, err)
		}
	}
	// after we've resolved all the dependencies we can build the descriptor
	fd, err = protohelpers.BuildFileDescriptor(b.files, b.types, rawDesc)
	if err != nil {
		return nil, fmt.Errorf("unable to build descriptor: %w", err)
	}
	err = b.types.registerFileDescriptorTypes(fd)
	if err != nil {
		return nil, fmt.Errorf("unable to register file descriptor types: %w", err)
	}
	return fd, nil
}
