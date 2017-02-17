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
	"strings"

	"github.com/google/gapid/framework/binary"
)

// Object is an instance of a Class.
type Object struct {
	Type   *ObjectClass
	Fields []interface{}
}

// Class implements binary.Object using the schema system to do the encoding and
// decoding of fields.
func (o *Object) Class() binary.Class {
	return o.Type
}

// Base returns the value of the single, anonymous field of o. If o does not
// have a single anonymous field, then Base returns nil.
func (o *Object) Base() interface{} {
	if len(o.Fields) != 1 {
		return nil // Multiple fields
	}
	if ty := o.Type.Fields[0]; ty.Declared == "" {
		return o.Fields[0]
	}
	return nil // Not anonymous
}

// Underlying traverses the single, anonymous fields nested in v, returning the
// deepest-nested value that is not an object or does not have a single,
// anonymous field.
// If v is not an Object or does not have a single, anonymous field then v is
// returned.
func Underlying(v interface{}) interface{} {
	if o, ok := v.(*Object); ok {
		if b := o.Base(); b != nil {
			return Underlying(b)
		}
	}
	return v
}

func (o *Object) String() string {
	params := make([]string, len(o.Type.Fields))
	for i, f := range o.Type.Fields {
		v := o.Fields[i]
		params[i] = fmt.Sprintf("%v: %v", f.Name(), v)
	}
	return fmt.Sprintf("{%v}", strings.Join(params, ", "))
}
