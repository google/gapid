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

package endian_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

const (
	readErr   = fault.Const("ReadError")
	writeErr  = fault.Const("WriteError")
	secondErr = fault.Const("SecondError")
)

type test struct {
	name   string
	values interface{}
	data   []byte
}

var tests = []test{
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
			0x00, 0x00,
			0xff, 0x7f,
			0x00, 0x80,
			0xff, 0xff,
		}},

	{"Uint16",
		[]uint16{0, 0xbeef, 0xc0de},
		[]byte{
			0x00, 0x00,
			0xef, 0xbe,
			0xde, 0xc0,
		}},

	{"Int32",
		[]int32{0, 2147483647, -2147483648, -1},
		[]byte{
			0x00, 0x00, 0x00, 0x00,
			0xff, 0xff, 0xff, 0x7f,
			0x00, 0x00, 0x00, 0x80,
			0xff, 0xff, 0xff, 0xff,
		}},

	{"Uint32",
		[]uint32{0, 0x01234567, 0x10abcdef},
		[]byte{
			0x00, 0x00, 0x00, 0x00,
			0x67, 0x45, 0x23, 0x01,
			0xef, 0xcd, 0xab, 0x10,
		}},

	{"Int64",
		[]int64{0, 9223372036854775807, -9223372036854775808, -1},
		[]byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		}},

	{"Uint64",
		[]uint64{0, 0x0123456789abcdef, 0xfedcba9876543210},
		[]byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01,
			0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe,
		}},

	{"Float32",
		[]float32{0, 1, 64.5},
		[]byte{
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x80, 0x3f,
			0x00, 0x00, 0x81, 0x42,
		}},

	{"Float64",
		[]float64{0, 1, 64.5},
		[]byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x50, 0x40,
		}},

	{"String",
		[]string{
			"Hello",
			"",
			"World",
			"こんにちは世界",
		},
		[]byte{
			'H', 'e', 'l', 'l', 'o', 0x00,
			0x00,
			'W', 'o', 'r', 'l', 'd', 0x00,
			0xe3, 0x81, 0x93, 0xe3, 0x82, 0x93, 0xe3, 0x81, 0xab, 0xe3, 0x81, 0xa1, 0xe3, 0x81, 0xaf, 0xe4, 0xb8, 0x96, 0xe7, 0x95, 0x8c, 0x00,
		}},
}

func factory(r io.Reader, w io.Writer) (binary.Reader, binary.Writer) {
	return endian.Reader(r, device.LittleEndian), endian.Writer(w, device.LittleEndian)
}

func TestReadWrite(t *testing.T) {
	ctx := log.Testing(t)
	for _, t := range tests {
		ctx := log.V{"name": t.name}.Bind(ctx)
		b := &bytes.Buffer{}
		reader, writer := factory(b, b)
		r := reflect.ValueOf(reader).MethodByName(t.name)
		w := reflect.ValueOf(writer).MethodByName(t.name)
		s := reflect.ValueOf(t.values)
		for i := 0; i < s.Len(); i++ {
			w.Call([]reflect.Value{s.Index(i)})
		}
		assert.For(ctx, "bytes").ThatSlice(b.Bytes()).Equals(t.data)
		for i := 0; i < s.Len(); i++ {
			ctx := log.V{"index": i}.Bind(ctx)
			expected := s.Index(i)
			result := r.Call(nil)
			got := result[0]
			assert.For(ctx, "err").ThatError(reader.Error()).Succeeded()
			assert.For(ctx, "val").That(got.Interface()).Equals(expected.Interface())
		}
	}
}

func TestData(t *testing.T) {
	ctx := log.Testing(t)
	for _, t := range tests {
		ctx := log.V{"name": t.name}.Bind(ctx)
		b := &bytes.Buffer{}
		reader, writer := factory(b, b)
		writer.Data(t.data)
		assert.For(ctx, "written").ThatSlice(b.Bytes()).Equals(t.data)
		got := make([]byte, len(t.data))
		reader.Data(got)
		assert.For(ctx, "result").ThatSlice(got).Equals(t.data)
	}
}

