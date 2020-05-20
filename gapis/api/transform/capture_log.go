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

package transform

import (
	"context"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
)

type captureLog struct {
	file   *os.File
	header *capture.Header
	cmds   []api.Cmd
	ids    map[api.CmdID]api.CmdID
}

var (
	_ = Transformer(&captureLog{})
)

// NewCaptureLog returns a Transformer that will log all commands passed through it
// to the capture file at path.
func NewCaptureLog(ctx context.Context, sourceCapture *capture.GraphicsCapture, path string) Transformer {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create replay capture file %v: %v", path, err)
		return nil
	}
	return &captureLog{
		file:   f,
		header: sourceCapture.Header,
		cmds:   []api.Cmd{},
		ids:    map[api.CmdID]api.CmdID{},
	}
}

func (t *captureLog) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) error {
	// Don't write out the custom commands (e.g. replay.Custom)
	if cmd.API() != nil {
		if id.IsReal() {
			t.ids[id] = api.CmdID(len(t.cmds))
		}
		t.cmds = append(t.cmds, cmd)
	}
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *captureLog) Flush(ctx context.Context, out Writer) error {
	a := arena.New()
	defer a.Dispose()

	for idx := range t.cmds {
		cmd := t.cmds[idx].Clone(a)
		t.cmds[idx] = cmd
	}

	c, err := capture.NewGraphicsCapture(ctx, a, "capturelog", t.header, nil, t.cmds)
	if err != nil {
		log.E(ctx, "Failed to create replay storage capture: %v", err)
		return err
	}
	if err := c.Export(ctx, t.file); err != nil {
		log.E(ctx, "Failed to write capture to file %v: %v", t.file, err)
		return err
	}
	t.file.Close()
	return nil
}

func (t *captureLog) PreLoop(ctx context.Context, output Writer)  {}
func (t *captureLog) PostLoop(ctx context.Context, output Writer) {}
func (t *captureLog) BuffersCommands() bool                       { return false }
