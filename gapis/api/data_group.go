// Copyright (C) 2019 Google Inc.
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
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/gapis/service/path"
)

func (list *KeyValuePairList) AppendKeyValuePair(name string, value *DataValue, dynamic bool) *KeyValuePairList {
	values := append(list.KeyValues,
		&KeyValuePair{
			Name:    name,
			Value:   value,
			Dynamic: dynamic,
			Active:  true,
		})

	return &KeyValuePairList{
		KeyValues: values,
	}
}

func (list *KeyValuePairList) AppendDependentKeyValuePair(name string, value *DataValue, dynamic bool, dependee string, active bool) *KeyValuePairList {
	values := append(list.KeyValues,
		&KeyValuePair{
			Name:     name,
			Value:    value,
			Dynamic:  dynamic,
			Dependee: dependee,
			Active:   active,
		})

	return &KeyValuePairList{
		KeyValues: values,
	}
}

func CreateEnumDataValue(typeName string, value fmt.Stringer) *DataValue {
	s := truncateEnumString(typeName, value.String())

	var i uint64 = 0
	v := reflect.ValueOf(value)
	switch v.Type().Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i = v.Uint()

	default:
		s = "INVALID ENUM"
	}

	return &DataValue{
		TypeName: typeName,
		Val: &DataValue_EnumVal{
			&EnumValue{
				Value:        i,
				StringValue:  value.String(),
				DisplayValue: s,
			},
		},
	}
}

func CreatePoDDataValue(ctx context.Context, s *GlobalState, typeName string, val interface{}) *DataValue {
	dv := &DataValue{
		TypeName: typeName,
	}

	if handle, ok := val.(Handle); ok {
		dv.Val = &DataValue_HandleVal{&HandleValue{Value: handle.Handle()}}
	} else {
		dv.Val = &DataValue_Value{pod.NewValue(val)}
	}

	if labeled, ok := val.(Labeled); ok {
		dv.Label = labeled.Label(ctx, s)
	}

	return dv
}

func CreateBitfieldDataValue(typeName string, val interface{}, index int32, a API) *DataValue {
	v := reflect.ValueOf(val)
	var n uint64
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n = uint64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n = v.Uint()
	default:
		return &DataValue{
			TypeName: typeName,
			Val: &DataValue_Bitfield{
				&BitfieldValue{
					SetBits:         []uint64{0},
					SetBitnames:     []string{"INVALID BITFIELD"},
					SetDisplayNames: []string{"INVALID BITFIELD"},
				},
			},
		}
	}

	cs := a.ConstantSets()
	set := cs.Sets[index]

	truncatedTypeName := strings.ToUpper(typeName)
	truncatedTypeName = strings.TrimSuffix(truncatedTypeName, "FLAGBITS")
	truncatedTypeName = strings.TrimSuffix(truncatedTypeName, "FLAGS")

	bits := []uint64{}
	names := []string{}
	displayNames := []string{}
	if set.IsBitfield {
		for _, e := range set.Entries {
			if n == 0 && e.V == 0 {
				bits = append(bits, 0)
				names = append(names, cs.Symbols.Get(e))
				displayNames = append(displayNames, truncateEnumString(typeName, cs.Symbols.Get(e)))
				break
			} else if n&e.V != 0 {
				bits = append(bits, e.V)
				names = append(names, cs.Symbols.Get(e))
				displayNames = append(displayNames, truncateEnumString(typeName, cs.Symbols.Get(e)))
				n &^= e.V
			}
		}

		if n != 0 {
			bits = append(bits, n)
			names = append(names, fmt.Sprintf("%s (%d)", typeName, n))
			displayNames = append(displayNames, fmt.Sprintf("%s (%d)", typeName, n))
		}
	} else {
		bits = append(bits, 0)
		names = append(names, "INVALID BITFIELD")
		displayNames = append(displayNames, "INVALID BITFIELD")
	}

	return &DataValue{
		TypeName: typeName,
		Val: &DataValue_Bitfield{
			&BitfieldValue{
				SetBits:         bits,
				SetBitnames:     names,
				SetDisplayNames: displayNames,
				Combined:        typeName == "VkColorComponentFlagBits",
			},
		},
	}
}

func CreateLinkedDataValue(typeName string, p []*path.Any, val *DataValue) *DataValue {
	return &DataValue{
		TypeName: typeName,
		Val: &DataValue_Link{
			&LinkedValue{
				Link:       p,
				DisplayVal: val,
			},
		},
	}
}

func truncateEnumString(typeName string, s string) string {
	typeName = strings.ToUpper(typeName)
	typeName = strings.TrimSuffix(typeName, "FLAGBITS")
	typeName = strings.TrimSuffix(typeName, "FLAGS")

	for i, j := 0, 0; i < len(s); i++ {
		if s[i] == typeName[j] {
			j++
		}

		if j == len(typeName) {
			s = s[i+2:]
			break
		}
	}

	s = strings.TrimSuffix(s, "_BIT")

	return s
}
