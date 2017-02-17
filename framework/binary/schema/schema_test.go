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

package schema_test

import (
	"testing"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
)

// TestNothing exists because some tools don't like packages that don't have any tests in them.
func TestNothing(t *testing.T) {}

var _ = []binary.Type{
	(*schema.Any)(nil),
	(*schema.Array)(nil),
	(*schema.Interface)(nil),
	(*schema.Map)(nil),
	(*schema.Pointer)(nil),
	(*schema.Primitive)(nil),
	(*schema.Slice)(nil),
	(*schema.Struct)(nil),
	(*schema.Variant)(nil),
}
