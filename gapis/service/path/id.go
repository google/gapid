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

package path

import (
	"encoding/hex"
	"fmt"

	"github.com/google/gapid/core/data/id"
)

// ID returns the identifier as a id.ID.
func (i *ID) ID() id.ID {
	var out id.ID
	if i != nil {
		copy(out[:], i.Data)
	}
	return out
}

// NewID returns a new ID from an id.ID.
func NewID(i id.ID) *ID {
	data := make([]byte, 20)
	copy(data, i[:])
	return &ID{Data: data}
}

// IsValid returns true if the ID is valid.
func (i *ID) IsValid() bool {
	return i != nil && i.ID().IsValid()
}

// Format implements fmt.Formatter to print id to f.
func (i *ID) Format(f fmt.State, r rune) {
	f.Write([]byte(hex.EncodeToString(i.Data)))
}
