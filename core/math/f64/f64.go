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

package f64

import "math"

// MinOf returns the minimum value of all the arguments.
func MinOf(a float64, b ...float64) float64 {
	v := a
	for _, x := range b {
		if x < v {
			v = x
		}
	}
	return v
}

// MaxOf returns the maximum value of all the arguments.
func MaxOf(a float64, b ...float64) float64 {
	v := a
	for _, x := range b {
		if x > v {
			v = x
		}
	}
	return v
}

// Round rounds v to the nearest integer.
// Examples:
//
//	Round(-0.9) = -1
//	Round(-0.1) = 0
//	Round(0.0) = 0
//	Round(0.1) = 0
//	Round(0.9) = 1
func Round(v float64) int {
	if v < 0 {
		return int(math.Ceil(v - 0.5))
	}
	return int(math.Floor(v + 0.5))
}

// FromBits takes binary representation of floating-point value
// with user-defined bit sizes and expands it to full float64.
func FromBits(val uint64, expBits, manBits uint32) float64 {
	manMask := (uint64(1) << manBits) - 1
	expMask := (uint64(1) << expBits) - 1
	expBias := expMask / 2
	const expMaskFloat64 = (1 << 11) - 1
	const expBiasFloat64 = expMaskFloat64 / 2 // 1023

	// Extract mantissa, exponent and sign from the packed value
	man, val := val&manMask, val>>manBits
	exp, sig := val&expMask, val>>expBits

	// Special cases in increasing numerical order of the value
	if exp == 0 {
		if man == 0 {
			// Zero - return zero with the same sign
			return math.Float64frombits(sig << 63)
		} else if expMask != expMaskFloat64 {
			// Denormalized number - promote it to normalized
			exp++
			for man&(1<<manBits) == 0 {
				man *= 2
				exp--
			}
			man &= manMask
			exp += expBiasFloat64 - expBias
		}
	} else if exp < expMask {
		// Normalized number - just adjust the exponent's bias
		exp += expBiasFloat64 - expBias
	} else /* exp == expMask */ {
		// NaN or Inf - set all bits in exponent; keep mantissa
		exp = expMaskFloat64
	}

	// Compact mantissa, exponent and sign to 64-bit float value
	val = (sig << 63) | (exp << 52) | (man << (52 - manBits))
	return math.Float64frombits(val)
}
