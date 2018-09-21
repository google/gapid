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

package box_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service/box"
)

type StructB struct {
	Pointer *int32
}

type StructA struct {
	Bool   bool
	Int    int
	Float  float32
	Map    map[string]int
	Slice  []byte
	Struct StructB
	Any    interface{}
}

type Cyclic struct {
	I int
	S *Cyclic
}

type Memory struct {
	P memory.Pointer
	S memory.Slice
}

type StringːString struct {
	M map[string]string
}

// Get returns the value of the entry with the given key.
func (m StringːString) Get(k string) string { return m.M[k] }

// Put inserts the key-value pair, replacing any existing entry with the same
// key.
func (m StringːString) Put(k, v string) { m.M[k] = v }

// Lookup searches for the value of the entry with the given key.
func (m StringːString) Lookup(k string) (val string, ok bool) { val, ok = m.M[k]; return }

// Contains returns true if the dictionary contains an entry with the given
// key.
func (m StringːString) Contains(k string) bool { _, ok := m.M[k]; return ok }

// Remove removes the entry with the given key. If no entry with the given
// key exists then this call is a no-op.
func (m StringːString) Remove(k string) { delete(m.M, k) }

// Len returns the number of entries in the dictionary.
func (m StringːString) Len() int { return len(m.M) }

// Keys returns all the entry keys in the map.
func (m StringːString) Keys() []string {
	out := make([]string, 0, len(m.M))
	for k := range m.M {
		out = append(out, k)
	}
	return out
}

var _ data.Assignable = &StringːString{}

func (m *StringːString) Assign(v interface{}) bool {
	m.M = map[string]string{}
	return deep.Copy(&m.M, v) == nil
}

type StringːDictionary struct {
	M map[string]StringːString
}

// Get returns the value of the entry with the given key.
func (m StringːDictionary) Get(k string) StringːString { return m.M[k] }

// Put inserts the key-value pair, replacing any existing entry with the same
// key.
func (m StringːDictionary) Put(k string, v StringːString) { m.M[k] = v }

// Lookup searches for the value of the entry with the given key.
func (m StringːDictionary) Lookup(k string) (val StringːString, ok bool) { val, ok = m.M[k]; return }

// Contains returns true if the dictionary contains an entry with the given
// key.
func (m StringːDictionary) Contains(k string) bool { _, ok := m.M[k]; return ok }

// Remove removes the entry with the given key. If no entry with the given
// key exists then this call is a no-op.
func (m StringːDictionary) Remove(k string) { delete(m.M, k) }

// Len returns the number of entries in the dictionary.
func (m StringːDictionary) Len() int { return len(m.M) }

// Keys returns all the entry keys in the map.
func (m StringːDictionary) Keys() []string {
	out := make([]string, 0, len(m.M))
	for k := range m.M {
		out = append(out, k)
	}
	return out
}

var _ data.Assignable = &StringːDictionary{}

func (m *StringːDictionary) Assign(v interface{}) bool {
	m.M = map[string]StringːString{}
	return deep.Copy(&m.M, v) == nil
}

type DictInContainer struct {
	Struct StringːString
	Slice  []StringːString
	Map    map[string]StringːString
	Dict   StringːDictionary
}

func TestBoxSimpleTypes(t *testing.T) {
	ctx := log.Testing(t)
	ty := box.NewType(reflect.TypeOf(StructA{}))
	got, _ := json.MarshalIndent(ty, "", "  ")
	expect := `{
  "type_id": 1,
  "Ty": {
    "Struct": {
      "fields": [
        {
          "type": {
            "Ty": {
              "Pod": 13
            }
          },
          "name": "Bool"
        },
        {
          "type": {
            "Ty": {
              "Pod": 4
            }
          },
          "name": "Int"
        },
        {
          "type": {
            "Ty": {
              "Pod": 1
            }
          },
          "name": "Float"
        },
        {
          "type": {
            "type_id": 2,
            "Ty": {
              "Map": {
                "key_type": {
                  "Ty": {
                    "Pod": 14
                  }
                },
                "value_type": {
                  "Ty": {
                    "Pod": 4
                  }
                }
              }
            }
          },
          "name": "Map"
        },
        {
          "type": {
            "Ty": {
              "Pod": 19
            }
          },
          "name": "Slice"
        },
        {
          "type": {
            "type_id": 3,
            "Ty": {
              "Struct": {
                "fields": [
                  {
                    "type": {
                      "type_id": 4,
                      "Ty": {
                        "Reference": {
                          "Ty": {
                            "Pod": 10
                          }
                        }
                      }
                    },
                    "name": "Pointer"
                  }
                ]
              }
            }
          },
          "name": "Struct"
        },
        {
          "type": {
            "Ty": {
              "Any": true
            }
          },
          "name": "Any"
        }
      ]
    }
  }
}`
	assert.For(ctx, "NewType(structA)").ThatString(got).Equals(expect)
}

