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
	"fmt"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform2"
	"github.com/google/gapid/gapis/config"
)

var _ transform2.Transform = &fileLogTransform{}

type fileLogTransform struct {
	file *os.File
}

// NewFileLog returns a Transform that will log all commands passed through it
// to the text file at path.
func newFileLog(ctx context.Context, path string) *fileLogTransform {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create replay log file %v: %v", path, err)
		return nil
	}
	return &fileLogTransform{file: f}
}

func (fileLog *fileLogTransform) RequiresAccurateState() bool {
	return false
}

func (fileLog *fileLogTransform) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (fileLog *fileLogTransform) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}

func (fileLog *fileLogTransform) EndTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	fileLog.file.Close()
	return inputCommands, nil
}

func (fileLog *fileLogTransform) TransformCommand(ctx context.Context, id api.CmdID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if len(inputCommands) == 0 {
		return inputCommands, nil
	}

	for _, cmd := range inputCommands {
		if cmd.API() != nil {
			fileLog.file.WriteString(fmt.Sprintf("%v: %v\n", id, cmd))
		} else {
			fileLog.file.WriteString(fmt.Sprintf("%T\n", cmd))
		}

		if config.LogExtrasInTransforms {
			if extras := cmd.Extras(); extras != nil {
				for _, e := range extras.All() {
					if o, ok := e.(*api.CmdObservations); ok {
						if config.LogMemoryInExtras {
							fileLog.file.WriteString(o.DataString(ctx))
						} else {
							fileLog.file.WriteString(o.String())
						}
					} else {
						fileLog.file.WriteString(fmt.Sprintf("[extra] %T: %v\n", e, e))
					}
				}
			}
		}
	}

	return inputCommands, nil
}
