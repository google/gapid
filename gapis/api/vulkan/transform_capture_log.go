// Copyright (C) 2020 Google Inc.
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

package vulkan

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
}

func newCaptureLog(ctx context.Context, sourceCapture *capture.GraphicsCapture, path string) *captureLog {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create replay capture file %v: %v", path, err)
		return nil
	}
	return &captureLog{
		file:   f,
		header: sourceCapture.Header,
		cmds:   []api.Cmd{},
	}
}

func (logTransform *captureLog) RequiresAccurateState() bool {
	return false
}

func (logTransform *captureLog) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (logTransform *captureLog) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}

func (logTransform *captureLog) TransformCommand(ctx context.Context, id api.CmdID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	for _, cmd := range inputCommands {
		if cmd.API() != nil {
			logTransform.cmds = append(logTransform.cmds, cmd)
		}
	}

	return inputCommands, nil
}

func (logTransform *captureLog) EndTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	a := arena.New()
	defer a.Dispose()

	for idx := range logTransform.cmds {
		cmd := logTransform.cmds[idx].Clone(a)
		logTransform.cmds[idx] = cmd
	}

	c, err := capture.NewGraphicsCapture(ctx, a, "capturelog", logTransform.header, nil, logTransform.cmds)
	if err != nil {
		log.E(ctx, "Failed to create replay storage capture: %v", err)
		return inputCommands, err
	}
	if err := c.Export(ctx, logTransform.file); err != nil {
		log.E(ctx, "Failed to write capture to file %v: %v", logTransform.file, err)
		return inputCommands, err
	}
	logTransform.file.Close()

	return inputCommands, nil
}
