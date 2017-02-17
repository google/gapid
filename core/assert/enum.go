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

package assert

import (
	"encoding/json"
	"reflect"
)

// Enum is the interface for something that acts like an enumeration value.
type Enum interface {
	// String matches the Stringer interface, and returns the name of the value.
	String() string
	// Parse takes a name and sets the value to match the name.
	Parse(value string) error
}

// OnEnum is the result of calling ThatEnum on an Assertion.
// It provides enumeration assertion tests.
type OnEnum struct {
	Assertion
	enum Enum
}

// ThatEnum returns an OnEnum for assertions on enumeration objects.
func (a Assertion) ThatEnum(enum Enum) OnEnum {
	return OnEnum{Assertion: a, enum: enum}
}

// HasName asserts that the enum value has the supplied name.
// It verifies both the String/Parse and JSON marshalling.
func (o OnEnum) HasName(name string) bool {
	return o.Add("For enum", o.enum).Test(func() bool {
		// Check String method
		s := o.enum.String()
		if s != name {
			o.Add("Expected name", name)
			return false
		}
		// Check Parse method
		copy := reflect.New(reflect.TypeOf(o.enum).Elem()).Interface().(Enum)
		err := copy.Parse(name)
		if err != nil {
			o.Add("Unexpected parse error", err)
			return false
		}
		if !reflect.DeepEqual(copy, o.enum) {
			o.Add("Bad Parse", copy)
			return false
		}
		// Check json marshalling
		data, err := json.Marshal(o.enum)
		if err != nil {
			o.Add("JSON marshal error", err)
			return false
		}
		quoted := `"` + name + `"`
		if string(data) != quoted {
			o.Rawln("Bad JSON", "", string(data))
			o.ExpectRaw("", quoted)
			return false
		}
		// Check json un-marshalling
		err = json.Unmarshal([]byte(quoted), copy)
		if err != nil {
			o.Add("JSON unmarshal error", err)
			return false
		}
		if !reflect.DeepEqual(copy, o.enum) {
			o.Add("Bad JSON Unmarshal", copy)
			return false
		}
		return true
	}())
}

// CannotUnmarshal asserts that the enum type cannot unmarshal the supplied string without error.
func (o OnEnum) CannotUnmarshal(data string) bool {
	err := json.Unmarshal([]byte(data), o.enum)
	return o.Rawln("For", "", data).CompareRaw(o.enum, "", "Unmarshal to fail").Test(err != nil)
}
