// Copyright (C) 2018 Google Inc.
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

type mappingPrinter struct {
	file   *os.File
	thread uint64
}

func NewMappingPrinter(ctx context.Context, path string) transform.Transformer {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create mapping file %v: %v", path, err)
		return nil
	}
	return &mappingPrinter{
		file:   f,
		thread: 0,
	}
}

func (m *mappingPrinter) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	m.thread = cmd.Thread()
	out.MutateAndWrite(ctx, id, cmd)
}

func (m *mappingPrinter) Flush(ctx context.Context, out transform.Writer) {
	out.MutateAndWrite(ctx, api.CmdNoID, Custom{m.thread,
		func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			var ret error

			total := 0
			output := make([]string, 0, total)

			done := func() {
				sort.Strings(output)
				for _, l := range output {
					fmt.Fprint(m.file, l)
				}
				m.file.Close()
			}

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
							// When the last one is done, output the file
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
						output = append(output,
							fmt.Sprintf("%v(%v): %v\n", typ.Name(), k, val))
					})
				}(k, typ, size)
			}
			if total == 0 {
				done()
			}
			return ret
		},
	})
}

func (m *mappingPrinter) PreLoop(ctx context.Context, out transform.Writer)  {}
func (m *mappingPrinter) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *mappingPrinter) BuffersCommands() bool                              { return false }
