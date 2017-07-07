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
	"context"
	"fmt"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/config"
)

// FileLog is an implementation of Transformer that will log all atoms to a text file.
type FileLog struct {
	file *os.File
}

func NewFileLog(ctx context.Context, name string) *FileLog {
	f, err := os.Create(name)
	if err != nil {
		log.E(ctx, "Failed to create replay log file %v: %v", name, err)
		return nil
	}
	return &FileLog{file: f}
}

func (t *FileLog) Transform(ctx context.Context, id atom.ID, cmd api.Cmd, out Writer) {
	if cmd.API() != nil {
		t.file.WriteString(fmt.Sprintf("%v: %v\n", id, cmd))
	} else {
		t.file.WriteString(fmt.Sprintf("%T\n", cmd))
	}
	if config.LogExtrasInTransforms {
		if extras := cmd.Extras(); extras != nil {
			for _, e := range extras.All() {
				if o, ok := e.(*api.CmdObservations); ok {
					if config.LogMemoryInExtras {
						t.file.WriteString(o.DataString(ctx))
					} else {
						t.file.WriteString(o.String())
					}
				} else {
					t.file.WriteString(fmt.Sprintf("[extra] %T: %v\n", e, e))
				}
			}
		}
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *FileLog) Flush(ctx context.Context, out Writer) {
	t.file.Close()
}
