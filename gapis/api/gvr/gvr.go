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

package gvr

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service/path"
)

func (c *State) preMutate(ctx context.Context, s *api.State, cmd api.Cmd) error {
	return nil
}

type CustomState struct{}

// GetFramebufferAttachmentInfo returns the width, height and format of the specified framebuffer attachment.
func (API) GetFramebufferAttachmentInfo(state *api.State, thread uint64, attachment api.FramebufferAttachment) (width, height, index uint32, format *image.Format, err error) {
	return 0, 0, 0, nil, fmt.Errorf("GVR does not support framebuffers")
}

// Context returns the active context for the given state and thread.
func (API) Context(s *api.State, thread uint64) api.Context {
	return nil
}

// Mesh implements the api.MeshProvider interface.
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh) (*api.Mesh, error) {
	return nil, nil
}
