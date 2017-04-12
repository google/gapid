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

func TestBoxSimpleTypes(t *testing.T) {
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
              "Basic": 13
            }
          },
          "name": "Bool"
        },
        {
          "type": {
            "Ty": {
              "Basic": 4
            }
          },
          "name": "Int"
        },
        {
          "type": {
            "Ty": {
              "Basic": 1
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
                    "Basic": 14
                  }
                },
                "value_type": {
                  "Ty": {
                    "Basic": 4
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
              "Basic": 19
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
                        "Pointer": {
                          "Ty": {
                            "Basic": 10
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
	assert.To(t).For("NewType(structA)").ThatString(got).Equals(expect)
}

func TestBoxUnboxStructB(t *testing.T) {
	ten := int32(10)
	val := StructB{Pointer: &ten}
	boxed := box.NewValue(val)
	unboxed := boxed.Get()
	var got StructB
	reflect.ValueOf(&got).Elem().Set(reflect.ValueOf(unboxed))
	assert.To(t).For("unboxed").That(got).DeepEquals(val)
}

func TestBoxUnboxStructA(t *testing.T) {
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
	if assert.To(t).For("AssignTo").ThatError(err).Succeeded() {
		assert.To(t).For("unboxed").That(unboxed).DeepEquals(val)
	}
}

func TestBoxCyclicType(t *testing.T) {
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
              "Basic": 4
            }
          },
          "name": "I"
        },
        {
          "type": {
            "type_id": 2,
            "Ty": {
              "Pointer": {
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
	assert.To(t).For("NewType(Struct)").ThatString(got).Equals(expect)
}

func TestBoxUnboxCyclic(t *testing.T) {
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
	if assert.To(t).For("AssignTo").ThatError(err).Succeeded() {
		assert.To(t).For("unboxed").That(unboxed).DeepEquals(val)
	}
}