func TestBoxUnboxStructB(t *testing.T) {
	ctx := log.Testing(t)
	ten := int32(10)
	val := StructB{Pointer: &ten}
	boxed := box.NewValue(val)
	unboxed := boxed.Get()
	var got StructB
	reflect.ValueOf(&got).Elem().Set(reflect.ValueOf(unboxed))
	assert.For(ctx, "unboxed").That(got).DeepEquals(val)
}

func TestBoxUnboxStructA(t *testing.T) {
	ctx := log.Testing(t)
	ten := int32(10)
	val := StructA{
		Bool:  true,
		Int:   123,
		Float: 1.23,
		Map: map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		},
		Slice:  []byte{1, 2, 3},
		Struct: StructB{Pointer: &ten},
	}
	boxed := box.NewValue(val)
	var unboxed StructA
	err := boxed.AssignTo(&unboxed)
	if assert.For(ctx, "AssignTo").ThatError(err).Succeeded() {
		assert.For(ctx, "unboxed").That(unboxed).DeepEquals(val)
	}
}

func TestBoxCyclicType(t *testing.T) {
	ctx := log.Testing(t)
	ty := box.NewType(reflect.TypeOf(Cyclic{}))
	got, _ := json.MarshalIndent(ty, "", "  ")
	expect := `{
  "type_id": 1,
  "Ty": {
    "Struct": {
      "fields": [
        {
          "type": {
            "Ty": {
              "Pod": 4
            }
          },
          "name": "I"
        },
        {
          "type": {
            "type_id": 2,
            "Ty": {
              "Reference": {
                "type_id": 1,
                "Ty": {
                  "BackReference": true
                }
              }
            }
          },
          "name": "S"
        }
      ]
    }
  }
}`
	assert.For(ctx, "NewType(Struct)").ThatString(got).Equals(expect)
}

func TestBoxUnboxCyclic(t *testing.T) {
	ctx := log.Testing(t)
	val := Cyclic{
		I: 10,
		S: &Cyclic{
			I: 20,
			S: &Cyclic{
				I: 30,
			},
		},
	}
	boxed := box.NewValue(val)
	var unboxed Cyclic
	err := boxed.AssignTo(&unboxed)
	if assert.For(ctx, "AssignTo").ThatError(err).Succeeded() {
		assert.For(ctx, "unboxed").That(unboxed).DeepEquals(val)
	}
}

func TestBoxMemoryType(t *testing.T) {
	ctx := log.Testing(t)
	ty := box.NewType(reflect.TypeOf(Memory{}))
	got, _ := json.MarshalIndent(ty, "", "  ")
	expect := `{
  "type_id": 1,
  "Ty": {
    "Struct": {
      "fields": [
        {
          "type": {
            "Ty": {
              "Pointer": true
            }
          },
          "name": "P"
        },
        {
          "type": {
            "Ty": {
              "Slice": true
            }
          },
          "name": "S"
        }
      ]
    }
  }
}`
	assert.For(ctx, "NewType(Struct)").ThatString(got).Equals(expect)
}

func TestBoxUnboxMemory(t *testing.T) {
	ctx := log.Testing(t)
	val := Memory{
		P: memory.BytePtr(1234),
		S: memory.NewSlice(1234, 1256, 42, 42, 333, reflect.TypeOf(byte(0))),
	}
	boxed := box.NewValue(val)
	var unboxed Memory
	err := boxed.AssignTo(&unboxed)
	if assert.For(ctx, "AssignTo").ThatError(err).Succeeded() {
		assert.For(ctx, "unboxed").That(unboxed).DeepEquals(val)
	}
}

func TestBoxUnboxDictionaryInContainer(t *testing.T) {
	ctx := log.Testing(t)
	val := DictInContainer{
		Struct: StringːString{map[string]string{"cat": "meow", "dog": "woof"}},
		Slice: []StringːString{
			StringːString{map[string]string{"bird": "tweet", "cow": "mooh"}},
			StringːString{map[string]string{"mouse": "squeek", "sheep": "baah"}},
		},
		Map: map[string]StringːString{
			"savanna": StringːString{map[string]string{"lion": "roar", "wildebeest": "grunt"}},
			"prairie": StringːString{map[string]string{"coyote": "howl", "snake": "hiss"}},
		},
		Dict: StringːDictionary{
			map[string]StringːString{
				"grassland": StringːString{map[string]string{"hyena": "hehe"}},
				"forest":    StringːString{map[string]string{"me": "sigh"}},
			},
		},
	}
	boxed := box.NewValue(val)
	var unboxed DictInContainer
	err := boxed.AssignTo(&unboxed)
	if assert.For(ctx, "AssignTo").ThatError(err).Succeeded() {
		assert.For(ctx, "unboxed").That(unboxed).DeepEquals(val)
	}
}
