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

package testutils

import (
	"context"
	"unsafe"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapis/api"
)

type API struct{ Idx int }

func (API) Definition() api.Definition                      { return api.Definition{} }
func (API) State(a arena.Arena, p unsafe.Pointer) api.State { return nil }
func (API) Name() string                                    { return "test api" }
func (a API) Index() uint8                                  { return uint8(a.Idx) }
func (API) ID() api.ID                                      { return api.ID{} }
func (API) ConstantSets() *constset.Pack                    { return nil }
func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (info api.FramebufferAttachmentInfo, err error) {

	return api.FramebufferAttachmentInfo{}, nil
}
func (API) Context(ctx context.Context, state *api.GlobalState, thread uint64) api.Context { return nil }
func (API) CreateCmd(a arena.Arena, name string) api.Cmd                                   { return nil }
func (API) RebuildState(ctx context.Context, s *api.GlobalState) ([]api.Cmd, interval.U64RangeList) {
	return nil, nil
}
