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

package resolve

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Memory resolves and returns the memory from the path p.
func Memory(ctx context.Context, p *path.Memory) (*service.MemoryInfo, error) {
	ctx = capture.Put(ctx, p.After.Commands.Capture)
	list, err := NCommands(ctx, p.After.Commands, p.After.Index+1)
	if err != nil {
		return nil, err
	}

	s := capture.NewState(ctx)
	for _, a := range list.Atoms[:p.After.Index] {
		a.Mutate(ctx, s, nil /* no builder, just mutate */)
	}

	pool, ok := s.Memory[memory.PoolID(p.Pool)]
	if !ok {
		return nil, fmt.Errorf("Pool %d not found", p)
	}

	r := memory.Range{Base: p.Address, Size: p.Size}

	var reads, writes memory.RangeList
	pool.OnRead = func(rng memory.Range) {
		if rng.Overlaps(r) {
			interval.Merge(&reads, rng.Window(r).Span(), false)
		}
	}
	pool.OnWrite = func(rng memory.Range) {
		if rng.Overlaps(r) {
			interval.Merge(&writes, rng.Window(r).Span(), false)
		}
	}
	list.Atoms[p.After.Index].Mutate(ctx, s, nil /* no builder, just mutate */)

	slice := pool.Slice(r)
	data := make([]byte, slice.Size())
	if err := slice.Get(ctx, 0, data); err != nil {
		return nil, err
	}

	observed := slice.ValidRanges()

	return &service.MemoryInfo{
		Data:     data,
		Reads:    service.NewMemoryRanges(reads),
		Writes:   service.NewMemoryRanges(writes),
		Observed: service.NewMemoryRanges(observed),
	}, nil
}
