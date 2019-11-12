// Copyright (C) 2019 Google Inc.
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

package replay

import (
	"context"

	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay/builder"
)

type WaitForFence struct {
	transformCallback func(ctx context.Context, p *gapir.FenceReadyRequest)
	flushCallback     func(ctx context.Context, p *gapir.FenceReadyRequest)
	shouldWait        func(ctx context.Context, id api.CmdID, cmd api.Cmd) bool
}

func (t *WaitForFence) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	if t.transformCallback != nil && t.shouldWait != nil && t.shouldWait(ctx, id, cmd) {
		t.AddTransformWait(ctx, id, out)
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *WaitForFence) Flush(ctx context.Context, out transform.Writer) {
	if t.flushCallback != nil {
		t.AddFlushWait(ctx, out)
	}
}

func (t *WaitForFence) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *WaitForFence) PostLoop(ctx context.Context, out transform.Writer) {}

func (t *WaitForFence) AddTransformWait(ctx context.Context, id api.CmdID, out transform.Writer) {
	out.MutateAndWrite(ctx, id, Custom{T: 0, F: func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		fenceID := uint32(id)
		b.Wait(fenceID)
		tcb := func(p *gapir.FenceReadyRequest) {
			t.transformCallback(ctx, p)
		}
		return b.RegisterFenceReadyRequestCallback(fenceID, tcb)
	}})
}

func (t *WaitForFence) AddFlushWait(ctx context.Context, out transform.Writer) {
	out.MutateAndWrite(ctx, api.CmdNoID, Custom{T: 0, F: func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		fenceID := uint32(0x3ffffff)
		b.Wait(fenceID)
		fcb := func(p *gapir.FenceReadyRequest) {
			t.flushCallback(ctx, p)
		}
		return b.RegisterFenceReadyRequestCallback(fenceID, fcb)
	}})
}
