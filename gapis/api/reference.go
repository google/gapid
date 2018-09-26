// Copyright (C) 2018 Google Inc.
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

package api

import (
	"sync/atomic"

	"github.com/google/gapid/core/data/compare"
)

// RefID is a type used to identify instances of the reference types used in the API models.
type RefID uint64

// NilRefID identifies a nil reference in the API models.
const NilRefID = RefID(0)

// Reference is an interface which exposes a unique identifier.
// Reference types in the API models should implement this interface.
type Reference interface {
	RefID() RefID
}

var lastRefID uint64

// NewRefID creates a new RefID.
// This RefID is unique within the process.
func NewRefID() RefID {
	return RefID(atomic.AddUint64(&lastRefID, 1))
}

// NilReference is a type representing a nil reference where an implementation
// of the `Reference` interface is expected.
type NilReference struct{}

func (NilReference) RefID() RefID {
	return NilRefID
}

func init() {
	// Don't compare RefIDs.
	compare.Register(func(c compare.Comparator, a, b RefID) {
	})
}
