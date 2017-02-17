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
	"fmt"
	"reflect"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/test"
	"github.com/google/gapid/framework/binary/vle"

	"math/rand"
	"testing"
)

const count = 100000

type buffer struct {
	data []byte
	at   int
}

func (b *buffer) Write(p []byte) (int, error) {
	n := len(b.data)
	b.data = b.data[:n+len(p)]
	return copy(b.data[n:], p), nil
}

func (b *buffer) Read(p []byte) (int, error) {
	n := copy(p, b.data[b.at:])
	b.at += n
	return n, nil
}

func (b *buffer) Reset() {
	b.at = 0
	b.data = b.data[0:0]
}

func (b *buffer) Rewind() {
	b.at = 0
}

func prepare(a interface{}) (binary.Encoder, binary.Decoder, *buffer) {
	rand.Seed(1)
	s := reflect.ValueOf(a)
	size := 0
	for i := 0; i < s.Len(); i++ {
		e := s.Index(i)
		switch e.Kind() {
		case reflect.Int8:
			e.SetInt(rand.Int63())
			size += 2
		case reflect.Uint8:
			e.SetUint(uint64(rand.Int63()))
			size += 2
		case reflect.Int16:
			e.SetInt(rand.Int63())
			size += 3
		case reflect.Uint16:
			e.SetUint(uint64(rand.Int63()))
			size += 3
		case reflect.Int32:
			e.SetInt(rand.Int63())
			size += 5
		case reflect.Uint32:
			e.SetUint(uint64(rand.Int63()))
			size += 5
		case reflect.Int64:
			e.SetInt(int64((uint64(rand.Uint32()) << 32) | uint64(rand.Uint32())))
			size += 9
		case reflect.Uint64:
			e.SetUint((uint64(rand.Uint32()) << 32) | uint64(rand.Uint32()))
			size += 9
		case reflect.Interface: // binary.Object
			var v binary.Object
			switch rand.Uint32() & 0x3 {
			case 0:
				v = &test.TypeA{Data: fmt.Sprintf("%d", rand.Uint32()&0xfff)}
			case 1:
				v = &test.TypeB{Data: fmt.Sprintf("%d", rand.Uint32()&0xfff)}
			}
			if v != nil {
				e.Set(reflect.ValueOf(v))
			}
			size += 20 // high-estimate guess
		}
	}
	// build a big enough buffer, and wrap it in coders
	buf := &buffer{data: make([]byte, 0, size)}
	e := cyclic.Encoder(vle.Writer(buf))
	d := cyclic.Decoder(vle.Reader(buf))
	return e, d, buf
}

func TestRoundTripInt8(t *testing.T) {
	values := make([]int8, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Int8(v)
	}
	for i, v := range values {
		got := d.Int8()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripUint8(t *testing.T) {
	values := make([]uint8, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Uint8(v)
	}
	for i, v := range values {
		got := d.Uint8()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripInt16(t *testing.T) {
	values := make([]int16, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Int16(v)
	}
	for i, v := range values {
		got := d.Int16()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripUint16i(t *testing.T) {
	values := make([]uint16, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Uint16(v)
	}
	for i, v := range values {
		got := d.Uint16()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripInt32(t *testing.T) {
	values := make([]int32, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Int32(v)
	}
	for i, v := range values {
		got := d.Int32()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripUint32(t *testing.T) {
	values := make([]uint32, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Uint32(v)
	}
	for i, v := range values {
		got := d.Uint32()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripInt64(t *testing.T) {
	values := make([]int64, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Int64(v)
	}
	for i, v := range values {
		got := d.Int64()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func TestRoundTripUint64(t *testing.T) {
	values := make([]uint64, count)
	e, d, _ := prepare(values)
	for _, v := range values {
		e.Uint64(v)
	}
	for i, v := range values {
		got := d.Uint64()
		if v != got {
			t.Errorf("Bad value at %d. Expected: %x, got: %x", i, v, got)
			break
		}
	}
}

func BenchmarkEncodeUint64(b *testing.B) {
	values := make([]uint64, count)
	e, _, buf := prepare(values)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf.Reset()
		for _, v := range values {
			e.Uint64(v)
		}
	}
}

func BenchmarkDecodeUint64(b *testing.B) {
	values := make([]uint64, count)
	e, d, buf := prepare(values)
	for _, v := range values {
		e.Uint64(v)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf.Rewind()
		for range values {
			d.Uint64()
		}
	}
}

func BenchmarkEncodeObject(b *testing.B) {
	values := make([]binary.Object, count)
	e, _, buf := prepare(values)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf.Reset()
		for _, v := range values {
			e.Object(v)
		}
	}
}

func BenchmarkDecodeObject(b *testing.B) {
	values := make([]binary.Object, count)
	e, d, buf := prepare(values)
	for _, v := range values {
		e.Object(v)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf.Rewind()
		for range values {
			d.Object()
		}
	}
}

func BenchmarkUint64(b *testing.B) {
	values := make([]uint64, count)
	e, d, buf := prepare(values)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf.Reset()
		for _, v := range values {
			e.Uint64(v)
		}
		for range values {
			d.Uint64()
		}
	}
}
