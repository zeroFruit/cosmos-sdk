package protohelpers

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoimpl"
)

type TypesRegistry interface {
	protoregistry.ExtensionTypeResolver
}

type FilesRegistry interface {
	FindFileByPath(string) (protoreflect.FileDescriptor, error)
	FindDescriptorByName(protoreflect.FullName) (protoreflect.Descriptor, error)
	RegisterFile(protoreflect.FileDescriptor) error
}

func Unzip(desc []byte) ([]byte, error) {
	buf := bytes.NewBuffer(desc)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return desc, nil
	}

	unzipped, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return unzipped, nil
}

func GetDependenciesFromBytesDescriptor(desc []byte) ([]string, error) {
	fd, err := BuildFileDescriptor(new(protoregistry.Files), new(protoregistry.Types), desc)
	if err != nil {
		return nil, err
	}
	dependencies := make([]string, fd.Imports().Len())

	for i := 0; i < fd.Imports().Len(); i++ {
		dep := fd.Imports().Get(i)
		dependencies[i] = dep.Path()
	}
	return dependencies, nil
}

func BuildFileDescriptor(files FilesRegistry, types TypesRegistry, descBytes []byte) (fd protoreflect.FileDescriptor, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	fd = (&protoimpl.DescBuilder{
		RawDescriptor: descBytes,
		TypeResolver:  types,
		FileRegistry:  files,
	}).Build().File
	return fd, nil
}

func AllSymbolsFromRawDescBytes(rawDesc []byte) (symbols []protoreflect.FullName, fd protoreflect.FileDescriptor, err error) {
	fd, err = BuildFileDescriptor(new(protoregistry.Files), new(protoregistry.Types), rawDesc)
	if err != nil {
		return nil, nil, err
	}
	return AllSymbolsFromFileDescriptor(fd), fd, nil
}

func AllSymbolsFromFileDescriptor(fd protoreflect.FileDescriptor) []protoreflect.FullName {
	mds := fd.Messages()
	eds := fd.Enums()
	sds := fd.Services()
	xds := fd.Extensions()

	names := make([]protoreflect.FullName, 0, mds.Len()+eds.Len()+sds.Len()+xds.Len())

	for i := 0; i < mds.Len(); i++ {
		md := mds.Get(i)
		names = append(names, md.FullName())
	}

	for i := 0; i < eds.Len(); i++ {
		ed := eds.Get(i)
		names = append(names, ed.FullName())
	}

	for i := 0; i < sds.Len(); i++ {
		sd := sds.Get(i)
		names = append(names, sd.FullName())
	}

	for i := 0; i < xds.Len(); i++ {
		xd := xds.Get(i)
		names = append(names, xd.FullName())
	}

	return names
}
