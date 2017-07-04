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

package atom

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// FramebufferObservation is an Atom that holds a snapshot of the color-buffer
// of the bound framebuffer at the time of capture. These atoms can be used to
// verify that replay gave the same results as what was captured.
type FramebufferObservation struct {
	OriginalWidth  uint32 `param:"OriginalWidth"`  // Framebuffer width in pixels
	OriginalHeight uint32 `param:"OriginalHeight"` // Framebuffer height in pixels
	DataWidth      uint32 `param:"DataWidth"`      // Dimensions of downsampled data.
	DataHeight     uint32 `param:"DataHeight"`     // Dimensions of downsampled data.
	Data           []byte `param:"Data"`           // The RGBA color-buffer data
}

func (a *FramebufferObservation) String() string {
	return fmt.Sprintf("FramebufferObservation %dx%d", a.OriginalWidth, a.OriginalHeight)
}

// Atom compliance
func (FramebufferObservation) Thread() uint64   { return 0 }
func (FramebufferObservation) SetThread(uint64) {}
func (FramebufferObservation) AtomName() string { return "<FramebufferObservation>" }
func (FramebufferObservation) API() gfxapi.API  { return nil }
func (FramebufferObservation) AtomFlags() Flags { return 0 }
func (FramebufferObservation) Extras() *Extras  { return nil }
func (FramebufferObservation) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	return nil
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a *FramebufferObservation) (*atom_pb.FramebufferObservation, error) {
			return &atom_pb.FramebufferObservation{
				OriginalWidth:  a.OriginalWidth,
				OriginalHeight: a.OriginalHeight,
				DataWidth:      a.DataWidth,
				DataHeight:     a.DataHeight,
				Data:           a.Data,
			}, nil
		},
		func(ctx context.Context, a *atom_pb.FramebufferObservation) (*FramebufferObservation, error) {
			return &FramebufferObservation{
				OriginalWidth:  a.OriginalWidth,
				OriginalHeight: a.OriginalHeight,
				DataWidth:      a.DataWidth,
				DataHeight:     a.DataHeight,
				Data:           a.Data,
			}, nil
		},
	)
}
