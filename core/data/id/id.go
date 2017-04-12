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

package id

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Size is the size of an ID.
const Size = 20

// ID is a codeable unique identifier.
type ID [Size]byte

// IsValid returns true if the id is not the default value.
func (id ID) IsValid() bool {
	return id != ID{}
}

func (id ID) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%x", id[:])
}

func (id ID) String() string {
	return hex.EncodeToString(id[:])
}

// Parse parses lowercase string s as a 20 byte hex-encoded ID, or copies s
// to the ID if it is 20 bytes long.
func (id *ID) Parse(s string) error {
	if len(s) == Size {
		copy((*id)[:], s)
		return nil
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(bytes) != Size {
		return fmt.Errorf("Invalid ID size: got %d, expected %d", len(bytes), Size)
	}
	copy((*id)[:], bytes)
	return nil
}

// Parse parses lowercase string s as a 20 byte hex-encoded ID.
func Parse(s string) (ID, error) {
	id := ID{}
	return id, id.Parse(s)
}

// MarshalJSON encodes the ID as a JSON string.
func (i ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON decodes a JSON string as an ID.
func (i *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return i.Parse(s)
}
