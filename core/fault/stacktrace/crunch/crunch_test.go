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

package crunch

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

var callstack = stacktrace.Callstack{
	0x400b281,
	0x4737568,
	0x4a62224,
	0x4c4d740,
	0x4c5a2c7,
	0x45760d5,
	0x4576ea1,
	0x45761b3,
	0x4576d91,
	0x45756f0,
	0x45761b3,
	0x4576d91,
	0x4cd89dc,
	0x45760d5,
	0x4576ea1,
	0x45761b3,
	0x4576d91,
	0x4576266,
	0x45146b5,
	0x45740a1,
	0x4575b4c,
	0x4c4028c,
	0x4da8d8b,
	0x457a4cc,
	0x4462d2b,
	0x4578293,
	0x457767b,
	0x457af13,
	0x456ed48,
	0x456e1c9,
	0x405aa61,
}
var raw []byte

func init() {
	buf := bytes.Buffer{}
	w := endian.Writer(&buf, device.LittleEndian)
	for _, pc := range callstack {
		w.Uint64(uint64(pc))
	}
	raw = buf.Bytes()
}

func TestCrunchUncrunch(t *testing.T) {
	ctx := log.Testing(t)
	data := Crunch(callstack)
	got := Uncrunch(data)
	assert.For(ctx, "callstack").ThatSlice(got).Equals(callstack)
	fmt.Printf("Crunched from %v to %d bytes\n", len(raw), len(data))
}

func TestCompressDecompress(t *testing.T) {
	ctx := log.Testing(t)
	for _, c := range compressors {
		values := toU64s(callstack)
		compressed := c.compress(values)
		decompressed := c.decompress(compressed)
		assert.For(ctx, "%T%v", c, c).ThatSlice(decompressed).Equals(values)
	}
}

func TestCompressorCrunchUncrunch(t *testing.T) {
	ctx := log.Testing(t)
	printCompressionRatio("raw", raw)
	for i, c := range compressors {
		bs := binary.BitStream{}
		crunchWith(callstack, c, &bs)
		uncrunched := uncrunchWith(c, &bs)
		assert.For(ctx, "%T%v", c, c).ThatSlice(uncrunched).Equals(callstack)
		printCompressionRatio(fmt.Sprint("  ", i), bs.Data)
	}
}

func printCompressionRatio(method string, data []byte) {
	fmt.Printf("%s: %.2f:1 %3.d bytes (flate: %v, gzip: %v, lzw: %v, zlib: %v)\n",
		method,
		float32(len(raw))/float32(len(data)),
		len(data),
		len(compressFlate(data)),
		len(compressGzip(data)),
		len(compressLZW(data)),
		len(compressZlib(data)),
	)
}

func compressFlate(data []byte) []byte {
	buf := bytes.Buffer{}
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func compressGzip(data []byte) []byte {
	buf := bytes.Buffer{}
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func compressLZW(data []byte) []byte {
	buf := bytes.Buffer{}
	w := lzw.NewWriter(&buf, lzw.LSB, 8)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func compressZlib(data []byte) []byte {
	buf := bytes.Buffer{}
	w, _ := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
