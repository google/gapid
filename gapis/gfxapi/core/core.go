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

package core

import (
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/gfxapi"
)

type CustomState struct{}

// GetFramebufferAttachmentInfo returns the width, height and format of the specified framebuffer attachment.
func (api) GetFramebufferAttachmentInfo(*gfxapi.State, uint64, gfxapi.FramebufferAttachment) (width, height uint32, index uint32, format *image.Format, err error) {
	return 0, 0, 0, nil, nil
}

// Context returns the active context for the given state.
func (api) Context(*gfxapi.State, uint64) gfxapi.Context {
	return nil
}
