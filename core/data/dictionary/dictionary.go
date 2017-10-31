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

// Package dictionary provides utilities for operating on map-like types.
package dictionary

import (
	"reflect"

	"github.com/google/gapid/core/data/slice"
)

// I is the interface for a dictionary.
type I interface {
	Get(key interface{}) (val interface{})
	Set(key, val interface{})
	Lookup(key interface{}) (val interface{}, ok bool)
	Contains(key interface{}) bool
	Delete(key interface{})
	Clear()
	Len() int
	Entries() []Entry
	KeyTy() reflect.Type
	ValTy() reflect.Type
	KeysSorted() []interface{}
}

// Entry holds a key-value pair.
type Entry struct {
	K, V interface{}
}

// Provider is the interface implemented by types that provide a dictionary
// interface.
type Provider interface {
	Dictionary() I
}

// From returns a dictionary wrapping o.
// o can be a Provider or a map, otherwise nil is returned.
func From(o interface{}) I {
	if p, ok := o.(Provider); ok {
		return p.Dictionary()
	}
	v := reflect.ValueOf(o)
	if v.Kind() == reflect.Map {
		return dict{v}
	}
	return nil
}

// dict is an implements I using a reflect.Value of a map.
type dict struct{ reflect.Value }

func (d dict) Get(key interface{}) (val interface{}) {
	v := d.Value.MapIndex(reflect.ValueOf(key))
	if !v.IsValid() {
		return reflect.New(d.ValTy()).Elem().Interface()
	}
	return v.Interface()
}

func (d dict) Set(key, val interface{}) {
	d.Value.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
}

func (d dict) Lookup(key interface{}) (val interface{}, ok bool) {
	v := d.Value.MapIndex(reflect.ValueOf(key))
	if !v.IsValid() {
		return reflect.New(d.ValTy()).Elem().Interface(), false
	}
	return v.Interface(), true
}

func (d dict) Contains(key interface{}) bool {
	return d.Value.MapIndex(reflect.ValueOf(key)).IsValid()
}

func (d dict) Delete(key interface{}) {
	d.Value.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
}

func (d dict) Clear() {
	for _, k := range d.Value.MapKeys() {
		d.Value.SetMapIndex(k, reflect.Value{})
	}
}

func (d dict) Len() int {
	return d.Value.Len()
}

func (d dict) Entries() []Entry {
	keys := d.Value.MapKeys()
	out := make([]Entry, len(keys))
	for i, k := range keys {
		out[i] = Entry{k.Interface(), d.Value.MapIndex(k).Interface()}
	}
	return out
}

func (d dict) KeyTy() reflect.Type {
	return d.Value.Type().Key()
}

func (d dict) ValTy() reflect.Type {
	return d.Value.Type().Elem()
}

func (d dict) KeysSorted() []interface{} {
	keys := d.Value.MapKeys()
	slice.SortValues(keys, d.KeyTy())
	out := make([]interface{}, len(keys))
	for i, k := range keys {
		out[i] = k.Interface()
	}
	return out
}
