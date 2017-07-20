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
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
)

type CustomState struct{}

func (API) GetFramebufferAttachmentInfo(*api.State, uint64, api.FramebufferAttachment) (uint32, uint32, uint32, *image.Format, error) {
	return 0, 0, 0, nil, nil
}

func (API) Context(*api.State, uint64) api.Context { return nil }

func (i Remapped) remap(cmd api.Cmd, s *api.State) (interface{}, bool) {
	return i, true
}
