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

import "fmt"

// TypeTag is an enumerator of class, interface or array.
type TypeTag uint8

const (
	Class     = TypeTag(1) // Type is a class.
	Interface = TypeTag(2) // Type is an interface.
	Array     = TypeTag(3) // Type is an array.
)

func (t TypeTag) String() string {
	switch t {
	case Class:
		return "Class"
	case Interface:
		return "Interface"
	case Array:
		return "Array"
	default:
		return fmt.Sprintf("TypeTag<%v>", int(t))
	}
}
