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

package f64_test

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/math/f64"
)

func TestFromBits(t *testing.T) {
	assert := assert.To(t)
	var checks = map[string]float64{
		"0 00000 000000":     0.0,
		"0 00000 0000000000": 0.0,
		"0 00000 0000000001": 5.960464477539063e-08,
		"0 00000 0000001000": 4.76837158203125e-07,
		"0 00000 000101":     4.76837158203125e-06,
		"0 00000 0001010000": 4.76837158203125e-06,
		"0 00000 0001010011": 4.947185516357422e-06,
		"0 00000 110100":     4.9591064453125e-05,
		"0 00000 1101000000": 4.9591064453125e-05,
		"0 00000 1101000110": 4.9948692321777344e-05,
		"0 00000 111111":     6.008148193359375e-05,
		"0 00000 1111111111": 6.097555160522461e-05,
		"0 00001 000000":     6.103515625e-05,
		"0 00001 0000000000": 6.103515625e-05,
		"0 00001 101000":     9.918212890625e-05,
		"0 00001 1010001101": 9.995698928833008e-05,
		"0 01101 010101":     0.33203125,
		"0 01101 0101010101": 0.333251953125,
		"0 01111 000000":     1.0,
		"0 01111 0000000000": 1.0,
		"0 01111 0000000001": 1.0009765625,
		"0 01111 000001":     1.015625,
		"0 10000 000000":     2.0,
		"0 10000 0000000000": 2.0,
		"0 10000 100000":     3.0,
		"0 10000 1000000000": 3.0,
		"0 10001 000000":     4.0,
		"0 10001 0000000000": 4.0,
		"0 10001 010000":     5.0,
		"0 10001 0100000000": 5.0,
		"0 10110 111110":     252.0,
		"0 10110 111111":     254.0,
		"0 10111 000000":     256.0,
		"0 11110 100001":     49664,
		"0 11110 1000011010": 49984,
		"0 11110 111111":     65024,
		"0 11110 1111111111": 65504,
		"0 11111 000000":     math.Inf(+1),
		"0 11111 0000000000": math.Inf(+1),
		"1 01101 0101010101": -0.333251953125,
		"1 01111 0000000000": -1.0,
		"1 10000 0000000000": -2.0,
		"1 10000 1000000000": -3.0,
		"1 10001 0000000000": -4.0,
		"1 10001 0100000000": -5.0,
		"1 11111 0000000000": math.Inf(-1),
		// Check that 64-bit floats round-trip
		"0 00000000000 0000000000000000000000000000000000000000000000000001": 4.94065645841246544176568792868e-324,
		"0 01111111111 0000000000000000000000000000000000000000000000000000": 1.0,
		"0 11111111111 0000000000000000000000000000000000000000000000000000": math.Inf(+1),
		"1 01111111111 0000000000000000000000000000000000000000000000000000": -1.0,
	}

	for bits, expected := range checks {
		// Parse the string value and check it is in the expected format.
		v, err := strconv.ParseUint(strings.Replace(bits, " ", "", -1), 2, 64)
		assert.For("err").ThatError(err).Succeeded()
		parts := strings.Split(bits, " ")
		assert.For("split").That(len(parts)).Equals(3)
		signBits, expBits, manBits := len(parts[0]), uint32(len(parts[1])), uint32(len(parts[2]))
		assert.For("signBits").That(signBits).Equals(1)

		assert.For("Expand %v", bits).That(f64.FromBits(v, expBits, manBits)).Equals(expected)
	}
}
