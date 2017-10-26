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

package stream

import (
	"fmt"
	"math"
	"reflect"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/math/f16"
	"github.com/google/gapid/core/math/f32"
	"github.com/google/gapid/core/math/f64"
)

type buf struct {
	bytes     []byte
	component *Component
	offset    uint32 // in bits
	stride    uint32 // in bits
}

func (b buf) clone() buf {
	b.component = b.component.Clone()
	return b
}

type mapping struct {
	dst, src buf
}

// Convert converts count elements from data in format src to dst.
// Components are matched based on channel and semantic information.
// Components found in src that are not in dst are ignored.
// Certain components found in dst that are not in src are filled with default
// values (Y=0, Z=0, W=1, Alpha=1).
// Component order and datatypes can be changed.
func Convert(dst, src *Format, data []byte) ([]byte, error) {
	if dst == src || reflect.DeepEqual(dst, src) {
		return data, nil
	}

	dstStride, srcStride := dst.Stride(), src.Stride()
	count := len(data) / srcStride
	out := make([]byte, dst.Size(count))
	mappings := make([]mapping, len(dst.Components))
	srcOffsets := src.BitOffsets()
	dstOffset := uint32(0)

	// Fill in the mappings slice with direct component matches.
	for i, d := range dst.Components {
		m := &mappings[i]
		m.dst = buf{out, d, dstOffset, uint32(dstStride) * 8}

		s, err := src.Component(d.Channel)
		if err != nil {
			return nil, err
		}
		if s != nil {
			m.src = buf{data, s, srcOffsets[s], uint32(srcStride) * 8}
		}
		dstOffset += d.DataType.Bits()

		if err := m.convertCurve(count); err != nil {
			return nil, err
		}
	}

	if src.Channels().Contains(Channel_SharedExponent) && !dst.Channels().Contains(Channel_SharedExponent) {
		return convertSharedExponent(dst, src, data)
	}

	// Some components can be implicitly added (alpha, Y, Z, W).
	mappings = resolveImplicitMappings(count, mappings, src, data)

	// Do the conversion work.
	for _, m := range mappings {
		if m.src.component == nil {
			return nil, fmt.Errorf("Channel %v not found in source format: %v",
				m.dst.component.Channel, src)
		}
		if err := m.conv(count); err != nil {
			return nil, err
		}

	}
	return out, nil
}

func resolveImplicitMappings(count int, mappings []mapping, srcFmt *Format, srcData []byte) []mapping {
	for i := range mappings {
		m := &mappings[i]
		if m.src.component != nil {
			continue
		}
		srcChannels := srcFmt.Channels()
		switch m.dst.component.Channel {
		case Channel_Alpha:
			if srcChannels.ContainsColor() || srcChannels.ContainsDepth() {
				m.src = buf1Norm
			}
		case Channel_W:
			if srcChannels.ContainsVector() {
				m.src = buf1Norm
			}
		case Channel_Y, Channel_Z:
			if srcChannels.ContainsVector() {
				m.src = buf0
			}
		case Channel_Red, Channel_Green, Channel_Blue:
			if c, _ := srcFmt.Component(Channel_Gray, Channel_Luminance); c != nil {
				// Missing red, green or blue but have a gray or luminance channel. Use that.
				m.src = buf{srcData, c, srcFmt.BitOffsets()[c], uint32(srcFmt.Stride()) * 8}
			} else if c, _ := srcFmt.Component(Channel_Depth); c != nil {
				// Convert depth to RGB.
				m.src = buf{srcData, c, srcFmt.BitOffsets()[c], uint32(srcFmt.Stride()) * 8}
			} else if srcChannels.ContainsColor() {
				m.src = buf0
			}
		case Channel_Luminance:
			if c := srcFmt.GetSingleComponent(func(c *Component) bool { return c.Channel.IsColor() }); c != nil {
				// A format with a single color channel is equivalent to a luninance format.
				m.src = buf{srcData, c, srcFmt.BitOffsets()[c], uint32(srcFmt.Stride()) * 8}
			} else if c, _ := srcFmt.Component(Channel_Depth); c != nil {
				// Convert depth to luminance.
				m.src = buf{srcData, c, srcFmt.BitOffsets()[c], uint32(srcFmt.Stride()) * 8}
			}
			// TODO: RGB->Luminance conversion (#276)
		}
	}
	return mappings
}

var (
	buf0     = buf{[]byte{0}, &Component{DataType: &U1, Sampling: Linear}, 0, 0}
	buf1Norm = buf{[]byte{1}, &Component{DataType: &U1, Sampling: LinearNormalized}, 0, 0}
)

