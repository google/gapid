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

package capture

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay/builder"
)

// FBOName is the special name given to FBO commands.
const FBOName = "<FBO>"

// FBO is a deserialized FramebufferObservation.
type FBO struct {
	OriginalWidth  uint32 `param:"OriginalWidth"`  // Framebuffer width in pixels
	OriginalHeight uint32 `param:"OriginalHeight"` // Framebuffer height in pixels
	DataWidth      uint32 `param:"DataWidth"`      // Dimensions of downsampled data.
	DataHeight     uint32 `param:"DataHeight"`     // Dimensions of downsampled data.
	Data           []byte `param:"Data"`           // The RGBA color-buffer data
}

func (a *FBO) String() string {
	return fmt.Sprintf("FBO %dx%d", a.OriginalWidth, a.OriginalHeight)
}

// api.Cmd compliance
func (FBO) Thread() uint64         { return 0 }
func (FBO) SetThread(uint64)       {}
func (FBO) CmdName() string        { return FBOName }
func (FBO) API() api.API           { return nil }
func (FBO) CmdFlags() api.CmdFlags { return 0 }
func (FBO) Extras() *api.CmdExtras { return nil }
func (FBO) Mutate(ctx context.Context, s *api.State, b *builder.Builder) error {
	return nil
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a *FBO) (*FramebufferObservation, error) {
			return &FramebufferObservation{
				OriginalWidth:  a.OriginalWidth,
				OriginalHeight: a.OriginalHeight,
				DataWidth:      a.DataWidth,
				DataHeight:     a.DataHeight,
				Data:           a.Data,
			}, nil
		},
		func(ctx context.Context, a *FramebufferObservation) (*FBO, error) {
			return &FBO{
				OriginalWidth:  a.OriginalWidth,
				OriginalHeight: a.OriginalHeight,
				DataWidth:      a.DataWidth,
				DataHeight:     a.DataHeight,
				Data:           a.Data,
			}, nil
		},
	)
}
