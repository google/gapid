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

package memory

import (
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/os/device"
)

// Encoder provides methods to write primitives to a binary.Writer, respecting
// a given MemoryLayout.
// Encoder will automatically handle alignment and types sizes.
type Encoder struct {
	w binary.Writer
	m *device.MemoryLayout
	o uint64
}

// NewEncoder constructs and returns a new Encoder that writes to w using
// the memory layout m.
func NewEncoder(w binary.Writer, m *device.MemoryLayout) *Encoder { return &Encoder{w, m, 0} }

func (e *Encoder) alignAndOffset(d *device.DataTypeLayout) {
	e.Align(uint64(d.Alignment))
	e.o += uint64(d.Size)
}

// MemoryLayout returns the MemoryLayout used by the encoder.
func (e *Encoder) MemoryLayout() *device.MemoryLayout {
	return e.m
}

// Offset returns the byte offset of the writer from the initial Encoder
// creation.
func (e *Encoder) Offset() uint64 {
	return e.o
}

// Align writes zero bytes until the write position is a multiple of to.
func (e *Encoder) Align(to uint64) {
	alignment := u64.AlignUp(e.o, to)
	if pad := alignment - e.o; pad != 0 {
		e.Pad(pad)
	}
}

// Pad writes n zero bytes to the writer.
func (e *Encoder) Pad(n uint64) {
	binary.WriteBytes(e.w, 0, int32(n))
	e.o += n
}

// Pointer stores a pointer address.
func (e *Encoder) Pointer(addr uint64) {
	e.alignAndOffset(e.m.Pointer)
	binary.WriteUint(e.w, 8*e.m.Pointer.Size, addr)
}

// F32 stores a float32.
func (e *Encoder) F32(v float32) {
	e.alignAndOffset(e.m.F32)
	e.w.Float32(v)
}

// F64 stores a float64.
func (e *Encoder) F64(v float64) {
	e.alignAndOffset(e.m.F64)
	e.w.Float64(v)
}

// I8 stores a int8.
func (e *Encoder) I8(v int8) {
	e.alignAndOffset(e.m.I8)
	e.w.Int8(v)
}

// I16 stores a int16.
func (e *Encoder) I16(v int16) {
	e.alignAndOffset(e.m.I16)
	e.w.Int16(v)
}

// I32 stores a int32.
func (e *Encoder) I32(v int32) {
	e.alignAndOffset(e.m.I32)
	e.w.Int32(v)
}

// I64 stores a int64.
func (e *Encoder) I64(v int64) {
	e.alignAndOffset(e.m.I64)
	e.w.Int64(v)
}

// U8 stores a uint8.
func (e *Encoder) U8(v uint8) {
	e.alignAndOffset(e.m.I8)
	e.w.Uint8(v)
}

// U16 stores a uint16.
func (e *Encoder) U16(v uint16) {
	e.alignAndOffset(e.m.I16)
	e.w.Uint16(v)
}

// U32 stores a uint32.
func (e *Encoder) U32(v uint32) {
	e.alignAndOffset(e.m.I32)
	e.w.Uint32(v)
}

// U64 stores a uint64.
func (e *Encoder) U64(v uint64) {
	e.alignAndOffset(e.m.I64)
	e.w.Uint64(v)
}

// Char stores an char.
func (e *Encoder) Char(v Char) {
	e.alignAndOffset(e.m.Char)
	binary.WriteInt(e.w, 8*e.m.Char.Size, int64(v))
}

// Int stores an int.
func (e *Encoder) Int(v Int) {
	e.alignAndOffset(e.m.Integer)
	binary.WriteInt(e.w, 8*e.m.Integer.Size, int64(v))
}

// Uint stores a uint.
func (e *Encoder) Uint(v Uint) {
	e.alignAndOffset(e.m.Integer)
	binary.WriteUint(e.w, 8*e.m.Integer.Size, uint64(v))
}

// Size stores a size_t.
func (e *Encoder) Size(v Size) {
	e.alignAndOffset(e.m.Size)
	binary.WriteUint(e.w, 8*e.m.Size.Size, uint64(v))
}

// String stores a null-terminated string.
func (e *Encoder) String(v string) {
	e.w.String(v)
	e.o += uint64(len(v) + 1)
}

// Bool stores a boolean value.
func (e *Encoder) Bool(v bool) {
	if v {
		e.w.Uint8(1)
	} else {
		e.w.Uint8(0)
	}
	e.o++
}

// Data writes raw bytes from buf.
func (e *Encoder) Data(buf []byte) {
	e.w.Data(buf)
	e.o += uint64(len(buf))
}

// Error returns the error state of the underlying writer.
func (e *Encoder) Error() error {
	return e.w.Error()
}
