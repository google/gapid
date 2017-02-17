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
	"strings"
)

// Field describes a single field
type Field struct {
	ID        FieldID
	Name      string
	Signature string
	ModBits   ModBits
}

// Fields is a collection of fields
type Fields []Field

func (l Fields) String() string {
	parts := make([]string, len(l))
	for i, m := range l {
		parts[i] = fmt.Sprintf("%+v", m)
	}
	return strings.Join(parts, "\n")
}

// FindByName returns the field with the matching name, or nil if no field with
// a matching name is found in l.
func (l Fields) FindByName(name string) *Field {
	for _, f := range l {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

// FindBySignature returns the field with the matching signature in l, or nil
// if no field with a matching signature is found in l.
func (l Fields) FindBySignature(name, sig string) *Field {
	for _, f := range l {
		if f.Name == name && f.Signature == sig {
			return &f
		}
	}
	return nil
}

// FindByID returns the field with the matching identifier in l, or nil if no
// field with a matching identifier is found in l.
func (l Fields) FindByID(id FieldID) *Field {
	for _, f := range l {
		if f.ID == id {
			return &f
		}
	}
	return nil
}
