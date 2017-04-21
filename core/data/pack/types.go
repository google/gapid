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

type (
	// Type is an entry in the map of types stored in a packfile.
	Type struct {
		// Name is the cannocial unique name of the type.
		// This will be the same name as used in the proto registry.
		Name string
		// Index is the tag index used for the type in this packfile.
		Index uint64
		// Type is the reflection type that maps to this type registry.
		Type reflect.Type
		// Descriptor is the proto description of this type, it is packed
		// into the file and can be used to reflect on the type.
		Descriptor *descriptor.DescriptorProto
	}

	// Types stores the full type registry for a packfile.
	// It is exposed so that you can pre-build a cannocial type registry
	// rather than constructing on demand.
	Types struct {
		entries []*Type
		nextTag uint64
		byName  map[string]*Type
		byType  map[reflect.Type]*Type
	}
)

// NewTypes constructs a new empty type registry.
func NewTypes() *Types {
	return &Types{
		entries: []*Type{nil}, // The 0th entry is special, reserve it
		nextTag: 1,
		byName:  map[string]*Type{},
		byType:  map[reflect.Type]*Type{},
	}
}

// Get returns a type given it's tag index.
func (t *Types) Get(index uint64) (Type, bool) {
	if index < t.Count() {
		return *t.entries[index], true
	}
	return Type{}, false
}

// GetName returns a type given it's cannocial type name.
func (t *Types) GetName(name string) Type {
	if entry, found := t.byName[name]; found {
		return *entry
	}
	return Type{}
}

// AddMessage adds a registry entry for a given message if needed.
// It returns the registry entry, and a bool that is true if the entry
// was newly added.
func (t *Types) AddMessage(msg proto.Message) (Type, bool) {
	typ := reflect.TypeOf(msg).Elem()
	name := proto.MessageName(msg)
	return t.add(msg, name, typ)
}

// AddName adds a type by name.
// It uses the proto type registry to look up the name.
func (t *Types) AddName(name string) (Type, bool) {
	typ := proto.MessageType(name)
	if typ == nil {
		return Type{}, false
	}
	typ = typ.Elem()
	msg := reflect.New(typ).Interface().(proto.Message)
	return t.add(msg, name, typ)
}

// AddType adds a type by it's reflection type.
func (t *Types) AddType(typ reflect.Type) (Type, bool) {
	msg := reflect.New(typ).Interface().(proto.Message)
	name := proto.MessageName(msg)
	return t.add(msg, name, typ)
}

// Count returns the number of types in the registry.
func (t *Types) Count() uint64 {
	return uint64(len(t.entries))
}

func (t *Types) add(msg proto.Message, name string, typ reflect.Type) (Type, bool) {
	entry, found := t.byName[name]
	if found {
		return *entry, false
	}
	entry = &Type{
		Name:  name,
		Index: t.nextTag,
		Type:  typ,
	}
	t.nextTag++
	t.entries = append(t.entries, entry)
	t.byName[name] = entry
	t.byType[typ] = entry
	if d, ok := msg.(protoutil.Described); ok {
		entry.Descriptor, _ = protoutil.DescriptorOf(d)
	}
	return *entry, true
}