func TestCount(t *testing.T) {
	values := []uint32{0, 0x01234567, 0x10abcdef}
	raw := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x67, 0x45, 0x23, 0x01,
		0xef, 0xcd, 0xab, 0x10,
	}

	ctx := log.Testing(t)
	b := &bytes.Buffer{}
	reader, writer := factory(b, b)
	for _, v := range values {
		writer.Uint32(v)
	}
	assert.For(ctx, "bytes").ThatSlice(b.Bytes()).Equals(raw)
	for i, expect := range values {
		got := reader.Count()
		assert.For(ctx, "count at %v", i).That(got).Equals(expect)
	}
}

func TestSetErrors(t *testing.T) {
	ctx := log.Testing(t)
	for _, t := range tests {
		ctx := log.V{"name": t.name}.Bind(ctx)
		b := &bytes.Buffer{}
		reader, writer := factory(b, b)
		r := reflect.ValueOf(reader).MethodByName(t.name)
		w := reflect.ValueOf(writer).MethodByName(t.name)
		s := reflect.ValueOf(t.values)
		writer.SetError(writeErr)
		w.Call([]reflect.Value{s.Index(0)})
		assert.For(ctx, "err a").ThatError(writer.Error()).Equals(writeErr)
		writer.SetError(secondErr)
		w.Call([]reflect.Value{s.Index(0)})
		assert.For(ctx, "err b").ThatError(writer.Error()).Equals(writeErr)
		reader.SetError(readErr)
		r.Call(nil)
		assert.For(ctx, "err c").ThatError(reader.Error()).Equals(readErr)
		reader.SetError(secondErr)
		r.Call(nil)
		assert.For(ctx, "err d").ThatError(reader.Error()).Equals(readErr)
	}
	b := &bytes.Buffer{}
	data := []byte{1}
	reader, writer := factory(b, b)
	writer.SetError(writeErr)
	writer.Data(data)
	assert.For(ctx, "err e").ThatError(writer.Error()).Equals(writeErr)
	reader.SetError(readErr)
	reader.Data(data)
	assert.For(ctx, "err f").ThatError(reader.Error()).Equals(readErr)
}

func TestIOErrors(t *testing.T) {
	ctx := log.Testing(t)
	for _, t := range tests {
		ctx := log.V{"name": t.name}.Bind(ctx)
		reader, writer := factory(&bytesReader{}, &limitedWriter{})
		r := reflect.ValueOf(reader).MethodByName(t.name)
		w := reflect.ValueOf(writer).MethodByName(t.name)
		s := reflect.ValueOf(t.values)
		// Write twice, first time errors, second time compounds the error
		w.Call([]reflect.Value{s.Index(0)})
		assert.For(ctx, "err a").ThatError(writer.Error()).Equals(writeErr)
		// Read twice, first time errors, second time compounds the error
		r.Call(nil)
		assert.For(ctx, "err b").ThatError(reader.Error()).Equals(readErr)
	}
	buf := []byte{1}
	data := []byte{1, 2}
	reader, writer := factory(&bytesReader{buf}, &limitedWriter{Limit: 1})
	writer.Data(data)
	assert.For(ctx, "err c").ThatError(writer.Error()).Equals(io.ErrShortWrite)
	reader.Data(data)
	assert.For(ctx, "err d").ThatError(reader.Error()).Equals(readErr)
}

type bytesReader struct {
	Data []byte
}

func (b bytesReader) Add(data ...byte) bytesReader {
	return bytesReader{Data: append(b.Data, data...)}
}

func (d *bytesReader) Read(b []byte) (int, error) {
	if len(d.Data) == 0 {
		return 0, readErr
	}
	n := copy(b, d.Data)
	d.Data = d.Data[n:]
	return n, nil
}

type limitedWriter struct{ Limit int }

func (d *limitedWriter) Write(b []byte) (int, error) {
	if d.Limit <= 0 {
		return 0, writeErr
	}
	if d.Limit < len(b) {
		result := d.Limit
		d.Limit = 0
		return result, nil
	}
	return len(b), nil
}
