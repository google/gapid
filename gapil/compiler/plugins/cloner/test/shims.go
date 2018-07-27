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

package test

import (
	"context"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service/path"
)

type customState struct{}

func (customState) init(*State) {}

// ConstantSets is a no-op function so that API conforms to the api.API
// interface.
func (API) ConstantSets() *constset.Pack { return nil }

// GetFramebufferAttachmentInfo is a no-op function so that API conforms to the
// api.API interface.
func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (api.FramebufferAttachmentInfo, error) {
	return api.FramebufferAttachmentInfo{}, nil
}

// Context is a no-op function so that API conforms to the api.API interface.
func (API) Context(ctx context.Context, state *api.GlobalState, thread uint64) api.Context {
	return nil
}

// RebuildState is a no-op function.
func (API) RebuildState(ctx context.Context, s *api.GlobalState) ([]api.Cmd, interval.U64RangeList) {
	return nil, nil
}

// Root is a no-op function so that State conforms to the api.State interface.
func (State) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	return nil, nil
}

// SetupInitialState is a no-op function so that State conforms to the api.State
// interface.
func (State) SetupInitialState(ctx context.Context) {}

// Mutate is a no-op function.
func (*Foo) Mutate(context.Context, api.CmdID, *api.GlobalState, builder.Builder, api.StateWatcher) error {
	return nil
}
