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
	// Get returns the value of the entry with the given key.
	Get(key interface{}) (val interface{})
	// Add inserts the key-value pair, replacing any existing entry with the
	// same key.
	Add(key, val interface{})
	// Lookup searches for the value of the entry with the given key.
	Lookup(key interface{}) (val interface{}, found bool)
	// Contains returns true if the dictionary contains an entry with the given
	// key.
	Contains(key interface{}) bool
	// Remove removes the entry with the given key. If no entry with the given
	// key exists then this call is a no-op.
	Remove(key interface{})
	// Len returns the number of entries in the dictionary.
	Len() int
	// Keys returns all the entry keys in the map.
	Keys() []interface{}
	// KeyTy returns the type of all the entry keys.
	KeyTy() reflect.Type
	// ValTy returns the type of all the entry values.
	ValTy() reflect.Type
}

// Entry holds a key-value pair.
type Entry struct {
	K, V interface{}
}

// Clear removes all entries from the dictionary d.
func Clear(d I) {
	for _, key := range d.Keys() {
		d.Remove(key)
	}
}

// Entries returns the full list of entries in the dictionary d.
func Entries(d I) []Entry {
	keys := d.Keys()
	out := make([]Entry, len(keys))
	for i, key := range keys {
		out[i] = Entry{key, d.Get(key)}
	}
	return out
}

// From returns a dictionary wrapping o.
// o can be a map or a Source, otherwise nil is returned.
func From(o interface{}) I {
	if o == nil {
		return nil
	}
	v := reflect.ValueOf(o)
	if s := newSource(v); s != nil {
		return s
	}
	if v.Kind() == reflect.Map {
		return dict{v}
	}
	return nil
}

// dict implements I using a reflect.Value of a map.
type dict struct{ reflect.Value }

func (d dict) Get(key interface{}) (val interface{}) {
	v := d.Value.MapIndex(reflect.ValueOf(key))
	if !v.IsValid() {
		return reflect.New(d.ValTy()).Elem().Interface()
	}
	return v.Interface()
}

func (d dict) Add(key, val interface{}) {
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

func (d dict) Remove(key interface{}) {
	d.Value.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
}

func (d dict) Len() int {
	return d.Value.Len()
}

func (d dict) Keys() []interface{} {
	keys := d.Value.MapKeys()
	slice.SortValues(keys, d.KeyTy())
	out := make([]interface{}, len(keys))
	for i, k := range keys {
		out[i] = k.Interface()
	}
	return out
}

func (d dict) KeyTy() reflect.Type { return d.Value.Type().Key() }

func (d dict) ValTy() reflect.Type { return d.Value.Type().Elem() }
