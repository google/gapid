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

// Package crunch provides functions to compress and decompress stacktraces.
//
// crunch is intented to be used to compress a stacktrace down to something that
// can fit inside a Google Analytics exception payload (150 bytes). crunch
// uses a number of different compression algorithms to compact the list of
// program counters down to the smallest number of bytes possible. Given the
// exhaustive search, this is optimized for output size over compression
// performance.
//
// Note that the output of a crunch only makes sense to the particular build
// used to create it.
package crunch

import (
	"fmt"
	"sort"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/fault/stacktrace"
)

// Tweakables
const (
	bitChunkMinSize   = 2
	bitChunkIncrement = 1

	debug = false
)

func printVLE(v uint64) {
	fmt.Printf("[")
	writeVLE(v, bitPrinter{})
	fmt.Printf("]")
}

type bitPrinter struct{}

func (bitPrinter) Write(v uint64, c uint32) {
	for ; c > 0; c-- {
		fmt.Printf("%v", v&1)
		v >>= 1
	}
}
func (bitPrinter) WriteBit(v uint64) {
	if v == 0 {
		fmt.Print("⁰")
	} else {
		fmt.Print("¹")
	}
}

type bitwriter interface {
	Write(bits uint64, count uint32)
	WriteBit(bit uint64)
}

func writeVLE(i uint64, w bitwriter) {
	chunkSize := uint32(bitChunkMinSize)
	for i != 0 {
		w.WriteBit(0)
		w.Write(uint64(i), chunkSize)
		i >>= chunkSize
		chunkSize += bitChunkIncrement
	}
	w.WriteBit(1)
}

func readVLE(bs *binary.BitStream) (uint64, bool) {
	chunkSize := uint32(bitChunkMinSize)
	i, s := uint64(0), uint32(0)
	for {
		if !bs.CanRead(1) {
			return 0, false
		}
		if bs.ReadBit() == 1 {
			break
		}
		if !bs.CanRead(chunkSize) {
			return 0, false
		}
		i |= bs.Read(chunkSize) << s
		s += chunkSize
		chunkSize += bitChunkIncrement
	}
	return uint64(i), true
}

type compressor interface {
	compress([]uint64) []uint64
	decompress([]uint64) []uint64
}

type noCompressor struct{}

func (p noCompressor) compress(c []uint64) []uint64 {
	return c
}

func (p noCompressor) decompress(c []uint64) []uint64 {
	return c
}

type baseCompressor struct{}

func (p baseCompressor) compress(c []uint64) []uint64 {
	base := ^(uint64)(0)
	for _, v := range c {
		if v < base {
			base = v
		}
	}
	out := make([]uint64, len(c)+1)
	out[0] = base
	for i, v := range c {
		out[i+1] = v - base
	}
	return out
}

func (p baseCompressor) decompress(c []uint64) []uint64 {
	if len(c) < 2 {
		return c
	}
	out := make([]uint64, len(c)-1)
	base := c[0]
	for i, pc := range c[1:] {
		out[i] = pc + base
	}
	return out
}

type xorCompressor struct{}

func (p xorCompressor) compress(c []uint64) []uint64 {
	var prev uint64
	out := make([]uint64, len(c))
	for i, v := range c {
		out[i], prev = v^prev, v
	}
	return out
}

func (p xorCompressor) decompress(c []uint64) []uint64 {
	var prev uint64
	out := make([]uint64, len(c))
	for i, val := range c {
		v := val ^ prev
		out[i], prev = v, v
	}
	return out
}

type backRefCompressor struct {
	pack   func(ref, val uint64) uint64
	unpack func(ref, packed uint64) uint64
}

func packXor(ref, val uint64) uint64      { return ref ^ val }
func unpackXor(ref, packed uint64) uint64 { return ref ^ packed }

func packDiff(ref, val uint64) uint64 {
	if ref > val {
		return (ref - val) << 1
	}
	return (val-ref)<<1 | 1
}
func unpackDiff(ref, val uint64) uint64 {
	sign, val := val&1, val>>1
	if sign == 0 {
		return ref + val
	}
	return ref - val
}

