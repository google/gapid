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

package atom

import (
	"bytes"
	"context"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

// Data encodes and stores the value v to the database d, returning the
// memory range and new resource identifier. Data can be used to as a helper
// to AddRead and AddWrite methods on atoms.
func Data(ctx context.Context, l *device.MemoryLayout, at memory.Pointer, v ...interface{}) (memory.Range, id.ID) {
	buf := &bytes.Buffer{}
	e := memory.NewEncoder(endian.Writer(buf, l.GetEndian()), l)
	memory.Write(e, v)
	id, err := database.Store(ctx, buf.Bytes())
	if err != nil {
		panic(err)
	}
	return memory.Range{Base: at.Address(), Size: uint64(len(buf.Bytes()))}, id
}
