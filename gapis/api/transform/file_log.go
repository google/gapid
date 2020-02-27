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
	"github.com/google/gapid/gapis/config"
)

type fileLog struct {
	file *os.File
}

// NewFileLog returns a Transformer that will log all commands passed through it
// to the text file at path.
func NewFileLog(ctx context.Context, path string) Transformer {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create replay log file %v: %v", path, err)
		return nil
	}
	return &fileLog{file: f}
}

func (t *fileLog) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) error {
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
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *fileLog) Flush(ctx context.Context, out Writer) error {
	t.file.Close()
	return nil
}

func (t *fileLog) PreLoop(ctx context.Context, output Writer) {
	// Bypass the preloop to the next
	output.NotifyPreLoop(ctx)
}
func (t *fileLog) PostLoop(ctx context.Context, output Writer) {
	// Bypass the PostLoop to the next
	output.NotifyPostLoop(ctx)
}

func (t *fileLog) BuffersCommands() bool { return false }
