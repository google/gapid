// Copyright (C) 2019 Google Inc.
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

package replay

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

type VulkanHandleMappingItem struct {
	HandleType  string
	TraceValue  string
	ReplayValue string
}

type MappingExporter struct {
	mappings *[]VulkanHandleMappingItem
	thread   uint64
	path     string
}

func NewMappingExporter(ctx context.Context, mappings *[]VulkanHandleMappingItem) *MappingExporter {
	return &MappingExporter{
		mappings: mappings,
		thread:   0,
		path:     "",
	}
}

func NewMappingExporterWithPrint(ctx context.Context, path string) *MappingExporter {
	mapping := make([]VulkanHandleMappingItem, 0, 0)
	return &MappingExporter{
		mappings: &mapping,
		thread:   0,
		path:     path,
	}
}

func (m *MappingExporter) ExtractRemappings(ctx context.Context, s *api.GlobalState, b *builder.Builder, done func()) error {
	var ret error

	total := 0

	for k, v := range b.Remappings {
		typ := reflect.TypeOf(k)
		var size uint64
		if t, ok := k.(memory.SizedTy); ok {
			size = t.TypeSize(s.MemoryLayout)
		} else {
			size = uint64(typ.Size())
		}
		if size != 1 && size != 2 && size != 4 && size != 8 {
			// Ignore objects that are not handles
			continue
		}
		// Count the number of actual Posts we expect
		total += 1
		func(k interface{}, typ reflect.Type, size uint64) {
			b.Post(v, size, func(r binary.Reader, err error) {
				defer func() {
					total -= 1

					if total == 0 {
						done()
					}
				}()

				if ret != nil {
					return
				}
				if err != nil {
					ret = err
					return
				}
				val := binary.ReadUint(r, int32(size*8))
				err = r.Error()
				if err != nil {
					ret = err
					return
				}

				*m.mappings = append(*m.mappings, VulkanHandleMappingItem{HandleType: typ.Name(), TraceValue: fmt.Sprintf("%v", k), ReplayValue: fmt.Sprintf("%v", val)})
			})
		}(k, typ, size)
	}

	if total == 0 {
		done()
	}

	return ret
}

func (m *MappingExporter) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	m.thread = cmd.Thread()
	out.MutateAndWrite(ctx, id, cmd)
}

func printToFile(ctx context.Context, path string, mappings *[]VulkanHandleMappingItem) {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create mapping file %v: %v", path, err)
		return
	}

	mappingsToFile := make([]string, 0, 0)

	for _, m := range *mappings {
		mappingsToFile = append(mappingsToFile, fmt.Sprintf("%v(%v): %v\n", m.HandleType, m.TraceValue, m.ReplayValue))
	}

	sort.Strings(mappingsToFile)
	for _, l := range mappingsToFile {
		fmt.Fprint(f, l)
	}
	f.Close()
}

func (m *MappingExporter) Flush(ctx context.Context, out transform.Writer) {
	out.MutateAndWrite(ctx, api.CmdNoID, Custom{m.thread,
		func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			return m.ExtractRemappings(ctx, s, b, func() {
				if len(m.path) > 0 {
					printToFile(ctx, m.path, m.mappings)
				}
			})
		},
	})
}

func (m *MappingExporter) PreLoop(ctx context.Context, out transform.Writer)  {}
func (m *MappingExporter) PostLoop(ctx context.Context, out transform.Writer) {}

func (t *MappingExporter) BuffersCommands() bool {
	return false
}