func (m *mapping) conv(count int) error {
	d, s := m.dst.component, m.src.component
	if d.GetSampling().GetCurve() != s.GetSampling().GetCurve() {
		return fmt.Errorf("Cannot convert curve from %v to %v", s.GetSampling().GetCurve(), d.GetSampling().GetCurve())
	}
	dstIsInt, srcIsInt := d.DataType.IsInteger(), s.DataType.IsInteger()
	dstIsFloat, srcIsFloat := d.DataType.IsFloat(), s.DataType.IsFloat()
	switch {
	case *d.DataType == *s.DataType:
		return clone(count, m.dst, m.src)
	case dstIsFloat && srcIsFloat:
		return ftof(count, m.dst, m.src)
	case dstIsInt && srcIsInt:
		if !s.IsNormalized() {
			return intCast(count, m.dst, m.src)
		}
		// Source is normalized
		if d.DataType.Signed == s.DataType.Signed {
			if d.DataType.GetInteger().Bits > s.DataType.GetInteger().Bits {
				return intExpand(count, m.dst, m.src)
			}
			return intCollapse(count, m.dst, m.src)
		}
	case dstIsFloat && srcIsInt:
		if s.DataType.Signed {
			return stof(count, m.dst, m.src)
		}
		return utof(count, m.dst, m.src)
	case dstIsInt && srcIsFloat:
		if d.DataType.Signed {
			return ftos(count, m.dst, m.src)
		}
		return ftou(count, m.dst, m.src)
	}
	return fmt.Errorf("Cannot convert from %v to %v", s, d)
}

// straight up copy.
func clone(count int, dst, src buf) error {
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	bits := dst.component.DataType.Bits()
	for i := 0; i < count; i++ {
		ds.Write(ss.Read(bits), bits)
		ds.WritePos += dst.stride - bits
		ss.ReadPos += src.stride - bits
	}
	return nil
}

// integer reinterpret cast with sign extension
func intCast(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	dstBitsIncSign, srcBitsIncSign := dstTy.Bits(), srcTy.Bits()
	srcBitsExcSign := srcTy.GetInteger().Bits
	signed := dstTy.Signed
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	srcSignEx := ^(uint64(1<<srcBitsExcSign) - 1)
	for i := 0; i < count; i++ {
		v := ss.Read(srcBitsExcSign)
		if signed && ss.Read(1) == 1 {
			v |= srcSignEx
		}
		ds.Write(v, dstBitsIncSign)
		ds.WritePos += dst.stride - dstBitsIncSign
		ss.ReadPos += src.stride - srcBitsIncSign
	}
	return nil
}

var uintExpandPatterns = []uint64{
	0x0000000000000000,
	0xffffffffffffffff, // 1111111111111111...
	0x5555555555555555, // 0101010101010101...
	0x2492492492492492, // 0010010010010010...
	0x1111111111111111, // 0001000100010001...
	0x0842108421084210, // 0000100000001000...
	0x0410410410410410, // 0000010000010000...
	0x0204081020408102, // 0000001000000100...
	0x0101010101010101, // 0000000100000001...
}

// int to larger bit precision (bit repeating)
func intExpand(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	if dstTy.Signed != srcTy.Signed {
		return fmt.Errorf("Cannot perform signed conversion")
	}
	dstBitsIncSign, srcBitsIncSign := dstTy.Bits(), srcTy.Bits()
	dstBitsExcSign, srcBitsExcSign := dstTy.GetInteger().Bits, srcTy.GetInteger().Bits
	toU64 := uintExpandPatterns[srcBitsExcSign] // index out of range? Add more patterns!
	shift := 64 - dstBitsIncSign
	signed := dstTy.Signed
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	for i := 0; i < count; i++ {
		v := uint64(ss.Read(srcBitsExcSign))
		v = (v * toU64) >> shift
		ds.Write(uint64(v), dstBitsExcSign)
		if signed {
			ds.Write(ss.Read(1), 1) // Copy sign
		}
		ds.WritePos += dst.stride - dstBitsIncSign
		ss.ReadPos += src.stride - srcBitsIncSign
	}
	return nil
}

