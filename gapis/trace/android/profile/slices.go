// Copyright (C) 2021 Google Inc.
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

package profile

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_service "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

const (
	slicesQuery = "" +
		"SELECT s.context_id, s.render_target, s.frame_id, s.submission_id, s.hw_queue_id, s.command_buffer, s.render_pass, s.ts, s.dur, s.id, s.name, depth, arg_set_id, track_id, t.name " +
		"FROM gpu_track t LEFT JOIN gpu_slice s " +
		"ON s.track_id = t.id WHERE t.scope = 'gpu_render_stage' ORDER BY s.ts"
	argsQueryFmt = "" +
		"SELECT key, string_value FROM args WHERE args.arg_set_id = %d"
)

type SliceData struct {
	Contexts       []int64
	RenderTargets  []int64
	Frames         []int64
	Submissions    []int64
	HardwareQueues []int64
	CommandBuffers []int64
	RenderPasses   []int64
	Timestamps     []int64
	Durations      []int64
	SliceIds       []int64
	Names          []string
	Depths         []int64
	ArgSets        []int64
	Tracks         []int64
	TrackNames     []string
	GroupIds       []int32 // To be filled in by caller.

	groups groupTree
}

func ExtractSliceData(ctx context.Context, processor *perfetto.Processor) (*SliceData, error) {
	slicesQueryResult, err := processor.Query(slicesQuery)
	if err != nil {
		return nil, log.Errf(ctx, err, "SQL query failed: %v", slicesQuery)
	}

	slicesColumns := slicesQueryResult.GetColumns()
	data := &SliceData{
		Contexts:       slicesColumns[0].GetLongValues(),
		RenderTargets:  slicesColumns[1].GetLongValues(),
		Frames:         slicesColumns[2].GetLongValues(),
		Submissions:    slicesColumns[3].GetLongValues(),
		HardwareQueues: slicesColumns[4].GetLongValues(),
		CommandBuffers: slicesColumns[5].GetLongValues(),
		RenderPasses:   slicesColumns[6].GetLongValues(),
		Timestamps:     slicesColumns[7].GetLongValues(),
		Durations:      slicesColumns[8].GetLongValues(),
		SliceIds:       slicesColumns[9].GetLongValues(),
		Names:          slicesColumns[10].GetStringValues(),
		Depths:         slicesColumns[11].GetLongValues(),
		ArgSets:        slicesColumns[12].GetLongValues(),
		Tracks:         slicesColumns[13].GetLongValues(),
		TrackNames:     slicesColumns[14].GetStringValues(),
		GroupIds:       make([]int32, slicesQueryResult.GetNumRecords()),
		groups:         groupTree{1, groupTreeNode{id: 0, name: "root"}},
	}

	return data, nil
}

func (d *SliceData) MapIdentifiers(ctx context.Context, handleMapping map[uint64][]service.VulkanHandleMappingItem) {
	ExtractTraceHandles(ctx, d.Contexts, "VkDevice", handleMapping)
	ExtractTraceHandles(ctx, d.RenderTargets, "VkFramebuffer", handleMapping)
	ExtractTraceHandles(ctx, d.CommandBuffers, "VkCommandBuffer", handleMapping)
	ExtractTraceHandles(ctx, d.RenderPasses, "VkRenderPass", handleMapping)
}

func (d *SliceData) CreateOrGetGroup(name string, link sync.SubCmdRange) int32 {
	return d.groups.createOrGetGroup(name, link)
}

func (d *SliceData) ToService(ctx context.Context, processor *perfetto.Processor, capture *path.Capture) *service.ProfilingData_GpuSlices {
	extraCache := newExtras(processor)

	tracks := map[int64]*service.ProfilingData_GpuSlices_Track{}
	slices := make([]*service.ProfilingData_GpuSlices_Slice, len(d.Contexts))

	for i := range d.Contexts {
		extras := d.fillInExtras(i, extraCache.get(ctx, d.ArgSets[i]))

		slices[i] = &service.ProfilingData_GpuSlices_Slice{
			Ts:      uint64(d.Timestamps[i]),
			Dur:     uint64(d.Durations[i]),
			Id:      uint64(d.SliceIds[i]),
			Label:   d.Names[i],
			Depth:   int32(d.Depths[i]),
			Extras:  extras,
			TrackId: int32(d.Tracks[i]),
			GroupId: d.GroupIds[i],
		}

		if _, ok := tracks[d.Tracks[i]]; !ok {
			tracks[d.Tracks[i]] = &service.ProfilingData_GpuSlices_Track{
				Id:   int32(d.Tracks[i]),
				Name: d.TrackNames[i],
			}
		}
	}

	return &service.ProfilingData_GpuSlices{
		Slices: slices,
		Tracks: flattenTracks(tracks),
		Groups: d.groups.flatten(nil, capture, 0),
	}
}