func (p backRefCompressor) compress(c []uint64) []uint64 {
	out := make([]uint64, 0, len(c))

	size := func(val uint64) uint32 {
		bs := binary.BitStream{}
		writeVLE(val, &bs)
		return bs.WritePos
	}

	for i, v := range c {
		bestValue := v
		bestIndex := 0
		bestScore := size(uint64(bestIndex)) + size(bestValue)
		for j := 0; j < i; j++ {
			idx := i - j
			val := p.pack(v, c[j])
			if score := size(val) + size(uint64(idx)); score < bestScore {
				bestValue, bestIndex, bestScore = val, idx, score
			}
		}

		out = append(out, uint64(bestIndex), bestValue)

		if debug {
			printVLE(v)
			fmt.Printf("(%d) -- <%v> ", size(v), bestIndex)
			printVLE(uint64(bestIndex))
			fmt.Printf(", ")
			printVLE(bestValue)
			fmt.Printf("(%d) ", bestScore)
			fmt.Printf("\n")
		}
	}
	return out
}

func (p backRefCompressor) decompress(c []uint64) []uint64 {
	out := make([]uint64, 0, len(c)/2)
	for len(c) >= 2 {
		idx, val := int(c[0]), c[1]
		if idx == 0 {
			out = append(out, val)
		} else {
			ref := out[len(out)-idx]
			out = append(out, p.unpack(ref, val))
		}
		c = c[2:]
	}
	return out
}

type dictionaryCompressor struct{}

func (p dictionaryCompressor) compress(c []uint64) []uint64 {
	unique := map[uint64]struct{}{}
	for _, v := range c {
		unique[v] = struct{}{}
	}

	sorted := make([]uint64, 0, len(unique))
	for v := range unique {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	indices := map[uint64]uint64{}
	for i, v := range sorted {
		indices[v] = uint64(i)
	}

	out := make([]uint64, 0, len(c)*2)

	// Delta
	prev := uint64(0)
	for _, v := range sorted {
		delta := v - prev
		out, prev = append(out, delta), v
	}

	// Delimiter
	out = append(out, 0)

	// Values
	for _, v := range c {
		idx := indices[v]
		out = append(out, uint64(idx))
	}

	return out
}

func (p dictionaryCompressor) decompress(c []uint64) []uint64 {
	table, prev := make([]uint64, 0, len(c)), uint64(0)
	for i, delta := range c {
		if delta == 0 {
			c = c[i+1:]
			break
		}
		val := prev + delta
		table, prev = append(table, val), val
	}

	out := make([]uint64, len(c))
	for i, v := range c {
		out[i] = table[v]
	}
	return out
}

type compressorList []compressor

func (l compressorList) compress(c []uint64) []uint64 {
	for _, p := range l {
		c = p.compress(c)
	}
	return c
}

func (l compressorList) decompress(c []uint64) []uint64 {
	for i := len(l) - 1; i >= 0; i-- {
		p := l[i]
		c = p.decompress(c)
	}
	return c
}

const compressorIdxBits = 2

var compressors = [1 << compressorIdxBits]compressor{
	/* 0 */ dictionaryCompressor{},
	/* 1 */ xorCompressor{},
	/* 2 */ compressorList{baseCompressor{}, backRefCompressor{packXor, unpackXor}},
	/* 3 */ compressorList{baseCompressor{}, backRefCompressor{packDiff, unpackDiff}},
}

// Crunch compresses the callstack down to a minimal compressed form.
func Crunch(c stacktrace.Callstack) []byte {
	var smallest []byte
	for i, p := range compressors {
		bs := binary.BitStream{}
		bs.Write(uint64(i), compressorIdxBits)
		crunchWith(c, p, &bs)
		if smallest == nil || len(bs.Data) < len(smallest) {
			smallest = bs.Data
		}
	}
	return smallest
}

func crunchWith(c stacktrace.Callstack, p compressor, bs *binary.BitStream) {
	compressed := p.compress(toU64s(c))
	for _, pc := range compressed {
		writeVLE(pc, bs)
	}
}

// Uncrunch decompresses the callstack from the minimal compressed form.
func Uncrunch(data []byte) stacktrace.Callstack {
	bs := binary.BitStream{Data: data}
	idx := bs.Read(compressorIdxBits)
	p := compressors[idx]
	return uncrunchWith(p, &bs)
}

func uncrunchWith(p compressor, bs *binary.BitStream) stacktrace.Callstack {
	vals := []uint64{}
	for {
		v, ok := readVLE(bs)
		if !ok {
			return toCallstack(p.decompress(vals))
		}
		vals = append(vals, v)
	}
}

func toU64s(c stacktrace.Callstack) []uint64 {
	out := make([]uint64, len(c))
	for i := range c {
		out[i] = uint64(c[i])
	}
	return out
}

func toCallstack(v []uint64) stacktrace.Callstack {
	out := make(stacktrace.Callstack, len(v))
	for i := range v {
		out[i] = uintptr(v[i])
	}
	return out
}
