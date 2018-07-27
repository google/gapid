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
	"unsafe"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/database"
)

type Native struct {
	arena     arena.Arena
	data      unsafe.Pointer
	size      uint64
	ownsAlloc bool
	id        id.ID
}

func NewNative(arena arena.Arena, size uint64) *Native {
	return &Native{
		arena:     arena,
		data:      arena.Allocate(int(size), 8),
		size:      size,
		ownsAlloc: true,
	}
}

func (r Native) Data() unsafe.Pointer {
	return r.data
}

func (r Native) Sli() []byte {
	return slice.Bytes(r.data, uint64(r.size))
}

func (r Native) Get(ctx context.Context, offset uint64, out []byte) error {
	copy(out, r.Sli()[offset:])
	return nil
}

func (r *Native) ResourceID(ctx context.Context) (id.ID, error) {
	if !r.id.IsValid() {
		i, err := database.Store(ctx, r.Sli())
		if err != nil {
			return id.ID{}, err
		}
		r.id = i
	}
	return r.id, nil
}

func (r Native) Size() uint64 {
	return r.size
}

func (r Native) Slice(rng Range) Data {
	return &Native{
		arena: r.arena,
		data:  unsafe.Pointer(uintptr(r.data) + uintptr(rng.Base)),
		size:  rng.Size,
	}
}

func (r Native) ValidRanges() RangeList {
	return RangeList{Range{Size: r.size}}
}

func (r Native) Strlen(ctx context.Context) (int, error) {
	for i, b := range r.Sli() {
		if b == 0 {
			return i, nil
		}
	}
	return -1, nil
}

func (r Native) String() string {
	return fmt.Sprintf("native[% x]", r.Sli())
}

func (r Native) NewReader(ctx context.Context) io.Reader {
	return bytes.NewReader(r.Sli())
}
