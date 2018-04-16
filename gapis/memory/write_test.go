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
	"bytes"
	"testing"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

func TestWriteOn32bitArch(t *testing.T) {
	arch := device.Big32
	values := []interface{}{
		uint8(0x12), int8(0x12),
		uint16(0x1234), int16(0x1234),
		uint32(0x12345678), int32(0x12345678),
		BytePtr(0x87654321),
		[]uint8{0x10, 0x20, 0x30},
		[]uint16{0x10, 0x20, 0x30},
		[]uint32{0x10, 0x20, 0x30},
		"hello",
	}

	buf := &bytes.Buffer{}
	e := NewEncoder(endian.Writer(buf, arch.GetEndian()), arch)
	Write(e, values)

	got := buf.Bytes()
	expected := []byte{
		0x12, 0x12, // uint8(0x12), int8(0x12),
		0x12, 0x34, 0x12, 0x34, // uint16(0x1234), int16(0x1234),
		0x00, 0x00, // padding
		0x12, 0x34, 0x56, 0x78, // uint32(0x12345678),
		0x12, 0x34, 0x56, 0x78, // int32(0x12345678),
		0x87, 0x65, 0x43, 0x21, // Pointer(0x87654321),
		0x10, 0x20, 0x30, // []uint8{0x10, 0x20, 0x30},
		0x00,                               // padding
		0x00, 0x10, 0x00, 0x20, 0x00, 0x30, // []uint16{0x10, 0x20, 0x30},
		0x00, 0x00, // padding
		0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x30, // []uint32{0x10, 0x20, 0x30},
		'h', 'e', 'l', 'l', 'o', 0, // "hello"
	}

	if !bytes.Equal(got, expected) {
		t.Errorf("Bytes were not as expected.\nExpected: % x\nGot:      % x", expected, got)
	}
}

type encodableStruct struct {
	X uint8
	Y Pointer
	Z int16
	W uint64
}

var _ Encodable = encodableStruct{}

func (s encodableStruct) Encode(e *Encoder) {
	e.U8(s.X)
	e.Pointer(s.Y.Address())
	e.I16(s.Z)
	e.U64(s.W)
}

func TestWriteStructOn32bitArch(t *testing.T) {
	arch := device.Big32
	values := encodableStruct{0x12, BytePtr(0xdeadbeef), 0x3456, 0x8888888899999999}

	buf := &bytes.Buffer{}
	e := NewEncoder(endian.Writer(buf, arch.GetEndian()), arch)
	Write(e, values)

	got := buf.Bytes()
	expected := []byte{
		0x12,             // uint8
		0x00, 0x00, 0x00, // padding before pointer
		0xde, 0xad, 0xbe, 0xef, // pointer
		0x34, 0x56, // int16
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // padding before uint64
		0x88, 0x88, 0x88, 0x88,
		0x99, 0x99, 0x99, 0x99,
	}

	if !bytes.Equal(got, expected) {
		t.Errorf("Bytes were not as expected.\nExpected: % x\nGot:      % x", expected, got)
	}
}

func TestWriteOn64bitArch(t *testing.T) {
	arch := device.Big64
	values := []interface{}{
		uint8(0x12), int8(0x12),
		uint16(0x1234), int16(0x1234),
		uint32(0x12345678), int32(0x12345678),
		BytePtr(0x87654321),
		[]uint8{0x10, 0x20, 0x30},
		BytePtr(0x1234567890abcdef),
		"hello",
		uint64(0xfedcba0987654321), // Mock size_t type
	}

	buf := &bytes.Buffer{}
	e := NewEncoder(endian.Writer(buf, arch.GetEndian()), arch)
	Write(e, values)

	got := buf.Bytes()
	expected := []byte{
		0x12, 0x12, // uint8(0x12), int8(0x12),
		0x12, 0x34, 0x12, 0x34, // uint16(0x1234), int16(0x1234),
		0x00, 0x00, // padding
		0x12, 0x34, 0x56, 0x78, // uint32(0x12345678),
		0x12, 0x34, 0x56, 0x78, // int32(0x12345678),
		0x00, 0x00, 0x00, 0x00, 0x87, 0x65, 0x43, 0x21, // Pointer(0x87654321),
		0x10, 0x20, 0x30, // []uint8{0x10, 0x20, 0x30},
		0x00, 0x00, 0x00, 0x00, 0x00, // padding
		0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, // Pointer(0x1234567890abcdef),
		'h', 'e', 'l', 'l', 'o', 0, // "hello"
		0x00, 0x00, // padding
		0xfe, 0xdc, 0xba, 0x09, 0x87, 0x65, 0x43, 0x21, // uint64(0xfedbca0987654321),
	}

	if !bytes.Equal(got, expected) {
		t.Errorf("Bytes were not as expected.\nExpected: % x\nGot:      % x", expected, got)
	}
}

func TestWriteStructOn64bitArch(t *testing.T) {
	arch := device.Big64
	values := encodableStruct{0x12, BytePtr(0xbeefdeaddeadbeef), 0x3456, 0x8888888899999999}

	buf := &bytes.Buffer{}
	e := NewEncoder(endian.Writer(buf, arch.GetEndian()), arch)
	Write(e, values)

	got := buf.Bytes()
	expected := []byte{
		0x12,                                     // uint8
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // padding before pointer
		0xbe, 0xef, 0xde, 0xad, 0xde, 0xad, 0xbe, 0xef, // pointer
		0x34, 0x56, // int16
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // padding before uint64
		0x88, 0x88, 0x88, 0x88,
		0x99, 0x99, 0x99, 0x99,
	}

	if !bytes.Equal(got, expected) {
		t.Errorf("Bytes were not as expected.\nExpected: % x\nGot:      % x", expected, got)
	}
}
