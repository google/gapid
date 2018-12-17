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

package jdwp

import (
	"fmt"
	"sort"
)

// TaggedObjectID is a type and object identifier pair.
type TaggedObjectID struct {
	Type   Tag
	Object ObjectID
}

// Location describes a code location.
type Location struct {
	Type     TypeTag
	Class    ClassID
	Method   MethodID
	Location uint64
}

// Char is a 16-bit character type.
type Char int16

// ObjectID is an object instance identifier.
// If the specific object type is known, then ObjectID can be cast to
// ThreadID, ThreadGroupID, StringID, ClassLoaderID, ClassObjectID or ArrayID.
type ObjectID uint64

// ThreadID is an thread instance identifier.
// ThreadID can always be safely cast to the less specific ObjectID.
type ThreadID uint64

// ThreadGroupID is an thread group identifier.
// ThreadGroupID can always be safely cast to the less specific ObjectID.
type ThreadGroupID uint64

// StringID is a string instance identifier.
// StringID can always be safely cast to the less specific ObjectID.
type StringID uint64

// ClassLoaderID is class loader identifier.
// ClassLoaderID can always be safely cast to the less specific ObjectID.
type ClassLoaderID uint64

// ClassObjectID is a class object instance identifier.
// ClassObjectID can always be safely cast to the less specific ObjectID.
type ClassObjectID uint64

// ArrayID is an array instance identifier.
// ArrayID can always be safely cast to the less specific ObjectID.
type ArrayID uint64

// Object is the interface implemented by all types that are a variant of ObjectID.
type Object interface {
	ID() ObjectID
}

// ID returns the ObjectID
func (i ObjectID) ID() ObjectID { return i }

// ID returns the ThreadID as an ObjectID
func (i ThreadID) ID() ObjectID { return ObjectID(i) }

// ID returns the ThreadGroupID as an ObjectID
func (i ThreadGroupID) ID() ObjectID { return ObjectID(i) }

// ID returns the StringID as an ObjectID
func (i StringID) ID() ObjectID { return ObjectID(i) }

// ID returns the ClassLoaderID as an ObjectID
func (i ClassLoaderID) ID() ObjectID { return ObjectID(i) }

// ID returns the ClassObjectID as an ObjectID
func (i ClassObjectID) ID() ObjectID { return ObjectID(i) }

// ID returns the ArrayID as an ObjectID
func (i ArrayID) ID() ObjectID { return ObjectID(i) }

// ID returns the ObjectID of the TaggedObjectID
func (i TaggedObjectID) ID() ObjectID { return i.Object }

// ReferenceTypeID is a reference type identifier.
// If the specific reference type is known, then ReferenceTypeID can be cast to
// ClassID, InterfaceID or ArrayTypeID.
type ReferenceTypeID uint64

// ClassID is a class reference type identifier.
// ClassID can always be safely cast to the less specific ReferenceTypeID.
type ClassID uint64

// InterfaceID is an interface reference type identifier.
// InterfaceID can always be safely cast to the less specific ReferenceTypeID.
type InterfaceID uint64

// ArrayTypeID is an array reference type identifier.
// ArrayTypeID can always be safely cast to the less specific ReferenceTypeID.
type ArrayTypeID uint64

// MethodID is the identifier for a single method for a class or interface.
type MethodID uint64

// FieldID is the identifier for a single method for a class or interface.
type FieldID uint64

// FrameID is the identifier for a stack frame.
type FrameID uint64

// FrameVariable contains all of the information a single variable.
type FrameVariable struct {
	CodeIndex uint64
	Name      string
	Signature string
	Length    int
	Slot      int
}

// VariableTable contains all of the variables for a stack frame.
type VariableTable struct {
	ArgCount int
	Slots    []FrameVariable
}

func (i ObjectID) String() string        { return fmt.Sprintf("ObjectID<%d>", uint64(i)) }
func (i ThreadID) String() string        { return fmt.Sprintf("ThreadID<%d>", uint64(i)) }
func (i ThreadGroupID) String() string   { return fmt.Sprintf("ThreadGroupID<%d>", uint64(i)) }
func (i StringID) String() string        { return fmt.Sprintf("StringID<%d>", uint64(i)) }
func (i ClassLoaderID) String() string   { return fmt.Sprintf("ClassLoaderID<%d>", uint64(i)) }
func (i ClassObjectID) String() string   { return fmt.Sprintf("ClassObjectID<%d>", uint64(i)) }
func (i ArrayID) String() string         { return fmt.Sprintf("ArrayID<%d>", uint64(i)) }
func (i ReferenceTypeID) String() string { return fmt.Sprintf("ReferenceTypeID<%d>", uint64(i)) }
func (i ClassID) String() string         { return fmt.Sprintf("ClassID<%d>", uint64(i)) }
func (i InterfaceID) String() string     { return fmt.Sprintf("InterfaceID<%d>", uint64(i)) }
func (i ArrayTypeID) String() string     { return fmt.Sprintf("ArrayTypeID<%d>", uint64(i)) }
func (i MethodID) String() string        { return fmt.Sprintf("MethodID<%d>", uint64(i)) }
func (i FieldID) String() string         { return fmt.Sprintf("FieldID<%d>", uint64(i)) }
func (i FrameID) String() string         { return fmt.Sprintf("FrameID<%d>", uint64(i)) }

// ArgumentSlots returns the slots that could possibly be method arguments.
// Slots that could be method arguments are slots that are acessible at
// location 0 and have a length > 0. Returns the result sorted by slot index.
func (v *VariableTable) ArgumentSlots() []FrameVariable {
	r := []FrameVariable{}
	for _, slot := range v.Slots {
		if slot.CodeIndex == 0 && slot.Length > 0 {
			r = append(r, slot)
		}
	}
	sort.Slice(r, func(i, j int) bool {
		return r[i].Slot < r[j].Slot
	})
	return r
}
