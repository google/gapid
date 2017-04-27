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
	"bytes"
	"fmt"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/replay/value"
)

type constantEncoder struct {
	writer      binary.Writer
	buffer      *bytes.Buffer
	constantMap map[id.ID]uint64
	data        []byte
	alignment   uint64
}

func newConstantEncoder(memoryLayout *device.MemoryLayout) *constantEncoder {
	buffer := &bytes.Buffer{}
	writer := endian.Writer(buffer, memoryLayout.GetEndian())
	return &constantEncoder{
		writer:      writer,
		buffer:      buffer,
		constantMap: make(map[id.ID]uint64),
		alignment:   uint64(memoryLayout.GetPointer().GetAlignment()),
	}
}

func (e *constantEncoder) writeValues(v ...value.Value) value.Pointer {
	if len(v) == 0 {
		panic("Cannot write an empty list of values!")
	}

	e.begin()
	for _, v := range v {
		switch v := v.(type) {
		case value.Bool:
			e.writer.Bool(bool(v))
		case value.U8:
			e.writer.Uint8(uint8(v))
		case value.S8:
			e.writer.Int8(int8(v))
		case value.U16:
			e.writer.Uint16(uint16(v))
		case value.S16:
			e.writer.Int16(int16(v))
		case value.F32:
			e.writer.Float32(float32(v))
		case value.U32:
			e.writer.Uint32(uint32(v))
		case value.S32:
			e.writer.Int32(int32(v))
		case value.F64:
			e.writer.Float64(float64(v))
		case value.U64:
			e.writer.Uint64(uint64(v))
		case value.S64:
			e.writer.Int64(int64(v))
		default:
			panic(fmt.Errorf("Cannot write Value %T to constant memory", v))
		}
	}
	return e.finish()
}

func (e *constantEncoder) writeString(s string) value.Pointer {
	e.begin()
	e.writer.String(s)
	return e.finish()
}

func (e *constantEncoder) begin() {
	e.buffer.Reset()
}

func (e *constantEncoder) finish() value.Pointer {
	data := e.buffer.Bytes()
	hash := id.OfBytes(data)
	offset, found := e.constantMap[hash]
	if !found {
		offset = uint64(len(e.data))
		if offset%e.alignment != 0 {
			padding := e.alignment - offset%e.alignment
			e.data = append(e.data, make([]byte, padding)...)
			offset += padding
		}
		e.data = append(e.data, data...)
		e.constantMap[hash] = offset
	}
	return value.ConstantPointer(offset)
}
