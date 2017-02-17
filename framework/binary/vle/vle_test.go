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

package vle_test

import (
	"io"
	"testing"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary/test"
	"github.com/google/gapid/framework/binary/vle"
)

var testData = []test.ReadWriteTests{
	{"Bool",
		[]bool{true, false},
		[]byte{1, 0},
	},
	{"Int8",
		[]int8{0, 127, -128, -1},
		[]byte{0x00, 0x7f, 0x80, 0xff},
	},
	{"Uint8",
		[]uint8{0x00, 0x7f, 0x80, 0xff},
		[]byte{0x00, 0x7f, 0x80, 0xff},
	},

	{"Int16",
		[]int16{0, 32767, -32768, -1},
		[]byte{
			0x00,
			0xc0, 0xff, 0xfe,
			0xc0, 0xff, 0xff,
			0x01,
		}},

	{"Uint16",
		[]uint16{0, 0xbeef, 0xc0de},
		[]byte{
			0x00,
			0xc0, 0xbe, 0xef,
			0xc0, 0xc0, 0xde,
		}},

	{"Int32",
		[]int32{0, 2147483647, -2147483648, -1},
		[]byte{
			0x00,
			0xf0, 0xff, 0xff, 0xff, 0xfe,
			0xf0, 0xff, 0xff, 0xff, 0xff,
			0x01,
		}},

	{"Uint32",
		[]uint32{0, 0x01234567, 0x10abcdef},
		[]byte{
			0x00,
			0xe1, 0x23, 0x45, 0x67,
			0xf0, 0x10, 0xab, 0xcd, 0xef,
		}},

	{"Int64",
		[]int64{0, 9223372036854775807, -9223372036854775808, -1},
		[]byte{
			0x00,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0x01,
		}},

	{"Uint64",
		[]uint64{0, 0x0123456789abcdef, 0xfedcba9876543210, 0xffffffff},
		[]byte{
			0x00,
			0xff, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
			0xff, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
			0xf0, 0xff, 0xff, 0xff, 0xff,
		}},

	{"Float32",
		[]float32{0, 1, 64.5},
		[]byte{
			0x00,
			0xc0, 0x80, 0x3f,
			0xc0, 0x81, 0x42,
		}},

	{"Float64",
		[]float64{0, 1, 64.5},
		[]byte{
			0x00,
			0xc0, 0xf0, 0x3f,
			0xe0, 0x20, 0x50, 0x40,
		}},

	{"String",
		[]string{
			"Hello",
			"",
			"World",
			"こんにちは世界",
		},
		[]byte{
			0x05, 'H', 'e', 'l', 'l', 'o',
			0x00,
			0x05, 'W', 'o', 'r', 'l', 'd',
			0x15, 0xe3, 0x81, 0x93, 0xe3, 0x82, 0x93, 0xe3, 0x81, 0xab, 0xe3, 0x81, 0xa1, 0xe3, 0x81, 0xaf, 0xe4, 0xb8, 0x96, 0xe7, 0x95, 0x8c,
		}},
}

func factory(r io.Reader, w io.Writer) (pod.Reader, pod.Writer) {
	return vle.Reader(r), vle.Writer(w)
}

func TestReadWrite(t *testing.T) {
	ctx := log.Testing(t)
	test.ReadWrite(ctx, testData, factory)
}

func TestData(t *testing.T) {
	ctx := log.Testing(t)
	test.ReadWriteData(ctx, testData, factory)
}

func TestCount(t *testing.T) {
	values := []uint32{0, 0x01234567, 0x10abcdef}
	raw := []byte{
		0x00,
		0xe1, 0x23, 0x45, 0x67,
		0xf0, 0x10, 0xab, 0xcd, 0xef,
	}

	ctx := log.Testing(t)
	test.ReadWriteCount(ctx, values, raw, factory)
}

func TestSimple(t *testing.T) {
	values := []test.Simple{test.Simple(0), test.Simple(127), test.Simple(-128), test.Simple(-1)}
	raw := []byte{0x00, 0x7f, 0x80, 0xff}
	ctx := log.Testing(t)
	test.ReadWriteSimple(ctx, values, raw, factory)
}

func TestSetErrors(t *testing.T) {
	ctx := log.Testing(t)
	test.ReadWriteErrors(ctx, testData, factory)
}

func TestIOErrors(t *testing.T) {
	ctx := log.Testing(t)
	test.ReadWriteIOErrors(ctx, testData, factory)
}
