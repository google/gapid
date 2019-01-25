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

package resolve

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Stats resolves and returns the stats list from the path p.
func Stats(ctx context.Context, p *path.Stats, r *path.ResolveConfig) (*service.Stats, error) {
	stats := &service.Stats{}
	if p.DrawCall {
		err := drawCallStats(ctx, p.Capture, stats, r)
		if err != nil {
			return nil, err
		}
	}
	c, err := capture.ResolveGraphicsFromPath(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	stats.TraceStart = c.Header.StartTime

	return stats, nil
}

func drawCallStats(ctx context.Context, capt *path.Capture, stats *service.Stats, r *path.ResolveConfig) error {
	d, err := SyncData(ctx, capt)
	if err != nil {
		return err
	}
	cmds, err := Cmds(ctx, capt)
	if err != nil {
		return err
	}

	st, err := capture.NewState(ctx)
	if err != nil {
		return err
	}
	flags := make([]api.CmdFlags, len(cmds))

	// Get the present calls
	events, err := Events(ctx, &path.Events{
		Capture:     capt,
		LastInFrame: true,
	}, r)
	if err != nil {
		return err
	}

	drawsPerFrame := make([]uint64, len(events.List))
	drawsSinceLastFrame := uint64(0)

	processed := map[sync.SyncNodeIdx]struct{}{}

	var process func(pt sync.SyncNodeIdx) error
	process = func(pt sync.SyncNodeIdx) error {
		if _, ok := processed[pt]; ok {
			return nil
		}
		processed[pt] = struct{}{}

		ptObj := d.SyncNodes[pt]
		if cmdIdx, ok := ptObj.(sync.CmdNode); ok {
			idx := cmdIdx.Idx
			cmd, err := Cmd(ctx, &path.Command{
				Capture: capt,
				Indices: []uint64(idx),
			}, r)
			if err != nil {
				return err
			}
			// If the command has subcommands, ignore it (vkQueueSubmit or similar)
			if _, ok := d.SubcommandReferences[api.CmdID(idx[0])]; len(idx) > 1 || !ok {
				var cmdflags api.CmdFlags
				if len(idx) == 1 {
					cmdflags = flags[idx[0]]
				} else {
					// NOTE: For subcommands its not clear
					// what the "correct" state to present
					// to CmdFlags is.  Since Vulkan
					// currently does not use the state,
					// pass nil here instead of a
					// potentially "incorrect" state.
					cmdflags = cmd.CmdFlags(ctx, api.CmdID(idx[0]), nil)
				}
				if (len(idx) == 1 && cmdflags.IsDrawCall()) ||
					(len(idx) > 1 && cmdflags.IsExecutedDraw()) {

					drawsSinceLastFrame += 1
				}
			}
		}

		deps, ok := d.SyncDependencies[pt]
		if ok {
			for _, dep := range deps {
				err := process(dep)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	processCmd := func(idx uint64) error {
		cmd := cmds[idx]
		err := cmd.Mutate(ctx, api.CmdID(idx), st, nil, nil)
		if err != nil {
			return err
		}
		flags[idx] = cmd.CmdFlags(ctx, api.CmdID(idx), st)

		// If the command wasn't included in the dependency graph,
		// assume its a synchronous command (e.g. glDraw)
		if _, ok := d.CmdSyncNodes[api.CmdID(idx)]; !ok {
			if flags[idx].IsDrawCall() {
				drawsSinceLastFrame += 1
			}
		}
		return nil
	}

	cmdIdx := uint64(0)
	for i, event := range events.List {
		limitIdx := event.Command.Indices[0]
		// Add any draws in the final unfinished frame to the last frame
		if i == len(events.List)-1 {
			limitIdx = uint64(len(cmds)) - 1
		}
		for cmdIdx <= limitIdx {
			err := processCmd(cmdIdx)
			if err != nil {
				return err
			}
			cmdIdx += 1
		}
		id := api.CmdID(event.Command.Indices[0])
		cmd := cmds[id]
		// If the frame boundary was on a synchronized api, process its dependencies
		if _, ok := cmd.API().(sync.SynchronizedAPI); ok {
			pt := d.CmdSyncNodes[id]
			err := process(pt)
			if err != nil {
				return err
			}
		}
		drawsPerFrame[i] = drawsSinceLastFrame
		drawsSinceLastFrame = 0
	}

	stats.DrawCalls = drawsPerFrame
	return nil
}
