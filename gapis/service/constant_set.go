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

package service

import (
	"fmt"
	"reflect"
	"strings"
)

// Sprint prints val, attempting to using the constant sets values.
func (e *ConstantSet) Sprint(val interface{}) string {
	v := reflect.ValueOf(val)
	var i uint64
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i = uint64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i = v.Uint()
	default:
		return fmt.Sprint(val)
	}

	if e.IsBitfield {
		bits := []string{}
		for _, c := range e.Constants {
			if i&c.Value != 0 {
				bits = append(bits, c.Name)
				i &^= c.Value
			}
		}
		if i != 0 {
			bits = append(bits, fmt.Sprintf("0x%x", i))
		}
		return strings.Join(bits, " | ")
	}

	for _, c := range e.Constants {
		if i == c.Value {
			return c.Name
		}
	}

	return fmt.Sprint(val)
}
