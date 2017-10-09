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

package binary_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/binary"
)

var testByteSequence = []byte{
	//        LSB        MSB
	0x00, // (0) 00000000
	0x01, // (1) 10000000
	0x02, // (2) 01000000
	0x10, // (3) 00001000
	0x80, // (4) 00000001
	0x3A, // (5) 01011100
	0x02, // (6) 01000000
	0x3C, // (7) 00111100
}

var testBitSequence = []struct {
	bits       uint64
	count, pos uint32
}{
	// (0) 0₀0₁0 0 0 0 0 0
	{bits: 0, count: 2, pos: 2},
	// (0) 0 0 0₀0₁0₂0 0 0
	{bits: 0, count: 3, pos: 5},
	// (0) 0 0 0 0 0 0₀0₁0₂
	{bits: 0, count: 3, pos: 8},
	// (1) 1₀0₁0₂0 0 0 0 0
	{bits: 1, count: 3, pos: 11},
	// (1) 1 0 0 0₀0₁0₂0₃0₄
	// (2) 0₅1₆0₇0₈0₉0 0 0
	{bits: 64, count: 10, pos: 21},
	// (2) 0 1 0 0 0 0₀0₁0₂
	// (3) 0₃0₄0₅0₆1₇0₈0₉0
	{bits: 128, count: 10, pos: 31},
	// (3) 0 0 0 1 0 0 0 0₀
	// (4) 0₁0₂0₃0₄0₅0₆0₇1₈
	// (5) 0₉1ₐ0 1 1 1 0 0
	{bits: 1280, count: 11, pos: 42},
	// (5) 0 1 0₀1₁1₂1₃0₄0₅
	// (6) 0₆1₇0₈0₉0ₐ0 0 0
	{bits: 142, count: 11, pos: 53},
	// (6) 0 1 0 0 0 0₀0₁0₂
	// (7) 0₃0₄1 1 1 1 0 0
	{bits: 0, count: 5, pos: 58},
	// (7) 0 0 1₀1₁1₂1 0 0
	{bits: 7, count: 3, pos: 61},
	// (7) 0 0 1 1 1 1₀0₁0₂
	{bits: 1, count: 3, pos: 64},
}

func TestBitStreamReadBit(t *testing.T) {
	assert := assert.To(t)
	data := []byte{
		//    LSB        MSB
		0x00, // 00000000
		0x01, // 10000000
		0x02, // 01000000
		0x10, // 00001000
		0x80, // 00000001
	}
	bits := []uint64{
		0, 0, 0, 0, 0, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0,
		0, 1, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 1,
	}
	bs := binary.BitStream{Data: data}
	for i, expected := range bits {
		assert.For("bit %d", i).That(bs.ReadBit()).Equals(expected)
	}
}

func TestBitStreamRead(t *testing.T) {
	assert := assert.To(t)
	bs := binary.BitStream{Data: testByteSequence}
	for i, c := range testBitSequence {
		assert.For("bits %d", i).That(bs.Read(c.count)).Equals(c.bits)
		assert.For("pos %d", i).That(bs.ReadPos).Equals(c.pos)
	}
}

func TestBitStreamZeroRemainder(t *testing.T) {
	assert := assert.To(t)
	data := []byte{
		//        LSB        MSB
		0x00, // (0) 00000000
		0x01, // (1) 10000000
		0x02, // (2) 01000000
		0x10, // (3) 00001000
	}
	bs := binary.BitStream{Data: data}
	got, expected := bs.Read(32), uint64(0x10020100)
	assert.For("data").That(got).Equals(expected)
}

func TestBitStreamWriteBit(t *testing.T) {
	assert := assert.To(t)
	expected := []byte{
		//    LSB        MSB
		0x00, // 00000000
		0x01, // 10000000
		0x02, // 01000000
		0x10, // 00001000
		0x80, // 00000001
	}
	bits := []uint64{
		0, 0, 0, 0, 0, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0,
		0, 1, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 1,
	}
	for _, test := range []struct {
		name        string
		initialData []byte
	}{
		{"empty", nil},
		{"primed", []byte{0x55, 0x55, 0x55, 0x55, 0x55}},
	} {
		bs := binary.BitStream{Data: test.initialData}
		for _, b := range bits {
			bs.WriteBit(b)
		}
		assert.For(test.name).ThatSlice(bs.Data).Equals(expected)
	}
}

func TestBitStreamWrite(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		name        string
		initialData []byte
	}{
		{"empty", nil},
		{"primed", []byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55}},
	} {
		bs := binary.BitStream{Data: test.initialData}
		for i, c := range testBitSequence {
			bs.Write(uint64(c.bits), c.count)
			assert.For("%s WritePos %d", test.name, i).That(bs.WritePos).Equals(c.pos)
		}
		assert.For(test.name).ThatSlice(bs.Data).Equals(testByteSequence)
	}
}

func BenchmarkBitstreamRead(b *testing.B) {
	data := make([]byte, b.N*32)
	bs := binary.BitStream{Data: data}
	for i := 0; i < b.N; i++ {
		bs.Read(uint32(i & 31))
	}
}

func BenchmarkBitstreamWrite(b *testing.B) {
	data := make([]byte, b.N*32)
	bs := binary.BitStream{Data: data}
	for i := 0; i < b.N; i++ {
		bs.Write(uint64(i), uint32(i&31))
	}
}