func (d *SliceData) fillInExtras(idx int, extras []*service.ProfilingData_GpuSlices_Slice_Extra) []*service.ProfilingData_GpuSlices_Slice_Extra {
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "contextId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.Contexts[idx])},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "renderTarget",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.RenderTargets[idx])},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "commandBuffer",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.CommandBuffers[idx])},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "renderPass",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.RenderPasses[idx])},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "frameId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.Frames[idx])},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "submissionId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.Submissions[idx])},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "hwQueueId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(d.HardwareQueues[idx])},
	})
	return extras
}

func flattenTracks(tracks map[int64]*service.ProfilingData_GpuSlices_Track) []*service.ProfilingData_GpuSlices_Track {
	flat := make([]*service.ProfilingData_GpuSlices_Track, 0, len(tracks))
	for _, v := range tracks {
		flat = append(flat, v)
	}
	return flat
}

type extras struct {
	processor *perfetto.Processor
	cache     map[int64]*perfetto_service.QueryResult
}

func newExtras(processor *perfetto.Processor) *extras {
	return &extras{processor, map[int64]*perfetto_service.QueryResult{}}
}

func (e *extras) get(ctx context.Context, argSet int64) []*service.ProfilingData_GpuSlices_Slice_Extra {
	argsQueryResult, ok := e.cache[argSet]
	if !ok {
		var err error
		argsQuery := fmt.Sprintf(argsQueryFmt, argSet)
		argsQueryResult, err = e.processor.Query(argsQuery)
		if err != nil {
			log.W(ctx, "SQL query failed: %v", argsQuery)
		}
		e.cache[argSet] = argsQueryResult
	}

	argsColumns := argsQueryResult.GetColumns()
	numArgsRows := argsQueryResult.GetNumRecords()
	var extras []*service.ProfilingData_GpuSlices_Slice_Extra
	for j := uint64(0); j < numArgsRows; j++ {
		keys := argsColumns[0].GetStringValues()
		values := argsColumns[1].GetStringValues()
		extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
			Name:  keys[j],
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_StringValue{StringValue: values[j]},
		})
	}
	return extras
}

type groupTreeNode struct {
	id   int32
	name string
	link sync.SubCmdRange

	children []groupTreeNode // sorted by child.link
}

type groupTree struct {
	nextID int32
	groupTreeNode
}

// TODO: this function makes some assumptions about command/sub command IDs:
// 1. we only get groups for command buffers, renderpasses and draw calls.
// 2. no overlaps.
// 3. the sub command ids are [cmdId, submission, cmdbuff, cmd].
// All these assumptions currently hold and are also made in other parts of the
// code in some way. The assumptions will need to be codified as part of the
// command/sub-command refactor that is already planned.
func (t *groupTree) createOrGetGroup(name string, link sync.SubCmdRange) int32 {
	submit, ok := t.findOrInsert(t.nextID, "submit", sync.SubCmdRange{From: link.From[:1], To: link.From[:1]})
	if !ok {
		t.nextID++
	}

	cmdBuf, ok := submit.findOrInsert(t.nextID, "cmdbuf", sync.SubCmdRange{From: link.From[:3], To: link.From[:3]})
	if !ok {
		t.nextID++
	}

	if len(link.From) == 3 {
		// We've found our command buffer. Let's update the name in case we created it with "cmdbuf".
		cmdBuf.name = name
		return cmdBuf.id
	}

	rp, ok := cmdBuf.findOrInsert(t.nextID, name, link)
	if !ok {
		t.nextID++
	}

	return rp.id
}

func (n *groupTreeNode) findOrInsert(id int32, name string, link sync.SubCmdRange) (*groupTreeNode, bool) {
	idx := sort.Search(len(n.children), func(i int) bool {
		return link.From.LEQ(n.children[i].link.From)
	})
	if idx < len(n.children) && n.children[idx].link.From.Equals(link.From) {
		return &n.children[idx], true
	}
	slice.InsertBefore(&n.children, idx, groupTreeNode{id, name, link, nil})
	return &n.children[idx], false
}

func (n *groupTreeNode) flatten(list []*service.ProfilingData_GpuSlices_Group, capture *path.Capture, parent int32) []*service.ProfilingData_GpuSlices_Group {
	if n.id != 0 {
		list = append(list, &service.ProfilingData_GpuSlices_Group{
			Id:       n.id,
			Name:     n.name,
			ParentId: parent,
			Link:     &path.Commands{Capture: capture, From: n.link.From, To: n.link.To},
		})
	}

	for i := range n.children {
		list = n.children[i].flatten(list, capture, n.id)
	}
	return list
}
