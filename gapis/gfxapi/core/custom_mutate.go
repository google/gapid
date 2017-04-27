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
	"context"

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/gles"
	"github.com/google/gapid/gapis/replay/builder"
)

var _ atom.Atom = &SwitchThread{}

func (a *SwitchThread) Mutate(ctx context.Context, gs *gfxapi.State, b *builder.Builder) error {
	err := a.mutate(ctx, gs, nil)
	if b == nil || err != nil {
		return err
	}
	return gles.OnSwitchThread(ctx, gs, b)
}
