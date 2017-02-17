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

package vle

import (
	"io"
	"math"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/math/f16"
)

// Reader creates a pod.Reader that reads from the provided io.Reader.
func Reader(r io.Reader) pod.Reader {
	return &reader{reader: r}
}

// Writer creates a pod.Writer that writes to the supplied io.Writer.
func Writer(w io.Writer) pod.Writer {
	return &writer{writer: w}
}

type reader struct {
	reader io.Reader
	tmp    [9]byte
	err    error
}

type writer struct {
	writer io.Writer
	tmp    [9]byte
	err    error
}

func shuffle32(v uint32) uint32 {
	return 0 |
		((v & 0x000000ff) << 24) |
		((v & 0x0000ff00) << 8) |
		((v & 0x00ff0000) >> 8) |
		((v & 0xff000000) >> 24)
}

func shuffle64(v uint64) uint64 {
	return 0 |
		((v & 0x00000000000000ff) << 56) |
		((v & 0x000000000000ff00) << 40) |
		((v & 0x0000000000ff0000) << 24) |
		((v & 0x00000000ff000000) << 8) |
		((v & 0x000000ff00000000) >> 8) |
		((v & 0x0000ff0000000000) >> 24) |
		((v & 0x00ff000000000000) >> 40) |
		((v & 0xff00000000000000) >> 56)
}

func (r *reader) intv() int64 {
	uv := r.uintv()
	v := int64(uv >> 1)
	if uv&1 != 0 {
		v = ^v
	}
	return v
}

func (w *writer) intv(v int64) {
	uv := uint64(v) << 1
	if v < 0 {
		uv = ^uv
	}
	w.uintv(uv)
}

func (r *reader) uintv() uint64 {
	tag := r.Uint8()
	count := uint(0)
	for ; ((0x80 >> count) & tag) != 0; count++ {
	}
	v := uint64(tag & (byte(0xff) >> count))
	if count == 0 {
		return v
	}
	r.Data(r.tmp[:count])
	for i := uint(0); i < count; i++ {
		v = (v << 8) | uint64(r.tmp[i])
	}
	return v
}

func (w *writer) uintv(v uint64) {
	space := uint64(0x7f)
	tag := byte(0)
	for o := 8; true; o-- {
		if v <= space {
			w.tmp[o] = byte(v) | byte(tag)
			w.Data(w.tmp[o:])
			return
		}
		w.tmp[o] = byte(v)
		v >>= 8
		space >>= 1
		tag = (tag >> 1) | 0x80
	}
}

func (r *reader) Data(p []byte) {
	if r.err != nil {
		return
	}
	_, r.err = io.ReadFull(r.reader, p)
}

func (w *writer) Data(data []byte) {
	if w.err != nil {
		return
	}
	n, err := w.writer.Write(data)
	if err != nil {
		w.err = err
		return
	}
	if n != len(data) {
		w.err = io.ErrShortWrite
		return
	}
}

func (r *reader) Bool() bool {
	b := r.Uint8()
	return b != 0
}

func (w *writer) Bool(v bool) {
	if v {
		w.Uint8(1)
	} else {
		w.Uint8(0)
	}
}

func (r *reader) Int8() int8 {
	return int8(r.Uint8())
}

func (w *writer) Int8(v int8) {
	w.Uint8(uint8(v))
}

func (r *reader) Uint8() uint8 {
	if r.err != nil {
		return 0
	}
	b := r.tmp[:1]
	_, r.err = io.ReadFull(r.reader, b[:1])
	return b[0]
}

func (w *writer) Uint8(v uint8) {
	w.tmp[0] = v
	w.Data(w.tmp[:1])
}

func (r *reader) Int16() int16         { return int16(r.intv()) }
func (w *writer) Int16(v int16)        { w.intv(int64(v)) }
func (r *reader) Uint16() uint16       { return uint16(r.uintv()) }
func (w *writer) Uint16(v uint16)      { w.uintv(uint64(v)) }
func (r *reader) Int32() int32         { return int32(r.intv()) }
func (w *writer) Int32(v int32)        { w.intv(int64(v)) }
func (r *reader) Uint32() uint32       { return uint32(r.uintv()) }
func (w *writer) Uint32(v uint32)      { w.uintv(uint64(v)) }
func (r *reader) Int64() int64         { return r.intv() }
func (w *writer) Int64(v int64)        { w.intv(v) }
func (r *reader) Uint64() uint64       { return r.uintv() }
func (w *writer) Uint64(v uint64)      { w.uintv(v) }
func (r *reader) Float16() f16.Number  { return f16.Number(r.Uint16()) }
func (w *writer) Float16(v f16.Number) { w.Uint16(uint16(v)) }

func (r *reader) Float32() float32 {
	return math.Float32frombits(shuffle32(r.Uint32()))
}

func (w *writer) Float32(v float32) {
	w.Uint32(shuffle32(math.Float32bits(v)))
}

func (r *reader) Float64() float64 {
	return math.Float64frombits(shuffle64(r.Uint64()))
}

func (w *writer) Float64(v float64) {
	w.Uint64(shuffle64(math.Float64bits(v)))
}

func (r *reader) String() string {
	s := make([]byte, r.Uint32())
	r.Data(s)
	return string(s)
}

func (w *writer) String(v string) {
	w.Uint32(uint32(len(v)))
	w.Data([]byte(v))
}

func (r *reader) Simple(o pod.Readable) {
	o.ReadSimple(r)
}

func (w *writer) Simple(o pod.Writable) {
	o.WriteSimple(w)
}

func (r *reader) Count() uint32 {
	return r.Uint32()
}

func (r *reader) Error() error {
	return r.err
}

func (w *writer) Error() error {
	return w.err
}

func (r *reader) SetError(err error) {
	if r.err != nil {
		return
	}
	r.err = err
}

func (w *writer) SetError(err error) {
	if w.err != nil {
		return
	}
	w.err = err
}
