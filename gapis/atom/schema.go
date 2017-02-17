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

package atom

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/gfxapi"
)

// Metadata is the meta information about an atom type that is added to the
// binary schema class for the atom.
type Metadata struct {
	binary.Generate  `java:"AtomMetadata"`
	API              gfxapi.ID // The api this atom belongs to.
	DisplayName      string    // The display name for this atom type.
	EndOfFrame       bool      // Indicates the atom is an end of frame marker.
	DrawCall         bool      // Indicates the atom is a draw call.
	DocumentationUrl string    // A url for documentation about this atom.
}

// FindMetadata finds the atom metadata for the given schema class.
// Returns nil if the class was not for an atom.
func FindMetadata(class *binary.Entity) *Metadata {
	for _, m := range class.Metadata {
		if meta, ok := m.(*Metadata); ok {
			return meta
		}
	}
	return nil
}

// MetadataOf returns the atom metadata for the given atom.
// Returns nil if the atom has no metadata.
func MetadataOf(atom Atom) *Metadata {
	return FindMetadata(atom.Class().Schema())
}

// newMetadata makes a new metadata object based on an example atom.
func newMetadata(example Atom, name string) *Metadata {
	t := reflect.TypeOf(example).Elem()
	if t.Kind() != reflect.Struct {
		panic(fmt.Errorf("Type %v is not struct kind (kind = %v)", t, t.Kind()))
	}
	if t.NumField() == 0 {
		panic(fmt.Errorf("No fields in struct %v (binary.Generated) expected", t))
	}
	f := t.Field(0)
	if !f.Anonymous || f.Type.Kind() != reflect.Struct || f.Type.NumField() != 0 {
		panic(fmt.Errorf(
			"Did not find anonymous empty field in type %v found %v", t, f.Type))
	}

	m := &Metadata{}
	m.API = example.API().ID()
	m.DrawCall = example.AtomFlags()&DrawCall != 0
	m.EndOfFrame = example.AtomFlags()&EndOfFrame != 0
	m.DisplayName = f.Tag.Get("display")
	return m
}

// AddMetadata adds a new Metadata object to the schema entity based on
// an example atom
func AddMetadata(example Atom, ent *binary.Entity) {
	ent.Metadata = append(ent.Metadata, newMetadata(example, ent.Name()))
}
