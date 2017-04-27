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

package test

import (
	"bytes"
	"context"
	"io"
	"reflect"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/log"
)

type ReadWriteTests struct {
	Name   string
	Values interface{}
	Data   []byte
}

type Factory func(io.Reader, io.Writer) (binary.Reader, binary.Writer)

func ReadWrite(ctx context.Context, tests []ReadWriteTests, factory Factory) {
	for _, e := range tests {
		ctx := log.V{"name": e.Name}.Bind(ctx)
		b := &bytes.Buffer{}
		reader, writer := factory(b, b)
		r := reflect.ValueOf(reader).MethodByName(e.Name)
		w := reflect.ValueOf(writer).MethodByName(e.Name)
		s := reflect.ValueOf(e.Values)
		for i := 0; i < s.Len(); i++ {
			w.Call([]reflect.Value{s.Index(i)})
		}
		assert.With(ctx).ThatSlice(b.Bytes()).Equals(e.Data)
		for i := 0; i < s.Len(); i++ {
			ctx := log.V{"index": i}.Bind(ctx)
			expected := s.Index(i)
			result := r.Call(nil)
			got := result[0]
			assert.With(ctx).ThatError(reader.Error()).Succeeded()
			assert.With(ctx).That(got.Interface()).Equals(expected.Interface())
		}
	}
}

func ReadWriteData(ctx context.Context, tests []ReadWriteTests, factory Factory) {
	for _, e := range tests {
		ctx := log.V{"name": e.Name}.Bind(ctx)
		b := &bytes.Buffer{}
		reader, writer := factory(b, b)
		writer.Data(e.Data)
		assert.For(ctx, "written").ThatSlice(b.Bytes()).Equals(e.Data)
		got := make([]byte, len(e.Data))
		reader.Data(got)
		assert.For(ctx, "result").ThatSlice(got).Equals(e.Data)
	}
}

func ReadWriteCount(ctx context.Context, values []uint32, raw []byte, factory Factory) {
	b := &bytes.Buffer{}
	reader, writer := factory(b, b)
	for _, v := range values {
		writer.Uint32(v)
	}
	assert.For(ctx, "bytes").ThatSlice(b.Bytes()).Equals(raw)
	for _, expect := range values {
		got := reader.Count()
		assert.With(ctx).That(got).Equals(expect)
	}
}

func ReadWriteSimple(ctx context.Context, values []Simple, raw []byte, factory Factory) {
	b := &bytes.Buffer{}
	reader, writer := factory(b, b)
	for _, v := range values {
		writer.Simple(v)
	}
	assert.For(ctx, "bytes").ThatSlice(b.Bytes()).Equals(raw)
	for _, expect := range values {
		var got Simple
		reader.Simple(&got)
		assert.With(ctx).That(got).Equals(expect)
	}
}

func ReadWriteErrors(ctx context.Context, tests []ReadWriteTests, factory Factory) {
	for _, e := range tests {
		ctx := log.V{"name": e.Name}.Bind(ctx)
		b := &bytes.Buffer{}
		reader, writer := factory(b, b)
		r := reflect.ValueOf(reader).MethodByName(e.Name)
		w := reflect.ValueOf(writer).MethodByName(e.Name)
		s := reflect.ValueOf(e.Values)
		writer.SetError(WriteError)
		w.Call([]reflect.Value{s.Index(0)})
		assert.With(ctx).ThatError(writer.Error()).Equals(WriteError)
		writer.SetError(SecondError)
		w.Call([]reflect.Value{s.Index(0)})
		assert.With(ctx).ThatError(writer.Error()).Equals(WriteError)
		reader.SetError(ReadError)
		r.Call(nil)
		assert.With(ctx).ThatError(reader.Error()).Equals(ReadError)
		reader.SetError(SecondError)
		r.Call(nil)
		assert.With(ctx).ThatError(reader.Error()).Equals(ReadError)
	}
	b := &bytes.Buffer{}
	data := []byte{1}
	reader, writer := factory(b, b)
	writer.SetError(WriteError)
	writer.Data(data)
	assert.With(ctx).ThatError(writer.Error()).Equals(WriteError)
	reader.SetError(ReadError)
	reader.Data(data)
	assert.With(ctx).ThatError(reader.Error()).Equals(ReadError)
}

func ReadWriteIOErrors(ctx context.Context, tests []ReadWriteTests, factory Factory) {
	for _, e := range tests {
		ctx := log.V{"name": e.Name}.Bind(ctx)
		reader, writer := factory(&Bytes{}, &LimitedWriter{})
		r := reflect.ValueOf(reader).MethodByName(e.Name)
		w := reflect.ValueOf(writer).MethodByName(e.Name)
		s := reflect.ValueOf(e.Values)
		// Write twice, first time errors, second time compounds the error
		w.Call([]reflect.Value{s.Index(0)})
		assert.With(ctx).ThatError(writer.Error()).Equals(WriteError)
		// Read twice, first time errors, second time compounds the error
		r.Call(nil)
		assert.With(ctx).ThatError(reader.Error()).Equals(ReadError)
	}
	buf := []byte{1}
	data := []byte{1, 2}
	reader, writer := factory(&Bytes{Data: buf}, &LimitedWriter{Limit: 1})
	writer.Data(data)
	assert.With(ctx).ThatError(writer.Error()).Equals(io.ErrShortWrite)
	reader.Data(data)
	assert.With(ctx).ThatError(reader.Error()).Equals(ReadError)
}
