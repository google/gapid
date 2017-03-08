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

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// FramebufferObservation is an Atom that holds a snapshot of the color-buffer
// of the bound framebuffer at the time of capture. These atoms can be used to
// verify that replay gave the same results as what was captured.
type FramebufferObservation struct {
	binary.Generate               `java:"disable"`
	OriginalWidth, OriginalHeight uint32 // Framebuffer dimensions in pixels
	DataWidth, DataHeight         uint32 // Dimensions of downsampled data.
	Data                          []byte // The RGBA color-buffer data
}

func (a *FramebufferObservation) String() string {
	return fmt.Sprintf("FramebufferObservation %dx%d", a.OriginalWidth, a.OriginalHeight)
}

// Atom compliance
func (a *FramebufferObservation) API() gfxapi.API  { return nil }
func (a *FramebufferObservation) AtomFlags() Flags { return 0 }
func (a *FramebufferObservation) Extras() *Extras  { return nil }
func (a *FramebufferObservation) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	return nil
}

func (a *FramebufferObservation) Convert(ctx context.Context, out atom_pb.Handler) error {
	return out(ctx, &atom_pb.FramebufferObservation{
		OriginalWidth:  a.OriginalWidth,
		OriginalHeight: a.OriginalHeight,
		DataWidth:      a.DataWidth,
		DataHeight:     a.DataHeight,
		Data:           a.Data,
	})
}

func FramebufferObservationFrom(from *atom_pb.FramebufferObservation) FramebufferObservation {
	return FramebufferObservation{
		OriginalWidth:  from.OriginalWidth,
		OriginalHeight: from.OriginalHeight,
		DataWidth:      from.DataWidth,
		DataHeight:     from.DataHeight,
		Data:           from.Data,
	}
}

func init() {
	s := (*FramebufferObservation)(nil).Class().Schema()
	s.Metadata = append(s.Metadata, &Metadata{
		API:              gfxapi.ID{},
		DisplayName:      "FramebufferObservation",
		DrawCall:         false,
		EndOfFrame:       false,
		DocumentationUrl: "[]",
	})
}
