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

package jdbg

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/java/jdwp"
)

var nilValue Value

// Value holds the value of a call.
type Value struct {
	ty  Type
	val interface{}
}

func newValue(ty Type, val interface{}) Value {
	if obj, ok := val.(jdwp.Object); ok {
		// Prevent GC of this object for the duration of the jdbg.Do call.
		j, id := ty.jdbg(), obj.ID()
		j.conn.DisableGC(id)
		j.objects = append(j.objects, id)
	}
	return Value{ty, val}
}

// Call invokes the method on the value.
func (v Value) Call(method string, args ...interface{}) Value {
	return v.ty.call(v, method, args)
}

// Field returns the value of the specified field.
func (v Value) Field(name string) Value {
	return v.ty.field(v, name)
}

// Get returns the value, unmarshalled.
func (v Value) Get() interface{} {
	return v.ty.jdbg().unmarshal(v.val)
}

// Type returns the value's type.
func (v Value) Type() Type {
	return v.ty
}

// AsType returns the value as a type.
func (v Value) AsType() Type {
	j := v.ty.jdbg()
	switch v := v.val.(type) {
	case jdwp.ClassObjectID:
		id, err := j.conn.ReflectedType(v)
		if err != nil {
			j.fail("%v", err)
		}
		return j.typeFromID(id)
	}
	panic(fmt.Errorf("Unhandled value type: %T %+v", v.val, v.val))
}

// SetArrayValues sets the array values to values. This value must be an Array.
func (v Value) SetArrayValues(values interface{}) {
	j := v.ty.jdbg()
	arrayTy, ok := v.ty.(*Array)
	if !ok {
		j.fail("SetArrayValues can only be used with Arrays, type is %v", v.ty)
	}
	r := reflect.ValueOf(values)
	if r.Kind() != reflect.Array && r.Kind() != reflect.Slice {
		j.fail("values must be an array or slice, got %v", r.Kind())
	}

	if !j.assignableT(v.ty, reflect.TypeOf(values)) {
		j.fail("value elements (type %T) does not match array element type %v", values, arrayTy.el)
	}

	if arrayTy.el == j.cache.objTy {
		values = j.toObjects(values.([]interface{}))
	}

	if err := j.conn.SetArrayValues(v.val.(jdwp.ArrayID), 0, values); err != nil {
		j.fail("Failed to set array (type %s) values (type %T): %v", arrayTy, values, err)
	}
}
