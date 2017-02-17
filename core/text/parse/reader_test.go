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

package parse

import (
	"testing"

	"github.com/google/gapid/core/assert"
)

var numericTests = []struct {
	in       string
	outToken string
	outKind  NumberKind
}{
	{"47", "47", Decimal},
	{"47u", "47u", Decimal},
	{"0x47u", "0x47u", Hexadecimal},
	{".47", ".47", Floating},
	{"42.47", "42.47", Floating},
	{"47.", "47.", Floating},
	{"47.f", "47.f", Floating},
	{"47.F", "47.F", Floating},
	{"0.47", "0.47", Floating},
	{"0.47f", "0.47f", Floating},
	{"0.47F", "0.47F", Floating},
	{"0", "0", Octal},
	{"047", "047", Octal},
	{"047u", "047u", Octal},
	{"47.e42", "47.e42", Scientific},
	{"4.7e4", "4.7e4", Scientific},
	{"4.7e-4", "4.7e-4", Scientific},
	{".", "", NotNumeric},
	{"47e", "", NotNumeric},
	{"47e+", "", NotNumeric},
}

func TestNumeric(t *testing.T) {
	assert := assert.To(t)
	for _, test := range numericTests {
		r := NewReader("reader_test.api", test.in)
		assert.For("kind").Add("in", test.in).That(r.Numeric()).Equals(test.outKind)
		assert.For("token").Add("in", test.in).That(r.Token().String()).Equals(test.outToken)
	}
}
