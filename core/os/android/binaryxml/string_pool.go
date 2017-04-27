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

package binaryxml

import (
	"bytes"
	"fmt"
	"unicode/utf16"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

type stringPoolRef struct {
	sp  *stringPool
	idx uint32
}

const missingString = 0xffffffff

var invalidStringPoolRef stringPoolRef = stringPoolRef{nil, missingString}

func (r stringPoolRef) isValid() bool {
	return r.idx != missingString && r.sp != nil
}

func (r stringPoolRef) encode(w binary.Writer) {
	w.Uint32(r.stringPoolIndex())
}

func (r stringPoolRef) stringPoolIndex() uint32 {
	if !r.isValid() {
		return missingString
	} else {
		return uint32(r.sp.ptrs[r.idx])
	}
}

func (r stringPoolRef) get() string {
	if r.isValid() && int(r.idx) < len(r.sp.ptrs) {
		return r.sp.strings[r.sp.ptrs[r.idx]]
	}
	return fmt.Sprintf("Resource<0x%x>", r)
}

// See:
// https://android.googlesource.com/platform/frameworks/base/+/master/tools/aapt2/StringPool.cpp
type stringPool struct {
	rootHolder
	strings []string
	styles  []string
	flags   uint32
	ptrs    []int // ptrs maps indices in stringPoolRefs to indices in the raw strings array.
}

func (c *stringPool) decode(header, data []byte) error {
	const sortedFlag = 1 << 0
	const utf8Flag = 1 << 8

	// dataOffset is the offset of data relative to the start of the chunk.
	dataOffset := 8 + uint32(len(header))

	r := endian.Reader(bytes.NewReader(header), device.LittleEndian)
	stringCount := r.Uint32()
	styleCount := r.Uint32()
	c.flags = r.Uint32()
	stringsStart := r.Uint32() - dataOffset
	stylesStart := r.Uint32() - dataOffset

	r = endian.Reader(bytes.NewReader(data), device.LittleEndian)
	indices := make([]uint32, stringCount)
	for i := range indices {
		indices[i] = r.Uint32()
	}

	c.ptrs = make([]int, stringCount)
	c.strings = make([]string, stringCount)
	c.styles = make([]string, styleCount)
	for i := range c.strings {
		offset := stringsStart + indices[i]
		r = endian.Reader(bytes.NewReader(data[offset:]), device.LittleEndian)
		if c.flags&utf8Flag != 0 {
			panic("TODO: UTF8 encoded xml support.")
		} else {
			runeCount := decodeLength(r)
			str := make([]uint16, runeCount)
			for i := range str {
				str[i] = r.Uint16()
			}
			c.strings[i] = string(utf16.Decode(str))
			c.ptrs[i] = i
		}
	}
	if styleCount > 0 {
		_ = stylesStart
		panic("TODO: Style decoding")
	}

	return nil
}

func (stringPool) xml(*xmlContext) string { return "" }

func utf16EncodeStringPoolEntry(str string) []byte {
	var b bytes.Buffer
	w := endian.Writer(&b, device.LittleEndian)
	runes := utf16.Encode([]rune(str))
	encodeLength(w, uint32(len(runes)))
	for _, rune := range runes {
		w.Uint16(rune)
	}
	w.Uint16(0) /* bin_xml.py says so */
	return b.Bytes()
}

func (c *stringPool) encode() []byte {
	if len(c.styles) > 0 {
		panic("TODO: implement style encoding support.")
	}

	return encodeChunk(resStringPoolType, func(w binary.Writer) {
		totalHeaderLength := 8 + 5*4 // 8 for the basic header + the five uint32s below
		w.Uint32(uint32(len(c.strings)))
		w.Uint32(uint32(len(c.styles)))
		w.Uint32(c.flags)
		w.Uint32(uint32(totalHeaderLength + len(c.strings)*4)) // strings start after header and indices
		w.Uint32(0)                                            // stylesStart
	}, func(w binary.Writer) {
		encodedStrings := make([][]byte, len(c.strings))
		for i, str := range c.strings {
			encodedStrings[i] = utf16EncodeStringPoolEntry(str)
		}

		// encode indices
		index := 0
		for _, es := range encodedStrings {
			w.Uint32(uint32(index))
			index += len(es)
		}
		// compute padding (copied logic in bin_xml.py)
		padding := index % 4
		if padding == 3 {
			padding = 1
		} else if padding == 1 {
			padding = 3
		}

		// encode actual strings
		for _, es := range encodedStrings {
			w.Data(es)
		}

		for p := 0; p < padding; p++ {
			w.Uint8(0)
		}
	})
}

// findFromStringPoolIndex returns a pool reference for the string at the given
// index in the encoded pool.
func (p *stringPool) findFromStringPoolIndex(idx uint32) (stringPoolRef, bool) {
	for i, ptr := range p.ptrs {
		if uint32(ptr) == idx {
			return stringPoolRef{p, uint32(i)}, true
		}
	}
	return invalidStringPoolRef, false
}

func (p *stringPool) find(str string) (stringPoolRef, bool) {
	for i, ptr := range p.ptrs {
		if p.strings[ptr] == str {
			return stringPoolRef{p, uint32(i)}, true
		}
	}
	return invalidStringPoolRef, false
}

func (p *stringPool) ref(str string) stringPoolRef {
	ref, found := p.find(str)
	if found {
		return ref
	}
	return p.insertStringAtIndex(str, len(p.strings))
}

// insertStringAtIndex inserts a string at a given index in the pool and then
// updates the ptrs array, so that existing pool references continue to work.
// This index is the final position of the string in the encoded string pool.
func (p *stringPool) insertStringAtIndex(str string, index int) stringPoolRef {
	p.strings = append(p.strings[0:index], append([]string{str}, p.strings[index:]...)...)
	for i, ptr := range p.ptrs {
		if ptr >= index && ptr != missingString {
			p.ptrs[i] = ptr + 1
		}
	}
	p.ptrs = append(p.ptrs, index)
	return stringPoolRef{p, uint32(len(p.ptrs) - 1)}
}
