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

package parse_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/parse"
)

var numericTests = []struct {
	in       string
	outToken string
	outKind  parse.NumberKind
}{
	{"47", "47", parse.Decimal},
	{"47u", "47u", parse.Decimal},
	{"0x47u", "0x47u", parse.Hexadecimal},
	{".47", ".47", parse.Floating},
	{"42.47", "42.47", parse.Floating},
	{"47.", "47.", parse.Floating},
	{"47.f", "47.f", parse.Floating},
	{"47.F", "47.F", parse.Floating},
	{"0.47", "0.47", parse.Floating},
	{"0.47f", "0.47f", parse.Floating},
	{"0.47F", "0.47F", parse.Floating},
	{"0", "0", parse.Octal},
	{"047", "047", parse.Octal},
	{"047u", "047u", parse.Octal},
	{"47.e42", "47.e42", parse.Scientific},
	{"4.7e4", "4.7e4", parse.Scientific},
	{"4.7e-4", "4.7e-4", parse.Scientific},
	{".", "", parse.NotNumeric},
	{"47e", "", parse.NotNumeric},
	{"47e+", "", parse.NotNumeric},
}

func TestNumeric(t *testing.T) {
	assert := assert.To(t)
	for _, test := range numericTests {
		r := parse.NewReader("reader_test.api", test.in)
		assert.For("kind").Add("in", test.in).That(r.Numeric()).Equals(test.outKind)
		assert.For("token").Add("in", test.in).That(r.Token().String()).Equals(test.outToken)
	}
}
