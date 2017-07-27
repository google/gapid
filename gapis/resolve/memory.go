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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Memory resolves and returns the memory from the path p.
func Memory(ctx context.Context, p *path.Memory) (*service.Memory, error) {
	ctx = capture.Put(ctx, path.FindCapture(p))

	cmdIdx := p.After.Indices[0]

	allCmds, err := Cmds(ctx, path.FindCapture(p))
	if err != nil {
		return nil, err
	}
	cmds, err := sync.MutationCmdsFor(ctx, path.FindCapture(p), allCmds, api.CmdID(cmdIdx), p.After.Indices[1:])
	if err != nil {
		return nil, err
	}

	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, s, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	pool, ok := s.Memory[memory.PoolID(p.Pool)]
	if !ok {
		return nil, fmt.Errorf("Pool %v not found", p)
	}

	r := memory.Range{Base: p.Address, Size: p.Size}

	var reads, writes, observed memory.RangeList
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
	api.MutateCmds(ctx, s, nil, cmds[cmdIdx])

	slice := pool.Slice(r)

	if !p.ExcludeObserved {
		observed = slice.ValidRanges()
	}

	var data []byte
	if !p.ExcludeData && slice.Size() > 0 {
		data = make([]byte, slice.Size())
		if err := slice.Get(ctx, 0, data); err != nil {
			return nil, err
		}
	}

	return &service.Memory{
		Data:     data,
		Reads:    service.NewMemoryRanges(reads),
		Writes:   service.NewMemoryRanges(writes),
		Observed: service.NewMemoryRanges(observed),
	}, nil
}