// int to smaller bit precision (drops LSBs)
func intCollapse(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	if dstTy.Signed != srcTy.Signed {
		return fmt.Errorf("intCollapse cannot perform signed conversion")
	}
	dstBitsIncSign, srcBitsIncSign := dstTy.Bits(), srcTy.Bits()
	dstBitsExcSign, srcBitsExcSign := dstTy.GetInteger().Bits, srcTy.GetInteger().Bits
	shift := srcBitsIncSign - dstBitsIncSign
	signed := dstTy.Signed
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	for i := 0; i < count; i++ {
		v := ss.Read(srcBitsExcSign) >> shift
		ds.Write(uint64(v), dstBitsExcSign)
		if signed {
			ds.Write(ss.Read(1), 1) // Copy sign
		}
		ds.WritePos += dst.stride - dstBitsIncSign
		ss.ReadPos += src.stride - srcBitsIncSign
	}
	return nil
}

// unsigned int to float
func utof(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	dstIsF16, dstIsF32, dstIsF64 := dstTy.Is(F16), dstTy.Is(F32), dstTy.Is(F64)
	if !(dstIsF16 || dstIsF32 || dstIsF64) {
		return fmt.Errorf("Cannot convert to %v", dstTy)
	}
	norm := src.component.IsNormalized()
	dstBitsIncSign := dstTy.Bits()
	srcBits := srcTy.Bits()
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	scale := 1.0 / float64((uint(1)<<srcBits)-1)
	for i := 0; i < count; i++ {
		f := float64(ss.Read(srcBits))
		if norm {
			f *= scale
		}
		switch {
		case dstIsF16:
			ds.Write(uint64(f16.From(float32(f))), 16)
		case dstIsF32:
			ds.Write(uint64(math.Float32bits(float32(f))), 32)
		case dstIsF64:
			ds.Write(math.Float64bits(f), 64)
		}
		ds.WritePos += dst.stride - dstBitsIncSign
		ss.ReadPos += src.stride - srcBits
	}

	return nil
}

// signed int to float
func stof(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	dstIsF16, dstIsF32, dstIsF64 := dstTy.Is(F16), dstTy.Is(F32), dstTy.Is(F64)
	if !(dstIsF16 || dstIsF32 || dstIsF64) {
		return fmt.Errorf("Cannot convert to %v", dstTy)
	}
	dstBitsIncSign := dstTy.Bits()
	srcBitsIncSign, srcBitsExcSign := srcTy.Bits(), srcTy.GetInteger().Bits
	norm := src.component.IsNormalized()
	srcSignEx := ^(uint64(1<<srcBitsExcSign) - 1)
	mid, max := float64(uint(1)<<srcBitsExcSign), float64((uint(1)<<srcBitsIncSign)-1)
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	for i := 0; i < count; i++ {
		u := ss.Read(srcBitsExcSign)
		if ss.Read(1) == 1 {
			u |= srcSignEx
		}
		f := float64(int32(u))
		if norm {
			f = 2 * ((f+mid)/max - 0.5)
		}
		switch {
		case dstIsF16:
			ds.Write(uint64(f16.From(float32(f))), 16)
		case dstIsF32:
			ds.Write(uint64(math.Float32bits(float32(f))), 32)
		case dstIsF64:
			ds.Write(math.Float64bits(f), 64)
		}
		ds.WritePos += dst.stride - dstBitsIncSign
		ss.ReadPos += src.stride - srcBitsIncSign
	}

	return nil
}

// float to unsigned int
func ftou(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	srcIsF16, srcIsF32, srcIsF64 := srcTy.Is(F16), srcTy.Is(F32), srcTy.Is(F64)
	srcExpBits, srcManBits := srcTy.GetFloat().ExponentBits, srcTy.GetFloat().MantissaBits
	norm := dst.component.IsNormalized()
	dstBits, srcBits := dstTy.Bits(), srcTy.Bits()
	dstMask := (1 << dstBits) - 1
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	switch {
	case srcIsF16:
		scale := float32(dstMask)
		for i := 0; i < count; i++ {
			f := f16.Number(ss.Read(16)).Float32()
			if norm {
				f *= scale
			}
			ds.Write(uint64(f), dstBits)
			ds.WritePos += dst.stride - dstBits
			ss.ReadPos += src.stride - 16
		}
	case srcIsF32:
		scale := float32(dstMask)
		for i := 0; i < count; i++ {
			f := math.Float32frombits(uint32(ss.Read(32)))
			if norm {
				f *= scale
			}
			ds.Write(uint64(f), dstBits)
			ds.WritePos += dst.stride - dstBits
			ss.ReadPos += src.stride - 32
		}
	case srcIsF64:
		scale := float64(dstMask)
		for i := 0; i < count; i++ {
			f := math.Float64frombits(ss.Read(64))
			if norm {
				f *= scale
			}
			ds.Write(uint64(f), dstBits)
			ds.WritePos += dst.stride - dstBits
			ss.ReadPos += src.stride - 64
		}
	default:
		scale := float64(dstMask)
		for i := 0; i < count; i++ {
			f := float64(f64.FromBits(ss.Read(srcBits), srcExpBits, srcManBits))
			if norm {
				f *= scale
			}
			ds.Write(uint64(f), dstBits)
			ds.WritePos += dst.stride - dstBits
			ss.ReadPos += src.stride - srcBits
		}
	}
	return nil
}

