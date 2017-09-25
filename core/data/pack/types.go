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
	entries []*ty
	byName  map[string]*ty
}

// newTypes constructs a new empty type registry.
func newTypes() *types {
	return &types{
		entries: []*ty{nil},
		byName:  map[string]*ty{},
	}
}

// addMessage adds a registry entry for a given message if needed.
// It returns the registry entry, and a bool that is true if the entry
// was newly added.
func (t *types) addMessage(msg proto.Message) (*ty, bool) {
	typ := reflect.TypeOf(msg)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	msg = reflect.New(typ).Interface().(proto.Message)
	name := proto.MessageName(msg)
	if ty, ok := t.byName[name]; ok {
		return ty, false
	}
	// TODO: We are missing all referenced types. Use FileDescriptorProto instead?
	var desc *descriptor.DescriptorProto
	if d, ok := msg.(protoutil.Described); ok {
		desc, _ = protoutil.DescriptorOf(d)
	}
	return t.add(name, desc), true
}

// count returns the number of types in the registry.
func (t *types) count() uint64 {
	return uint64(len(t.entries))
}

// add adds a type by name and descriptor.
func (t *types) add(name string, desc *descriptor.DescriptorProto) *ty {
	create := func() proto.Message { return newDynamic(desc, t) }
	if ty := proto.MessageType(name); ty != nil {
		create = func() proto.Message { return reflect.New(ty).Interface().(proto.Message) }
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
