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

type boxed interface {
	Unbox() interface{}
}

type Boxer func(interface{}) Object

var boxerList = []Boxer{}

// RegisterBoxer adds a new boxing function to the boxers attempted when an unknown type is encountered.
func RegisterBoxer(boxer Boxer) {
	boxerList = append(boxerList, boxer)
}

// Box returns v wrapped by a struct implementing binary.Object.
// If v is not boxable then ErrUnboxable is returned.
func Box(v interface{}) (Object, error) {
	if v == nil {
		return nil, nil
	}
	for _, boxer := range boxerList {
		boxed := boxer(v)
		if boxed != nil {
			return boxed, nil
		}
	}
	return nil, ErrUnboxable{Value: v}
}

// Unbox returns the value in o wrapped by a call to Box.
// If o is not a boxed value ErrNotBoxedValue is returned.
func Unbox(o Object) (interface{}, error) {
	if o == nil {
		return nil, nil
	} else if o, ok := o.(boxed); ok {
		return o.Unbox(), nil
	}
	return nil, ErrNotBoxedValue{Object: o}
}

// ErrUnboxable is returned when a non-boxable value type is passed to Box.
type ErrUnboxable struct {
	Value interface{} // The value that could not be encoded.
}

// Error returns the error message.
func (e ErrUnboxable) Error() string {
	return fmt.Sprintf("Value of type %T is not boxable", e.Value)
}

// ErrNotBoxedValue is returned when an Object is passed to Unbox that was not
// previously returned by a call to Box.
type ErrNotBoxedValue struct {
	Object Object // The object that is not a boxed value.
}

// Error returns the error message.
func (e ErrNotBoxedValue) Error() string {
	return fmt.Sprintf("Object of type %T was boxed by a call to Box", e.Object)
}
