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

package git

import "github.com/google/gapid/core/data/id"

// SHA represents a Git changelist SHA.
type SHA [20]byte

func (i SHA) String() string { return id.ID(i).String() }

// IsValid returns true if the identifier is non-zero.
func (i SHA) IsValid() bool { return id.ID(i).IsValid() }

// Parse parses the identifier string.
func (i *SHA) Parse(str string) error { return ((*id.ID)(i)).Parse(str) }

// MarshalJSON implements the json.Marshaler interface.
func (i SHA) MarshalJSON() ([]byte, error) { return id.ID(i).MarshalJSON() }

// UnmarshalJSON implements the json.Unmarshaler interface.
func (i *SHA) UnmarshalJSON(data []byte) error { return ((*id.ID)(i)).UnmarshalJSON(data) }
