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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
)

type captureLog struct {
	file         *os.File
	header       *capture.Header
	initialState *capture.InitialState
	cmds         []api.Cmd
}

var (
	_ = Transformer(&captureLog{})
)

// NewCaptureLog returns a Transformer that will log all commands passed through it
// to the capture file at path.
func NewCaptureLog(ctx context.Context, sourceCapture *capture.Capture, path string) Transformer {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create replay capture file %v: %v", path, err)
		return nil
	}
	return &captureLog{
		file:         f,
		header:       sourceCapture.Header,
		initialState: sourceCapture.InitialState,
		cmds:         []api.Cmd{},
	}
}

func (t *captureLog) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) {
	// Don't write out the custom commands (e.g. replay.Custom)
	if cmd.API() != nil {
		t.cmds = append(t.cmds, cmd)
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *captureLog) Flush(ctx context.Context, out Writer) {

	capt, err := capture.New(ctx, "capturelog", t.header, t.initialState, t.cmds)
	if err != nil {
		log.E(ctx, "Failed to create replay storage capture: %v", err)
		return
	}
	c, err := capture.ResolveFromPath(ctx, capt)
	if err != nil {
		log.E(ctx, "Failed to resolve capture from path %v: %v", capt, err)
		return
	}
	if err := c.Export(ctx, t.file); err != nil {
		log.E(ctx, "Failed to write capture to file %v: %v", t.file, err)
		return
	}
	t.file.Close()
}
