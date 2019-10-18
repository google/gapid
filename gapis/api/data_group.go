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

func CreateEnumDataValue(value uint64, stringValue string) *DataValue {
	return &DataValue{
		TypeName: "Enum",
		Val: &DataValue_EnumVal{
			&EnumValue{
				Value:       value,
				StringValue: stringValue,
			},
		},
	}
}

func CreatePoDDataValue(val interface{}) *DataValue {
	return &DataValue{
		TypeName: "PoD",
		Val: &DataValue_Value{
			pod.NewValue(val),
		},
	}
}
