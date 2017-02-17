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

package device_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/device"
)

func TestMemoryLayoutString(t *testing.T) {
	ctx := assert.Context(t)
	for _, test := range []struct {
		layout *device.MemoryLayout
		result string
	}{
		{
			device.ARMv7a.MemoryLayout(),
			"PointerAlignment:4 PointerSize:4 IntegerSize:4 SizeSize:4 U64Alignment:8 Endian:LittleEndian ",
		},
		{
			device.X86_64.MemoryLayout(),
			"PointerAlignment:8 PointerSize:8 IntegerSize:8 SizeSize:8 U64Alignment:8 Endian:LittleEndian ",
		},
	} {
		assert.With(ctx).ThatString(test.layout).Equals(test.result)
	}
}