// float to signed integer
func ftos(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	srcIsF16, srcIsF32, srcIsF64 := srcTy.Is(F16), srcTy.Is(F32), srcTy.Is(F64)
	srcExpBits, srcManBits := srcTy.GetFloat().ExponentBits, srcTy.GetFloat().MantissaBits
	dstBitsIncSign, srcBits := dstTy.Bits(), srcTy.Bits()
	norm := dst.component.IsNormalized()
	mul := (1 << dstBitsIncSign) - 1
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}
	switch {
	case srcIsF16:
		for i := 0; i < count; i++ {
			f := f16.Number(ss.Read(16)).Float32()
			if norm {
				f = (f * float32(mul) / 2) - 0.5
			}
			ds.Write(uint64(f32.Round(f)), dstBitsIncSign)
			ds.WritePos += dst.stride - dstBitsIncSign
			ss.ReadPos += src.stride - 16
		}
	case srcIsF32:
		for i := 0; i < count; i++ {
			f := math.Float32frombits(uint32(ss.Read(32)))
			if norm {
				f = (f * float32(mul) / 2) - 0.5
			}
			ds.Write(uint64(f32.Round(f)), dstBitsIncSign)
			ds.WritePos += dst.stride - dstBitsIncSign
			ss.ReadPos += src.stride - 32
		}
	case srcIsF64:
		for i := 0; i < count; i++ {
			f := math.Float64frombits(ss.Read(64))
			if norm {
				f = (f * float64(mul) / 2) - 0.5
			}
			ds.Write(uint64(f64.Round(f)), dstBitsIncSign)
			ds.WritePos += dst.stride - dstBitsIncSign
			ss.ReadPos += src.stride - 64
		}
	default:
		for i := 0; i < count; i++ {
			f := float64(f64.FromBits(ss.Read(srcBits), srcExpBits, srcManBits))
			if norm {
				f = (f * float64(mul) / 2) - 0.5
			}
			ds.Write(uint64(f64.Round(f)), dstBitsIncSign)
			ds.WritePos += dst.stride - dstBitsIncSign
			ss.ReadPos += src.stride - srcBits
		}
	}
	return nil
}

// float to float
func ftof(count int, dst, src buf) error {
	dstTy, srcTy := dst.component.DataType, src.component.DataType
	dstBits, srcBits := dstTy.Bits(), srcTy.Bits()
	srcIsF16, srcIsF32, srcIsF64 := srcTy.Is(F16), srcTy.Is(F32), srcTy.Is(F64)
	srcExpBits, srcManBits := srcTy.GetFloat().ExponentBits, srcTy.GetFloat().MantissaBits
	dstIsF16, dstIsF32, dstIsF64 := dstTy.Is(F16), dstTy.Is(F32), dstTy.Is(F64)
	if !(dstIsF16 || dstIsF32 || dstIsF64) {
		return fmt.Errorf("Cannot convert to %v", dstTy)
	}
	ss := binary.BitStream{Data: src.bytes, ReadPos: src.offset}
	ds := binary.BitStream{Data: dst.bytes, WritePos: dst.offset}

	for i := 0; i < count; i++ {
		var f float64
		switch {
		case srcIsF16:
			f = float64(f16.Number(ss.Read(16)).Float32())
		case srcIsF32:
			f = float64(math.Float32frombits(uint32(ss.Read(32))))
		case srcIsF64:
			f = float64(math.Float64frombits(ss.Read(64)))
		default:
			f = float64(f64.FromBits(ss.Read(srcBits), srcExpBits, srcManBits))
		}
		switch {
		case dstIsF16:
			ds.Write(uint64(f16.From(float32(f))), 16)
		case dstIsF32:
			ds.Write(uint64(math.Float32bits(float32(f))), 32)
		case dstIsF64:
			ds.Write(math.Float64bits(f), 64)
		}
		ds.WritePos += dst.stride - dstBits
		ss.ReadPos += src.stride - srcBits

	}
	return nil
}
