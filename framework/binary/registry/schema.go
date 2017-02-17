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

package registry

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/framework/binary"
)

// SchemaFactory is a function type used to make schema types from
// Go reflect types.
type SchemaFactory binary.MakeTypeFun

// MetaFactory is a function type used to register metadata factories
// The function takes the type of a binary object and the schema entity
// (which has already been constructed). The meta factory can add
// additionally metadata to the entity.
type MetaFactory func(reflect.Type, *binary.Entity)

// TypeKey represents a key for a unique key for a type.
type typeKey struct {
	pkgPath string
	name    string
	str     string
	kind    reflect.Kind
}

// SchemaByKind provides a mapping from reflect.Kind to a SchemaFactory
// and a list of MetaFactory to add metadata to structs.
type SchemaByKind struct {
	factories     map[reflect.Kind]SchemaFactory
	metaFactories []MetaFactory
}

var (
	// Factories is a singleton for registration of schema type factories.
	Factories = &SchemaByKind{factories: make(map[reflect.Kind]SchemaFactory)}
)

// Add adds a schema factory for a specific kind to the registry
func (s *SchemaByKind) Add(k reflect.Kind, f SchemaFactory) {
	if s.factories[k] != nil {
		panic(fmt.Errorf("Attempt for register multiple factories for %v", k))
	}
	s.factories[k] = f
}

// AddMetaFactory adds a meta factory to the registry.
func (s *SchemaByKind) AddMetaFactory(f MetaFactory) {
	s.metaFactories = append(s.metaFactories, f)
}

// makeType uses the appropriate factory to build a schema type for 't'.
// If 't' has sub-types 'makeFun' is used to build them.
func (s *SchemaByKind) makeType(t reflect.Type, tag reflect.StructTag, makeFun binary.MakeTypeFun, pkg string) binary.Type {
	kind := t.Kind()
	if f, ok := s.factories[kind]; !ok {
		panic(fmt.Errorf("factory: no factory for type %v kind %v", t, kind))
	} else {
		return f(t, tag, makeFun, pkg)
	}
}

// addMetadata runs any registered factories to add metadata to the
// specified entity for the reflected type t.
func (s *SchemaByKind) addMetadata(t reflect.Type, e *binary.Entity) {
	for _, f := range s.metaFactories {
		f(t, e)
	}
}

// makeKey builds a unique key for 't' of reflect.Type
func makeKey(t reflect.Type) typeKey {
	return typeKey{
		pkgPath: t.PkgPath(),
		name:    t.Name(),
		str:     t.String(),
		kind:    t.Kind(),
	}
}

// PreventLoopsFun returns a factory function which wraps 'f' in a layer
// which prevents a loop on 't' by returning 'loop'.
func PreventLoopsFun(t reflect.Type, loop binary.Type, makeType binary.MakeTypeFun) binary.MakeTypeFun {
	k := makeKey(t)
	makeFun := func(tt reflect.Type, tag reflect.StructTag, mf binary.MakeTypeFun, pkg string) binary.Type {
		if k == makeKey(tt) {
			return loop
		} else {
			return makeType(tt, tag, mf, pkg)
		}
	}
	return binary.MakeTypeFun(makeFun)
}

// PreventLoops uses the factory 'f' to make the schema type for 'make'.
// It prevents loops whilst building 'make' by returning 'loop' for 't'.
func PreventLoops(t reflect.Type, tag reflect.StructTag, loop binary.Type, f binary.MakeTypeFun, make reflect.Type, pkg string) binary.Type {
	newFun := PreventLoopsFun(t, loop, f)
	return newFun(make, tag, newFun, pkg)
}

// InitSchemaOf initializes a schema entity from the reflection information
// from the example object, which should be a nil pointer to the appropriate
// type. The global factory is used.
func InitSchemaOf(obj binary.Object) {
	t := reflect.TypeOf(obj).Elem()
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic during make entity on %T (%v) class %T: %v",
				obj, t, obj.Class(), r))
		}
	}()
	binary.InitEntity(t, Factories.makeType, obj.Class().Schema())
	Factories.addMetadata(t, obj.Class().Schema())
}
