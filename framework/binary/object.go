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

package binary

import "fmt"

// Object is the interface to any class that wants to be encoded/decoded.
type Object interface {
	// Class returns the serialize information and functionality for this type.
	// The method should be valid on a nil pointer.
	Class() Class
}

// ObjectCast is automatically called by the generated decoders.
func ObjectCast(obj Object) Object {
	return obj
}

// UpgradeDecoder provides a decoder interface which maybe used to
// decode a stream from an old version into a newer version.
type UpgradeDecoder interface {
	// New constructs a new Object that this decoder handles.
	// For a frozen decoder, this would be the new version, not the frozen one, and as such the type returned may not
	// match the schema this decoder was registered against.
	New() Object
	// DecodeTo reads into the supplied object from the supplied Decoder.
	// The object must be the same concrete type that New would create, the
	// implementation is allowed to panic if it is not.
	DecodeTo(Decoder, Object)
}

// Class represents a struct type in the binary registry.
type Class interface {
	// Encode writes the supplied object to the supplied Encoder.
	// The object must be a type the Class understands, the implementation is
	// allowed to panic if it is not.
	Encode(Encoder, Object)

	// DecodeTo reads into the supplied object from the supplied Decoder.
	// The object must be a type the Class understands, the implementation is
	// allowed to panic if it is not.
	DecodeTo(Decoder, Object)

	// Returns the type descriptor for the class.
	Schema() *Entity
}

type FrozenClassBase struct{}

// FrozenClassBase provides Encode() so that they satisfy the above Class
// interface, but they should never be called.
func (c *FrozenClassBase) Encode(e Encoder, o Object) {
	e.SetError(fmt.Errorf(
		"Attempt to encode a frozen object of type %T id %v", o, o.Class().Schema()))
}

// Generate is used to tag structures that need an auto generated Class.
// The codergen function searches packages for structs that have this type as an
// anonymous field, and then automatically generates the encoding and decoding
// functionality for those structures. For example, the following struct would
// create the Class with methods needed to encode and decode the Name and Value
// fields, and register that class.
// The embedding will also fully implement the binary.Object interface, but with
// methods that panic. This will get overridden with the generated Methods.
// This is important because it means the package is resolvable without the
// generated code, which means the types can be correctly evaluated during the
// generation process.
//
// type MyNamedValue struct {
//    binary.Generate
//    Name  string
//    Value []byte
// }
//
// Code generation can be customized with tags:
// type MySomethingThatWontHaveAJavaEquivalent struct {
//     binary.Generate `java:"disable"`
// }
//
// We currently use the following tags:
//   cpp
//   java
//   disable
//   display
//   globals
//   id
//   identity
//   implements
//   javaDefineEmptyArrays:"true"
//      If applied to a struct, any slice fields being read from the wire that
//      have a length of zero will be set to the same instance of an empty array
//      to avoid unnecessary allocations. e.g. ...service.atom.Observations.
//   version
//
type Generate struct{}

func (Generate) Class() Class { panic("Class() not implemented") }

var _ Object = Generate{} // Verify that Generate implements Object.

// Frozen is used to tag structures that represent past versions of types
// which may appear in old capture files. Frozen must be include a "name" tag
// which identifies the name of the struct now in use. Manual upgrading is
// provided with a function of the form:
//
//  func (before *FrozenStruct) upgrade(after *GenerateStruct)
//
// See binary/test/frozen.go for examples
type Frozen struct{}

func (Frozen) Class() Class { panic("Class() not implemented") }

var _ Object = Frozen{} // Verify that Frozen implements Object.
