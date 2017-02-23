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

package transform

import (
	"fmt"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/config"
)

// FileLog is an implementation of Transformer that will log all atoms to a text file.
type FileLog struct {
	file *os.File
}

func NewFileLog(ctx log.Context, name string) *FileLog {
	if f, err := os.Create(name); err != nil {
		ctx.Error().Logf("Failed to create replay log file %v: %v", name, err)
		return nil
	} else {
		return &FileLog{file: f}
	}
}

func (t *FileLog) Transform(ctx log.Context, id atom.ID, a atom.Atom, out Writer) {
	if a.API() != nil {
		if id != atom.NoID {
			t.file.WriteString(fmt.Sprintf("%v: %v\n", id, a))
		} else {
			t.file.WriteString(fmt.Sprintf("%v\n", a))
		}
	} else {
		t.file.WriteString(fmt.Sprintf("%T\n", a))
	}
	if config.LogExtrasInTransforms {
		if extras := a.Extras(); extras != nil {
			for _, e := range extras.All() {
				if o, ok := e.(*atom.Observations); ok {
					if config.LogMemoryInExtras {
						t.file.WriteString(o.DataString(ctx))
					} else {
						t.file.WriteString(o.String())
					}
				} else {
					t.file.WriteString(fmt.Sprintf("[extra] %s: %v\n", e.Class().Schema().Identity, e))
				}
			}
		}
	}
	out.MutateAndWrite(ctx, id, a)
}

func (t *FileLog) Flush(ctx log.Context, out Writer) {
	t.file.Close()
}
