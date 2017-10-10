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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service/path"
)

type CustomState struct{}

func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (width, height, index uint32, format *image.Format, err error) {
	return 0, 0, 0, nil, nil
}

func (API) Context(*api.GlobalState, uint64) api.Context { return nil }

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (s *State) Root(ctx context.Context, p *path.State) (path.Node, error) {
	return nil, nil
}

func (c *State) preMutate(ctx context.Context, s *api.GlobalState, cmd api.Cmd) error {
	return nil
}

func (i Remapped) remap(cmd api.Cmd, s *api.GlobalState) (interface{}, bool) {
	return i, true
}
