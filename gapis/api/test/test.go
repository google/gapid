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

package test

import (
	"context"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service/path"
)

type customState struct{}

func (customState) init(*State) {}

// RebuildState is a no-op to conform to the api.API interface.
func (API) RebuildState(ctx context.Context, s *api.GlobalState) ([]api.Cmd, interval.U64RangeList) {
	return nil, nil
}

// GetFramegraph is a no-op to conform to the api.API interface.
func (API) GetFramegraph(ctx context.Context, p *path.Capture) (*api.Framegraph, error) {
	return nil, nil
}

func (API) GetFramebufferAttachmentInfos(
	ctx context.Context,
	state *api.GlobalState) ([]api.FramebufferAttachmentInfo, error) {
	return []api.FramebufferAttachmentInfo{}, nil
}

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (s *State) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	return p, nil
}

// SetupInitialState sanitizes deserialized state to make it valid.
// It can fill in any derived data which we choose not to serialize,
// or it can apply backward-compatibility fixes for older traces.
func (*State) SetupInitialState(ctx context.Context, state *api.GlobalState) {}

func (i Remapped) remap(cmd api.Cmd, s *api.GlobalState) (interface{}, bool) {
	return i, true
}
