package unstructured

import "google.golang.org/protobuf/reflect/protoreflect"

type Object struct {
	m  map[string]interface{}
	md protoreflect.MessageDescriptor
}
