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

// Method describes a single method
type Method struct {
	ID        MethodID
	Name      string
	Signature string
	ModBits   ModBits
}

// Methods is a collection of methods
type Methods []Method

func (l Methods) String() string {
	parts := make([]string, len(l))
	for i, m := range l {
		parts[i] = fmt.Sprintf("%+v", m)
	}
	return strings.Join(parts, "\n")
}

// FindBySignature returns the method with the matching signature in l, or nil
// if no method with a matching signature is found in l.
func (l Methods) FindBySignature(name, sig string) *Method {
	for _, m := range l {
		if m.Name == name && m.Signature == sig {
			return &m
		}
	}
	return nil
}

// FindByID returns the method with the matching identifier in l, or nil if no
// method with a matching identifier is found in l.
func (l Methods) FindByID(id MethodID) *Method {
	for _, m := range l {
		if m.ID == id {
			return &m
		}
	}
	return nil
}
