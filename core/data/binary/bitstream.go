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

package binary

import "github.com/google/gapid/core/math/u32"

// BitStream provides methods for reading and writing bits to a slice of bytes.
// Bits are packed in a least-significant-bit to most-significant-bit order.
type BitStream struct {
	Data     []byte // The byte slice containing the bits
	ReadPos  uint32 // The current read offset from the start of the Data slice (in bits)
	WritePos uint32 // The current write offset from the start of the Data slice (in bits)
}

// ReadBit reads a single bit from the BitStream, incrementing ReadPos by one.
func (s *BitStream) ReadBit() uint64 {
	ReadPos := s.ReadPos
	s.ReadPos = ReadPos + 1
	return (uint64(s.Data[ReadPos/8]) >> (ReadPos % 8)) & 1
}

// WriteBit writes a single bit to the BitStream, incrementing WritePos by one.
func (s *BitStream) WriteBit(bit uint64) {
	b := s.WritePos / 8
	if b == uint32(len(s.Data)) {
		s.Data = append(s.Data, 0)
	}
	if bit&1 == 1 {
		s.Data[b] |= byte(1 << (s.WritePos % 8))
	} else {
		s.Data[b] &= ^byte(1 << (s.WritePos % 8))
	}
	s.WritePos++
}

// CanRead returns true if there's enough data to call Read(count).
func (s *BitStream) CanRead(count uint32) bool {
	return int(s.ReadPos+count) <= len(s.Data)*8
}

// Read reads the specified number of bits from the BitStream, increamenting the ReadPos by the
// specified number of bits and returning the bits packed into a uint64. The bits are packed into
// the uint64 from LSB to MSB.
func (s *BitStream) Read(count uint32) uint64 {
	if count == 0 {
		return 0
	}

	byteIdx := s.ReadPos / 8
	bitIdx := s.ReadPos & 7

	// Start
	val := uint64(s.Data[byteIdx]) >> bitIdx
	readCount := 8 - bitIdx
	if count <= readCount {
		s.ReadPos += count
		return val & ((1 << count) - 1)
	}
	s.ReadPos += readCount
	byteIdx++
	bitIdx = 0

	// Whole bytes
	for ; readCount+7 < count; readCount += 8 {
		val |= uint64(s.Data[byteIdx]) << readCount
		byteIdx++
		s.ReadPos += 8
	}

	// Remainder
	rem := count - readCount
	if rem > 0 {
		val |= (uint64(s.Data[byteIdx]) & ((1 << rem) - 1)) << readCount
		s.ReadPos += rem
	}
	return val
}

// Write writes the specified number of bits from the packed uint64, increamenting the WritePos by
// the specified number of bits. The bits are read from the uint64 from LSB to MSB.
func (s *BitStream) Write(bits uint64, count uint32) {
	// Ensure the buffer is big enough for all them bits.
	if reqBytes := (int(s.WritePos) + int(count) + 7) / 8; reqBytes > len(s.Data) {
		if reqBytes <= cap(s.Data) {
			s.Data = s.Data[:reqBytes]
		} else {
			buf := make([]byte, reqBytes, reqBytes*2)
			copy(buf, s.Data)
			s.Data = buf
		}
	}

	byteIdx := s.WritePos / 8
	bitIdx := s.WritePos & 7

	// Start
	if bitIdx != 0 {
		writeCount := u32.Min(8-bitIdx, count)
		mask := byte(((1 << writeCount) - 1) << bitIdx)
		s.Data[byteIdx] = (s.Data[byteIdx] & ^mask) | (byte(bits<<bitIdx) & mask)
		s.WritePos += writeCount
		count, byteIdx, bitIdx, bits = count-writeCount, byteIdx+1, 0, bits>>writeCount
	}

	// Whole bytes
	for count >= 8 {
		s.Data[byteIdx] = uint8(bits)
		s.WritePos += 8
		count, byteIdx, bits = count-8, byteIdx+1, bits>>8
	}

	// Remainder
	if count > 0 {
		mask := byte(((1 << count) - 1) << bitIdx)
		s.Data[byteIdx] = (s.Data[byteIdx] & ^mask) | (byte(bits<<bitIdx) & mask)
		s.WritePos += count
	}
}
