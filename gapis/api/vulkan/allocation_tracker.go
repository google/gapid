// Copyright (C) 2020 Google Inc.
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

package vulkan

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

type allocationTracker struct {
	allocations []*api.AllocResult
	state       *api.GlobalState
}

func NewAllocationTracker(inputState *api.GlobalState) *allocationTracker {
	return &allocationTracker{
		allocations: []*api.AllocResult{},
		state:       inputState,
	}
}

// TODO: If allocation fails, rather than panicking, we should check the result and return error.
func (handler *allocationTracker) AllocOrPanic(ctx context.Context, size uint64) api.AllocResult {
	res := handler.state.AllocOrPanic(ctx, size)
	handler.allocations = append(handler.allocations, &res)
	return res
}

func (handler *allocationTracker) AllocDataOrPanic(ctx context.Context, v ...interface{}) api.AllocResult {
	res := handler.state.AllocDataOrPanic(ctx, v...)
	handler.allocations = append(handler.allocations, &res)
	return res
}

func (handler *allocationTracker) Alloc(ctx context.Context, size uint64) (api.AllocResult, error) {
	res, err := handler.state.Alloc(ctx, size)
	if err != nil {
		handler.allocations = append(handler.allocations, &res)
	}

	return res, err
}

func (handler *allocationTracker) FreeAllocations() {
	for _, allocation := range handler.allocations {
		allocation.Free()
	}

	handler.allocations = []*api.AllocResult{}
}
