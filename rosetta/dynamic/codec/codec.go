package codec

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Codec is a dynamic codec which builds a representation
// of the protobuf registry of the target application
// with the expected anypb.Any type resolvers of types
// which contain anypb.Any fields.
type Codec struct {
	jsonMarshaler   protojson.MarshalOptions
	jsonUnmarshaler protojson.UnmarshalOptions

	protoMarshaler   proto.MarshalOptions
	protoUnmarsahler proto.UnmarshalOptions
}
