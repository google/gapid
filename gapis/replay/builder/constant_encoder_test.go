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

package builder

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/replay/value"
)

func TestConstantEncoderCache(t *testing.T) {
	ctx := assert.Context(t)
	c := newConstantEncoder(device.Little32)

	addr1 := c.writeValues(value.U32(0x1234), value.S16(-1))
	addr2 := c.writeValues(value.U32(0x1234), value.S16(-1))
	assert.With(ctx).That(addr1).Equals(addr2)
}

func TestConstantEncoderAlignment(t *testing.T) {
	ctx := assert.Context(t)
	c := newConstantEncoder(&device.MemoryLayout{
		PointerAlignment: 8,
		PointerSize:      4,
		IntegerSize:      4,
		SizeSize:         4,
		U64Alignment:     8,
		Endian:           device.LittleEndian,
	})

	c.writeValues(value.U32(0x1234))
	c.writeValues(value.S16(-1))

	expected := []byte{
		0x34, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xff, 0xff,
	}

	assert.With(ctx).That(c.data).DeepEquals(expected)
}
