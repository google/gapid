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

package analysis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/gapid/gapil/semantic"
)

// Value interface compliance check.
var _ = Value(&MapValue{})

// MapValue is an implementation of Value that represents a map variable.
// Map entries are stored in the KeyToValue and ValueToKey fields. These two
// maps are always kept in sync.
//
// MapValue attempts to reduce the number of entries stored to a minimum to
// maintain decent performance while also trying to keep enough information to
// provide useful information (avoid tending towards unbounded keys and values).
//
// All insertions and merges use the following method when merging the key value
// pair (k, v) into the map:
//   - If m contains an equivalent key to k, then change the existing value to be
//     a union of the existing value and v.
//   - If m contains an equivalent value to v, then change the existing key to be
//     a union of the existing key and k.
//   - If m does not contain an equivalent key to k or an equivalent value to v
//     then add a new entry to m.
type MapValue struct {
	// The map type.
	Map *semantic.Map

	// A map of keys to values. The inverse of ValueToKey.
	KeyToValue map[Value]Value

	// A map of values to keys. The inverse of KeyToValue.
	ValueToKey map[Value]Value
}

func (v *MapValue) String() string {
	parts := make([]string, 0, len(v.KeyToValue))
	for k, v := range v.KeyToValue {
		parts = append(parts, fmt.Sprintf("<%v: %v>", k, v))
	}
	sort.Strings(parts)
	return fmt.Sprintf("{ %v }", strings.Join(parts, ", "))
}

// Print returns a textual representation of the value.
func (v *MapValue) Print(results *Results) string {
	parts := make([]string, 0, len(v.KeyToValue))
	for k, v := range v.KeyToValue {
		parts = append(parts, fmt.Sprintf("<%v: %v>", k.Print(results), v.Print(results)))
	}
	sort.Strings(parts)
	return fmt.Sprintf("{ %v }", strings.Join(parts, ", "))
}

// Type returns the semantic map type of the value.
func (v *MapValue) Type() semantic.Type {
	return v.Map
}

// Equivalent returns false as maps do not support equivalency tests.
func (v *MapValue) Equivalent(o Value) bool {
	return false // Maps don't do equivalency.
}

// Equals returns False as maps do not support equality tests.
func (v *MapValue) Equals(o Value) Possibility {
	return False // Maps don't do equality.
}

// Valid returns true.
func (v *MapValue) Valid() bool {
	return true
}

// Union (∪) returns a map value with all the keys of v and o mapped to their
// respective values.
// o must be of type *MapValue.
func (v *MapValue) Union(o Value) Value {
	if v == o {
		return v
	}
	out := v.Clone().(*MapValue)
	for key, val := range o.(*MapValue).KeyToValue {
		out.mergeInline(key, val)
	}
	return out
}

// Intersect is not supported by MapValue and will panic if called.
func (v *MapValue) Intersect(o Value) Value {
	panic("Intersect not implemented for maps")
}

// Difference is not supported by MapValue and will panic if called.
func (v *MapValue) Difference(o Value) Value {
	panic("Difference not implemented for maps")
}

// Clone returns a copy of v with a unique pointer.
func (v *MapValue) Clone() Value {
	out := &MapValue{
		Map:        v.Map,
		KeyToValue: make(map[Value]Value, len(v.KeyToValue)),
		ValueToKey: make(map[Value]Value, len(v.KeyToValue)),
	}
	for k, v := range v.KeyToValue {
		out.KeyToValue[k] = v
		out.ValueToKey[v] = k
	}
	return out
}

// ContainsKey returns the possibility of the key existing in the map.
func (v *MapValue) ContainsKey(key Value) Possibility {
	result := False
	for k := range v.KeyToValue {
		switch k.Equals(key) {
		case Maybe:
			result = Maybe
		case True:
			return True
		}
	}
	return result
}

// Put returns a copy of v with the new mapping of key to val.
func (v *MapValue) Put(key, val Value) *MapValue {
	out := v.Clone().(*MapValue)
	if k, v := out.findEqualByKey(key); k != nil {
		delete(out.KeyToValue, k)
		delete(out.ValueToKey, v)
	}
	out.mergeInline(key, val)
	return out
}

// Get return the union of all values that might have a key equal to key.
func (v *MapValue) Get(s *scope, key Value) Value {
	candidates := []Value{}
	for k, v := range v.KeyToValue {
		if k.Equals(key).MaybeTrue() {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		// TODO: Track potential invalid map index.
		return s.defaultOf(v.Map.ValueType)
	}
	return UnionOf(candidates...)
}

// Clear returns a new map that does not contain anything
func (v *MapValue) Clear() Value {
	return &MapValue{
		Map:        v.Map,
		KeyToValue: make(map[Value]Value, len(v.KeyToValue)),
		ValueToKey: make(map[Value]Value, len(v.KeyToValue)),
	}
}

// findEqualByKey looks for an existing key in v that is equal to keyin,
// returning the existing key and value if found. If no key equal to keyin
// is found then (nil, nil) is returned.
func (v *MapValue) findEqualByKey(keyin Value) (key, val Value) {
	if val, ok := v.KeyToValue[keyin]; ok {
		return keyin, val
	}
	for k, v := range v.KeyToValue {
		if k.Equals(keyin) == True {
			return k, v
		}
	}
	return nil, nil
}

// findEquivalentByKey looks for an existing key in v that is equivalent to
// keyin, returning the existing key and value if found. If no key equivalent to
// keyin is found then (nil, nil) is returned.
func (v *MapValue) findEquivalentByKey(keyin Value) (key, val Value) {
	if val, ok := v.KeyToValue[keyin]; ok {
		return keyin, val
	}
	for k, v := range v.KeyToValue {
		if k.Equivalent(keyin) {
			return k, v
		}
	}
	return nil, nil
}

// findEquivalentByVal looks for an existing value in v that is equivalent to
// valin, returning the existing key and value if found. If no value equivalent
// to valin is found then (nil, nil) is returned.
func (v *MapValue) findEquivalentByVal(valin Value) (key, val Value) {
	if key, ok := v.ValueToKey[valin]; ok {
		return key, valin
	}
	for v, k := range v.ValueToKey {
		if v.Equivalent(valin) {
			return k, v
		}
	}
	return nil, nil
}

// mergeInline merges the key-val pair into the map v using the method described
// in the MapValue documentation.
func (v *MapValue) mergeInline(key, val Value) {
	if !key.Valid() {
		panic(fmt.Errorf("Attempting to put invalid key %T%v into map", key, key))
	}
	if existingKey, existingVal := v.findEquivalentByKey(key); existingVal != nil {
		delete(v.KeyToValue, existingKey)
		delete(v.ValueToKey, existingVal)
		val = val.Union(existingVal)
	}
	if existingKey, existingVal := v.findEquivalentByVal(val); existingKey != nil {
		delete(v.KeyToValue, existingKey)
		delete(v.ValueToKey, existingVal)
		key = key.Union(existingKey)
	}
	v.KeyToValue[key] = val
	v.ValueToKey[val] = key
}
