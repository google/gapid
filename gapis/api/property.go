// Copyright (C) 2018 Google Inc.
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

package api

import (
	"reflect"

	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/data/generic"
)

// Property represents a single field on an object.
// A Property has a getter for reading the field value, and an optional setter
// for assigning to the field.
type Property struct {
	// Name of the property.
	Name string
	// Type of the value.
	Type reflect.Type
	// Get gets the property value from the given object.
	Get func() interface{}
	// Set assigns the value to the property on the given object.
	// For read-only properties Set may be nil.
	Set func(value interface{})
	// Constants is the optional index of the constant set used by the value.
	// -1 represents no constant set.
	Constants int
}

// SetConstants is a helper method for setting the Constants field in a
// fluent expression.
func (p *Property) SetConstants(idx int) *Property {
	p.Constants = idx
	return p
}

// Properties is a list of property pointers.
type Properties []*Property

// PropertyProvider is the interface implemented by types that provide
// properties.
type PropertyProvider interface {
	Properties() Properties
}

// NewProperty returns a new Property using the given getter and setter
// functions. set may be nil in the case of a read-only property.
func NewProperty(name string, get, set interface{}) *Property {
	type Value = generic.T1
	var ValueTy = generic.T1Ty
	sigs := []generic.Sig{generic.Sig{Name: "get", Interface: func() Value { return Value{} }, Function: get}}
	if set != nil {
		sigs = append(sigs, generic.Sig{Name: "set", Interface: func(Value) {}, Function: set})
	}
	m := generic.CheckSigs(sigs...)
	if !m.Ok() {
		panic(m.Errors)
	}
	g, s := reflect.ValueOf(get), reflect.ValueOf(set)
	ty := m.Bindings[ValueTy]
	out := &Property{
		Name:      name,
		Type:      ty,
		Get:       func() interface{} { return g.Call([]reflect.Value{})[0].Interface() },
		Constants: -1,
	}
	if set != nil {
		out.Set = func(value interface{}) {
			v := reflect.ValueOf(value)
			if ty == v.Type() {
				// No cast needed.
				s.Call([]reflect.Value{v})
				return
			}
			c := reflect.New(ty)
			if err := deep.Copy(c.Interface(), v.Interface()); err != nil {
				panic(err)
			}
			s.Call([]reflect.Value{c.Elem()})
		}
	}
	return out
}

// Find returns the property with the given name, or nil if no matching
// property is found.
func (l Properties) Find(name string) *Property {
	for _, p := range l {
		if p.Name == name {
			return p
		}
	}
	return nil
}
