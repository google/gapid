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

package memory

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/database"
	"github.com/pkg/errors"
)

// readFully returns the entire data produced by the given Reader, by reading up to maxRead bytes at a time.
func readFully(r io.Reader, maxRead int) ([]byte, error) {
	var b bytes.Buffer
	readBuffer := make([]byte, maxRead)
	for {
		n, err := r.Read(readBuffer)
		b.Write(readBuffer[:n])
		if errors.Cause(err) == io.EOF {
			return b.Bytes(), nil
		} else if err != nil {
			return b.Bytes(), err
		}
	}
}

func checkData(ctx context.Context, s Data, expected []byte) {
	assert.For(ctx, "size").That(s.Size()).Equals(uint64(len(expected)))

	for _, offset := range []uint64{0, 1, s.Size() - 1, s.Size()} {
		got := make([]byte, len(expected)-int(offset))
		err := s.Get(ctx, offset, got)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		assert.For(ctx, "got").ThatSlice(got).Equals(expected[offset:])
	}

	for _, maxReadSize := range []int{1, 2, 3, 512} {
		got, err := readFully(s.NewReader(ctx), maxReadSize)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		assert.For(ctx, "got").ThatSlice(got).Equals(expected)
	}
}

func TestBlobSlice(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	data := Blob([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	for _, test := range []struct {
		rng      Range
		expected []byte
	}{
		{Range{Base: 0, Size: 10}, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{Range{Base: 3, Size: 3}, []byte{3, 4, 5}},
		{Range{Base: 6, Size: 3}, []byte{6, 7, 8}},
	} {
		checkData(ctx, data.Slice(test.rng), test.expected)
	}
}

// Write layout:
//
//	     0    1    2    3    4    5
//	  ╔════╤════╤════╤════╤════╗────┐
//	0 ║ 10 │ 11 │ 12 │ 13 │ 14 ║    │
//	  ╚════╧════╧════╧════╧════╝────┘
//	  ╔════╤════╤════╤════╤════╗────┐
//	1 ║ 20 │ 21 │ 22 │ 23 │ 24 ║    │
//	  ╚════╧════╧════╧════╧════╝────┘
//	  ╔════╤════╗────┬────┬────┬────┐
//	2 ║ 30 │ 31 ║    │    │    │    │
//	  ╚════╧════╝────┴────┴────┴────┘
//	  ╔════╤════╤════╤════╤════╤════╗
//	3 ║ 40 │ 41 │ 42 │ 43 │ 44 │ 45 ║
//	  ╚════╧════╧════╧════╧════╧════╝
func TestPoolBlobWriteRead(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	p := Pool{}
	for _, test := range []struct {
		data     Data
		expected []byte
	}{
		{Blob([]byte{10, 11, 12, 13, 14}), []byte{10, 11, 12, 13, 14, 0}},
		{Blob([]byte{20, 21, 22, 23, 24}), []byte{20, 21, 22, 23, 24, 0}},
		{Blob([]byte{30, 31}), []byte{30, 31, 22, 23, 24, 0}},
		{Blob([]byte{40, 41, 42, 43, 44, 45}), []byte{40, 41, 42, 43, 44, 45}},
	} {
		p.Write(0, test.data)

		checkData(ctx, p.Slice(Range{Base: 0, Size: 6}), test.expected)
	}
}

// Write layout:
//
//	     0    1    2    3    4    5    6    7    8    9   10   11
//	  ┌────╔════╤════╤════╗────┬────┬────┬────┬────┬────┬────┬────┐
//	0 │    ║ 10 │ 11 │ 12 ║    │    │    │    │    │    │    │    │
//	  └────╚════╧════╧════╝────┴────┴────┴────┴────┴────┴────┴────┘
//	  ┌────┬────┬────┬────┬────┬────┬────╔════╤════╤════╤════╗────┐
//	1 │    │    │    │    │    │    │    ║ 20 │ 21 │ 22 │ 23 ║    │
//	  └────┴────┴────┴────┴────┴────┴────╚════╧════╧════╧════╝────┘
//	  ┌────┬────╔════╤════╗────┬────┬────┬────┬────┬────┬────┬────┐
//	2 │    │    ║ 30 │ 31 ║    │    │    │    │    │    │    │    │
//	  └────┴────╚════╧════╝────┴────┴────┴────┴────┴────┴────┴────┘
//	  ┌────┬────╔════╤════╤════╗────┬────┬────┬────┬────┬────┬────┐
//	3 │    │    ║ 40 │ 41 │ 42 ║    │    │    │    │    │    │    │
//	  └────┴────╚════╧════╧════╝────┴────┴────┴────┴────┴────┴────┘
//	  ┌────┬────┬────┬────┬────┬────┬────┬────╔════╗────┬────┬────┐
//	4 │    │    │    │    │    │    │    │    ║ 50 ║    │    │    │
//	  └────┴────┴────┴────┴────┴────┴────┴────╚════╝────┴────┴────┘
func TestMemoryBlobWriteReadScattered(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	p := Pool{}
	p.Write(1, Blob([]byte{10, 11, 12}))
	p.Write(7, Blob([]byte{20, 21, 22, 23}))
	p.Write(2, Blob([]byte{30, 31}))
	p.Write(2, Blob([]byte{40, 41, 42}))
	p.Write(8, Blob([]byte{50}))

	for _, test := range []struct {
		rng      Range
		expected []byte
	}{
		{Range{Base: 0, Size: 12}, []byte{0, 10, 40, 41, 42, 00, 00, 20, 50, 22, 23, 00}},
		{Range{Base: 1, Size: 10}, []byte{10, 40, 41, 42, 00, 00, 20, 50, 22, 23}},
		{Range{Base: 2, Size: 3}, []byte{40, 41, 42}},
		{Range{Base: 3, Size: 6}, []byte{41, 42, 0, 0, 20, 50}},
		{Range{Base: 3, Size: 7}, []byte{41, 42, 0, 0, 20, 50, 22}},
		{Range{Base: 5, Size: 2}, []byte{0, 0}},
		{Range{Base: 8, Size: 1}, []byte{50}},
	} {
		checkData(ctx, p.Slice(test.rng), test.expected)
	}

}

// Write layout:
//
//	     0    1    2    3    4    5    6    7    8    9   10   11
//	  ┌────╔════╤════╤════╗────┬────┬────┬────┬────┬────┬────┬────┐
//	0 │    ║A 10│ 11 │ 12 ║    │    │    │    │    │    │    │    │
//	  └────╚════╧════╧════╝────┴────┴────┴────┴────┴────┴────┴────┘
//	  ┌────┬────┬────┬────┬────┬────┬────╔════╤════╤════╤════╗────┐
//	1 │    │    │    │    │    │    │    ║B 20│ 21 │ 22 │ 23 ║    │
//	  └────┴────┴────┴────┴────┴────┴────╚════╧════╧════╧════╝────┘
//	  ┌────┬────╔════╤════╗────┬────┬────┬────┬────┬────┬────┬────┐
//	2 │    │    ║C 30│ 31 ║    │    │    │    │    │    │    │    │
//	  └────┴────╚════╧════╝────┴────┴────┴────┴────┴────┴────┴────┘
//	  ┌────┬────╔════╤════╤════╗────┬────┬────┬────┬────┬────┬────┐
//	3 │    │    ║D 40│ 41 │ 42 ║    │    │    │    │    │    │    │
//	  └────┴────╚════╧════╧════╝────┴────┴────┴────┴────┴────┴────┘
//	  ┌────┬────┬────┬────┬────┬────┬────┬────╔════╗────┬────┬────┐
//	4 │    │    │    │    │    │    │    │    ║E 50║    │    │    │
//	  └────┴────┴────┴────┴────┴────┴────┴────╚════╝────┴────┴────┘
func TestMemoryResourceWriteReadScattered(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	resA, _ := database.Store(ctx, []byte{10, 11, 12})
	resB, _ := database.Store(ctx, []byte{20, 21, 22, 23})
	resC, _ := database.Store(ctx, []byte{30, 31})
	resD, _ := database.Store(ctx, []byte{40, 41, 42})
	resE, _ := database.Store(ctx, []byte{50})

	p := Pool{}
	p.Write(1, Resource(resA, 3))
	p.Write(7, Resource(resB, 4))
	p.Write(2, Resource(resC, 2))
	p.Write(2, Resource(resD, 3))
	p.Write(8, Resource(resE, 1))

	for _, test := range []struct {
		rng      Range
		expected []byte
	}{
		{Range{Base: 0, Size: 12}, []byte{0, 10, 40, 41, 42, 00, 00, 20, 50, 22, 23, 00}},
		{Range{Base: 1, Size: 10}, []byte{10, 40, 41, 42, 00, 00, 20, 50, 22, 23}},
		{Range{Base: 2, Size: 3}, []byte{40, 41, 42}},
		{Range{Base: 5, Size: 2}, []byte{0, 0}},
		{Range{Base: 8, Size: 1}, []byte{50}},
	} {
		slice := p.Slice(test.rng)
		checkData(ctx, slice, test.expected)

		gotID, err := slice.ResourceID(ctx)
		assert.For(ctx, "err").ThatError(err).Succeeded()

		expectedID, _ := database.Store(ctx, test.expected)
		assert.For(ctx, "id").That(gotID).Equals(expectedID)
	}
}

func TestPoolSliceReaderErrorPropagation(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	p := Pool{}
	p.Write(2, Resource(id.ID{}, 5))

	got, err := readFully(p.Slice(Range{Base: 0, Size: 10}).NewReader(ctx), 512)
	assert.For(ctx, "len").That(len(got)).Equals(2)
	assert.For(ctx, "err").ThatError(err).Failed()
}

// Write layout:
//
//	innerPool:
//	                         0    1    2
//	                      ┌────┬────┬────┐
//	0                     │ 4  │ 55 │ 66 │
//	                      └────┴────┴────┘
//	                        │    │    │
//	midPool:                │    │    │
//	                    0    1    2    3    4    5    6     7
//	                 ┌────╔════╤════╤════╗────┬────┬────┬────┐
//	0                │    ║ 4i │ 55i│ 66i║    │    │    │    │
//	                 └────╚════╧════╧════╝────┴────┴────┴────┘
//	                 ┌────┬────╔════╤════╤════╤════╗────┬────┐
//	1                │    │    ║ 5  │ 6  │ 7  │ 88 ║    │    │
//	                 └────┴────╚════╧════╧════╧════╝────┴────┘
//	                 ┌────┬────┬────┬────┬────╔════╤════╤════╗
//	2                │    │    │    │    │    ║ 8r │ 9r │ 10r║
//	                 └────┴────┴────┴────┴────╚════╧════╧════╝
//	                        │    │    │    │    │    │    │
//	outerPool:              │    │    │    │    │    │    │
//	     0    1    2    3    4    5    6    7    8     9    10
//	  ┌────╔════╤════╤════╤════╗────┬────┬────┬────┬────┬────┐
//	0 │    ║ 1  │ 2  │ 3  │ 44 ║    │    │    │    │    │    │
//	  └────╚════╧════╧════╧════╝────┴────┴────┴────┴────┴────┘
//	                        │    │    │    │    │    │    │
//	  ┌────┬────┬────┬────╔════╤════╤════╤════╤════╤════╤════╗
//	1 │    │    │    │    ║ 4m │ 5m │ 6m │ 7m │ 8m │ 9m │ 10m║
//	  └────┴────┴────┴────╚════╧════╧════╧════╧════╧════╧════╝
func TestSliceNesting(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	innerPool := Pool{}
	innerPool.Write(0, Blob([]byte{4, 55, 66}))

	midPool := Pool{}
	midPool.Write(1, innerPool.Slice(Range{Size: 3}))
	midPool.Write(2, Blob([]byte{5, 6, 7, 88}))
	res, _ := database.Store(ctx, []byte{8, 9, 10})
	midPool.Write(5, Resource(res, 3))

	outerPool := Pool{}
	outerPool.Write(1, Blob([]byte{1, 2, 3, 44}))
	outerPool.Write(4, midPool.Slice(Range{Base: 1, Size: 7}))

	checkData(ctx, outerPool.Slice(Range{Size: 11}), []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
}
