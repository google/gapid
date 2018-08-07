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

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
)

type blob struct {
	data []byte
	id   id.ID
}

func (r *blob) Get(ctx context.Context, offset uint64, out []byte) error {
	copy(out, r.data[offset:])
	return nil
}

func (r *blob) ResourceID(ctx context.Context) (id.ID, error) {
	if !r.id.IsValid() {
		ident, err := database.Store(ctx, r.data)
		if err != nil {
			return id.ID{}, err
		}
		r.id = ident
	}
	return r.id, nil
}

func (r *blob) Size() uint64 {
	return uint64(len(r.data))
}

func (r *blob) Slice(rng Range) Data {
	return Blob(r.data[rng.First() : rng.Last()+1])
}

func (r *blob) ValidRanges() RangeList {
	return RangeList{Range{Size: r.Size()}}
}

func (r *blob) Strlen(ctx context.Context) (int, error) {
	for i, b := range r.data {
		if b == 0 {
			return i, nil
		}
	}
	return -1, nil
}

func (r *blob) String() string {
	return fmt.Sprintf("Blob[% x]", r.data)
}

func (r *blob) NewReader(ctx context.Context) io.Reader {
	return bytes.NewReader(r.data)
}

// Blob returns a read-only Slice that wraps data.
func Blob(data []byte) Data {
	return &blob{data: data}
}

// NewData returns a read-only Slice that contains the encoding of data.
func NewData(layout *device.MemoryLayout, data ...interface{}) Data {
	buf := &bytes.Buffer{}
	e := NewEncoder(endian.Writer(buf, layout.GetEndian()), layout)
	for _, d := range data {
		Write(e, d)
	}
	return Blob(buf.Bytes())
}
