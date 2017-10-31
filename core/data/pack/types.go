// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pack

import (
	"context"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/google/gapid/core/data/protoutil"
)

// ty is an entry in the map of types stored in a packfile.
type ty struct {
	// name is the cannocial unique name of the type.
	// This will be the same name as used in the proto registry.
	name string
	// index is the tag index used for the type in this packfile.
	index uint64
	// create constructs a new proto of this type.
	create func() proto.Message
	// desc is the proto description of this type, it is packed
	// into the file and can be used to reflect on the type.
	desc *descriptor.DescriptorProto
}

// types stores the full type registry for a packfile.
// It is exposed so that you can pre-build a cannocial type registry
// rather than constructing on demand.
type types struct {
	entries      []*ty
	byName       map[string]*ty
	forceDynamic bool
}

// newTypes constructs a new empty type registry.
func newTypes(forceDynamic bool) *types {
	return &types{
		entries:      []*ty{nil},
		byName:       map[string]*ty{},
		forceDynamic: forceDynamic,
	}
}

// addForMessage adds all types required to serialize the message.
// The callback will be called for each newly added type.
func (t *types) addForMessage(ctx context.Context, msg proto.Message, cb func(t *ty) error) (*ty, error) {
	return t.addForType(ctx, reflect.TypeOf(msg), cb)
}

// addForType adds all types required to serialize the reflection type.
// The callback will be called for each newly added type.
func (t *types) addForType(ctx context.Context, typ reflect.Type, cb func(t *ty) error) (*ty, error) {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	msg := reflect.New(typ).Interface().(proto.Message)
	name := proto.MessageName(msg)
	if name == "" {
		panic(fmt.Errorf("Can not determine the message name of: %v", msg))
	}
	if ty, ok := t.byName[name]; ok {
		return ty, nil
	}
	var desc *descriptor.DescriptorProto
	if d, ok := msg.(protoutil.Described); ok {
		desc, _ = protoutil.DescriptorOf(d)
	}
	ty := t.add(name, desc)
	if err := cb(ty); err != nil {
		return nil, err
	}

	// Recursively add referenced types.
	if typ.Kind() == reflect.Struct {
		numFields := typ.NumField()
		protoMsgTy := reflect.TypeOf((*proto.Message)(nil)).Elem()
		for i := 0; i < numFields; i++ {
			fieldType := typ.Field(i).Type
			for fieldType.Kind() == reflect.Slice {
				fieldType = fieldType.Elem()
			}
			if fieldType.Implements(protoMsgTy) {
				if _, err := t.addForType(ctx, fieldType, cb); err != nil {
					return nil, err
				}
			}
		}
	}

	return ty, nil
}

// count returns the number of types in the registry.
func (t *types) count() uint64 {
	return uint64(len(t.entries))
}

// add adds a type by name and descriptor.
func (t *types) add(name string, desc *descriptor.DescriptorProto) *ty {
	create := func() proto.Message { return newDynamic(desc, t) }
	if !t.forceDynamic {
		if ty := proto.MessageType(name); ty != nil {
			create = func() proto.Message { return reflect.New(ty.Elem()).Interface().(proto.Message) }
		}
	}
	entry := &ty{
		name:   name,
		index:  t.count(),
		create: create,
		desc:   desc,
	}
	t.entries = append(t.entries, entry)
	t.byName[name] = entry
	return entry
}
