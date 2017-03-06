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

package gles

import (
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service/path"
)

func GetContext(s *gfxapi.State) *Context {
	return GetState(s).getContext()
}

// GetFramebufferAttachmentInfo returns the width, height and format of the specified framebuffer attachment.
func (api) GetFramebufferAttachmentInfo(state *gfxapi.State, attachment gfxapi.FramebufferAttachment) (width, height uint32, format *image.Format, err error) {
	w, h, ifmt, err := GetState(state).getFramebufferAttachmentInfo(attachment)
	var nilImgfmt imgfmt
	if ifmt == nilImgfmt {
		return 0, 0, nil, fmt.Errorf("No format set")
	}
	if err != nil {
		return 0, 0, nil, err
	}
	f, err := ifmt.asImage()
	return w, h, f, err
}

// Context returns the active context for the given state.
func (api) Context(s *gfxapi.State) gfxapi.Context {
	if c := GetContext(s); c != nil {
		return c
	}
	return nil
}

// Mesh implements the gfxapi.MeshProvider interface.
func (api) Mesh(ctx log.Context, o interface{}, p *path.Mesh) (*gfxapi.Mesh, error) {
	if dc, ok := o.(drawCall); ok {
		return drawCallMesh(ctx, dc, p)
	}
	return nil, nil
}
