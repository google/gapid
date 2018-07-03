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

package stream

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
)

var (
	// Linear is a Sampling state using a linear curve.
	Linear = &Sampling{Curve: Curve_Linear}
	// LinearNormalized is a Sampling state using a normalized, linear curve.
	LinearNormalized = &Sampling{
		Curve:      Curve_Linear,
		Normalized: true,
	}
	// SRGBNormalized is a Sampling state using a normalized, srgb curve
	SRGBNormalized = &Sampling{
		Curve:      Curve_sRGB,
		Normalized: true,
	}
)

// Format prints the Sampling to f.
func (s Sampling) Format(f fmt.State, r rune) {
	parts := []string{}
	if s.Normalized {
		if r == 'c' {
			parts = append(parts, "N")
		} else {
			parts = append(parts, "NORM")
		}
	}
	if s.Premultiplied {
		if r == 'c' {
			parts = append(parts, "P")
		} else {
			parts = append(parts, "PMA")
		}
	}
	if s.Curve != Curve_Linear {
		parts = append(parts, fmt.Sprint(s.Curve))
	}
	fmt.Fprint(f, strings.Join(parts, "_"))
}

// Is returns true if s is equivalent to o.
func (s Sampling) Is(o Sampling) bool {
	return proto.Equal(&s, &o)
}
