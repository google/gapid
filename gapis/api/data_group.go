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
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data/pod"
)

func (list *KeyValuePairList) AppendKeyValuePair(name string, value *DataValue) *KeyValuePairList {
	values := append(list.KeyValues,
		&KeyValuePair{
			Name:  name,
			Value: value,
		})

	return &KeyValuePairList{
		KeyValues: values,
	}
}

func CreateEnumDataValue(typeName string, value fmt.Stringer) *DataValue {
	s := value.String()

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
				Value:       i,
				StringValue: s,
			},
		},
	}
}

func CreatePoDDataValue(typeName string, val interface{}) *DataValue {
	return &DataValue{
		TypeName: typeName,
		Val: &DataValue_Value{
			pod.NewValue(val),
		},
	}
}
