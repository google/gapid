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

package assert_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/gapid/core/assert"
)

type testEnum int

type enumEntry struct {
	value testEnum
	name  string
}

var (
	enums = []enumEntry{
		{0, "UNKNOWN"},
		{1, "A"},
		{2, "B"},
		{3, "BadParse"},
		{4, "FailedParse"},
		{5, "BadJsonMarshal"},
		{6, "FailedJsonMarshal"},
		{7, "BadJsonUnmarshal"},
		{8, "FailedJsonUnmarshal"},
	}
	enumTests = append(enums, []enumEntry{
		{testEnum(-1), "testEnum(-1)"},
		{1, "B"},
	}...)
)

func (v testEnum) String() string {
	for _, e := range enums {
		if e.value == v {
			return e.name
		}
	}
	return fmt.Sprintf("testEnum(%d)", v)
}

func (v *testEnum) Parse(s string) error {
	if s == "BadParse" {
		*v = 0
		return nil
	}
	if s == "FailedParse" {
		s = "badparse"
	}
	for _, e := range enums {
		if e.name == s {
			*v = e.value
			return nil
		}
	}
	trimmed := strings.TrimSuffix(strings.TrimPrefix(s, "testEnum("), ")")
	if i, err := strconv.Atoi(trimmed); err == nil {
		*v = testEnum(i)
		return nil
	}
	return fmt.Errorf("%s not in testEnum", s)
}

func (v testEnum) MarshalJSON() ([]byte, error) {
	if v.String() == "FailedJsonMarshal" {
		return nil, fmt.Errorf("FailedJsonMarshal")
	}
	if v.String() == "BadJsonMarshal" {
		return json.Marshal("badjson")
	}
	return json.Marshal(v.String())
}

func (v *testEnum) UnmarshalJSON(bytes []byte) error {
	var str string
	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}
	if str == "FailedJsonUnmarshal" {
		return fmt.Errorf("FailedJsonUnmarshal")
	}
	if str == "BadJsonUnmarshal" {
		*v = 0
		return nil
	}
	return v.Parse(str)
}

// An example of testing enum values
func ExampleEnum() {
	assert := assert.To(nil)
	for _, test := range enumTests {
		assert.For(test.name).ThatEnum(&test.value).HasName(test.name)
	}
	var enum testEnum
	assert.For(`"A"`).ThatEnum(&enum).CannotUnmarshal(`"A"`)
	assert.For("0").ThatEnum(&enum).CannotUnmarshal(`0`)

	// Output:
	// Error:BadParse
	//     For enum   BadParse
	//     Bad Parse  UNKNOWN
	// Error:FailedParse
	//     For enum                FailedParse
	//     Unexpected parse error  `badparse not in testEnum`
	// Error:BadJsonMarshal
	//     For enum  BadJsonMarshal
	//     Bad JSON  "badjson"
	//     Expect    "BadJsonMarshal"
	// Error:FailedJsonMarshal
	//     For enum            FailedJsonMarshal
	//     JSON marshal error  `json: error calling MarshalJSON for type *assert_test.testEnum: FailedJsonMarshal`
	// Error:BadJsonUnmarshal
	//     For enum            BadJsonUnmarshal
	//     Bad JSON Unmarshal  UNKNOWN
	// Error:FailedJsonUnmarshal
	//     For enum              FailedJsonUnmarshal
	//     JSON unmarshal error  `FailedJsonUnmarshal`
	// Error:B
	//     For enum       A
	//     Expected name  `B`
	// Error:"A"
	//     For     "A"
	//     Got     A
	//     Expect  Unmarshal to fail

}
