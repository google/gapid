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

package gfxapi_test_import

import (
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

type Imported struct {
	binary.Generate
	Value uint32
}

func ImportedSize(l *device.MemoryLayout) uint64 {
	return uint64(4)
}

func ImportedAlignment(l *device.MemoryLayout) uint64 {
	return uint64(4)
}

func ImportedDecodeRaw(l *device.MemoryLayout, d pod.Reader, o *Imported) {
	o.Value = d.Uint32()
}

func ImportedEncodeRaw(l *device.MemoryLayout, e pod.Writer, o *Imported) {
	e.Uint32(o.Value)
}

func (Imported) Init() {}
func (Imported) value(ϟb *builder.Builder, ϟa atom.Atom, l *device.MemoryLayout) value.Value {
	return nil
}
