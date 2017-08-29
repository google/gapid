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

	"github.com/golang/protobuf/proto"
)

// Format prints the Component to f.
func (c Component) Format(f fmt.State, r rune) {
	fmt.Fprint(f, c.Channel)
	fmt.Fprint(f, "_")
	fmt.Fprint(f, c.DataType)
	if c.Sampling != nil {
		if s := fmt.Sprint(c.Sampling); len(s) > 0 {
			fmt.Fprint(f, "_", s)
		}
	}
}

// IsNormalized returns true if the component should be normalized when read.
func (c *Component) IsNormalized() bool {
	if c.Sampling != nil {
		return c.Sampling.Normalized
	}
	return false
}

// Clone returns a deep copy of c.
func (c *Component) Clone() *Component {
	return proto.Clone(c).(*Component)
}
