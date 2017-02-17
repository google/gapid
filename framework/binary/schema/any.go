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

package schema

import (
	"fmt"

	"github.com/google/gapid/framework/binary"
)

// Any is the schema Type descriptor for a field who's underlying type requires
// boxing and unboxing. The type is usually declared as an empty interface.
type Any struct{}

func (i *Any) Representation() string {
	return fmt.Sprintf("%r", i)
}

func (i *Any) String() string {
	return fmt.Sprint(i)
}

// Format implements the fmt.Formatter interface
func (i *Any) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprint(f, "~")
	default:
		fmt.Fprint(f, "<any>")
	}
}

func (Any) EncodeValue(e binary.Encoder, value interface{}) {
	if boxed, err := binary.Box(value); err != nil {
		e.SetError(err)
	} else {
		e.Variant(boxed)
	}
}

func (Any) DecodeValue(d binary.Decoder) interface{} {
	boxed := d.Variant()
	if d.Error() != nil {
		return nil
	}
	if unboxed, err := binary.Unbox(boxed); err != nil {
		d.SetError(err)
		return nil
	} else {
		return unboxed
	}
}

func (*Any) Subspace() *binary.Subspace {
	return nil
}

func (*Any) HasSubspace() bool {
	return false
}

func (*Any) IsPOD() bool {
	return false
}

func (*Any) IsSimple() bool {
	return false
}
