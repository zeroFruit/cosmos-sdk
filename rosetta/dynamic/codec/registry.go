package codec

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

var _ protoregistry.MessageTypeResolver = (*typesRegistry)(nil)
var _ protoregistry.ExtensionTypeResolver = (*typesRegistry)(nil)

func newTypesRegistry() *typesRegistry {
	return &typesRegistry{reg: new(protoregistry.Types), typeURLs: make(map[string]protoreflect.FullName)}
}

type typesRegistry struct {
	reg      *protoregistry.Types
	typeURLs map[string]protoreflect.FullName
}

func (t typesRegistry) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return t.reg.FindMessageByName(message)
}

func (t typesRegistry) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	fullname, ok := t.typeURLs[url]
	if !ok {
		return nil, protoregistry.NotFound
	}
	messageType, err := t.reg.FindMessageByName(fullname)
	return messageType, err
}

func (t typesRegistry) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return t.reg.FindExtensionByName(field)
}

func (t typesRegistry) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return t.reg.FindExtensionByNumber(message, field)
}

func (t typesRegistry) registerTypeURL(typeURL string, fullname protoreflect.FullName) error {
	gotFullname, exists := t.typeURLs[typeURL]
	if !exists {
		t.typeURLs[typeURL] = fullname
		return nil
	}
	if gotFullname != fullname {
		return fmt.Errorf("disallowed overwrite of typeURL with wrong proto type %s: %s <-> %s", typeURL, fullname, gotFullname)
	}
	return nil
}

func (t typesRegistry) registerFileDescriptorTypes(fd protoreflect.FileDescriptor) error {
	mds := fd.Messages()
	for i := 0; i < mds.Len(); i++ {
		md := mds.Get(i)
		typ := dynamicpb.NewMessageType(md)
		err := t.reg.RegisterMessage(typ)
		if err != nil {
			return err
		}
	}

	eds := fd.Enums()
	for i := 0; i < eds.Len(); i++ {
		ed := eds.Get(i)
		typ := dynamicpb.NewEnumType(ed)
		err := t.reg.RegisterEnum(typ)
		if err != nil {
			return err
		}
	}

	xds := fd.Extensions()
	for i := 0; i < xds.Len(); i++ {
		xd := xds.Get(i)
		typ := dynamicpb.NewExtensionType(xd)
		err := t.reg.RegisterExtension(typ)
		if err != nil {
			return err
		}
	}

	return nil
}
