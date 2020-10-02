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
	"fmt"
	"io"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/database"
)

// Resource returns a Data that wraps a resource stored in the database.
// resID is the identifier of the data and size is the size in bytes of the
// data.
func Resource(resID id.ID, size uint64) Data {
	return resource{resID, size}
}

type resource struct {
	resID id.ID
	size  uint64
}

func (r resource) Get(ctx context.Context, offset uint64, out []byte) error {
	data, err := r.getData(ctx)
	if err != nil {
		return err
	}
	copy(out, data[offset:])
	return nil
}

func (r resource) getData(ctx context.Context) ([]byte, error) {
	res, err := database.Resolve(ctx, r.resID)
	if err != nil {
		return nil, err
	}
	data := res.([]byte)
	if r.size != uint64(len(data)) {
		return nil, fmt.Errorf("Loaded resource is unexpected size. Expected 0x%x, got 0x%x for resource %v",
			r.size, len(data), r.resID)
	}
	return data, nil
}

func (r resource) ResourceID(ctx context.Context) (id.ID, error) {
	return r.resID, nil
}

func (r resource) Size() uint64 {
	return r.size
}

func (r resource) Slice(rng Range) Data {
	return newResourceSlice(r, rng)
}

func (r resource) ValidRanges() RangeList {
	return RangeList{Range{Size: r.Size()}}
}

func (r resource) Strlen(ctx context.Context) (int, error) {
	data, err := r.getData(ctx)
	if err != nil {
		return 0, err
	}
	for i, b := range data {
		if b == 0 {
			return i, nil
		}
	}
	return -1, nil
}

func (r resource) String() string {
	return fmt.Sprintf("Resource[%v]", r.resID)
}

func (r resource) NewReader(ctx context.Context) io.Reader {
	data, err := r.getData(ctx)
	if err != nil {
		return failedReader{err}
	}
	return bytes.NewReader(data)
}

func newResourceSlice(src resource, rng Range) Data {
	if uint64(rng.Last()) > src.Size() {
		panic(fmt.Errorf("Slice range %v out of bounds %v", rng, Range{Base: 0, Size: src.Size()}))
	}
	return resourceSlice{src, rng}
}

type resourceSlice struct {
	src resource
	rng Range
}

func (s resourceSlice) Get(ctx context.Context, offset uint64, out []byte) error {
	trim := min(s.rng.Size-offset, uint64(len(out)))
	return s.src.Get(ctx, s.rng.First()+offset, out[:trim])
}

func (r resourceSlice) NewReader(ctx context.Context) io.Reader {
	data, err := r.src.getData(ctx)
	if err != nil {
		return failedReader{err}
	}
	return bytes.NewReader(data[r.rng.First() : r.rng.Last()+1])
}

func (s resourceSlice) ResourceID(ctx context.Context) (id.ID, error) {
	src, offset, size := s.src.resID, s.rng.Base, s.rng.Size
	return database.Store(ctx, func(ctx context.Context) ([]byte, error) {
		res, err := database.Resolve(ctx, src)
		if err != nil {
			return nil, err
		}
		data := res.([]byte)
		out := make([]byte, size)
		copy(out, data[offset:])
		return out, nil
	})
}

func (s resourceSlice) Size() uint64 {
	return s.rng.Size
}

func (s resourceSlice) Slice(rng Range) Data {
	return newResourceSlice(s.src, Range{Base: s.rng.Base + rng.Base, Size: rng.Size})
}

func (s resourceSlice) ValidRanges() RangeList {
	return RangeList{Range{Size: s.rng.Size}}
}

func (s resourceSlice) Strlen(ctx context.Context) (int, error) {
	data := make([]byte, s.Size())
	if err := s.Get(ctx, 0, data); err != nil {
		return 0, err
	}
	for i, b := range data {
		if b == 0 {
			return i, nil
		}
	}
	return -1, nil
}

func (s resourceSlice) String() string {
	return fmt.Sprintf("%v[%v]", s.src, s.rng)
}

type failedReader struct {
	err error
}

func (fr failedReader) Read([]byte) (int, error) {
	return 0, fr.err
}
