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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_service "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/service"
)

const (
	slicesQuery = "" +
		"SELECT s.context_id, s.render_target, s.frame_id, s.submission_id, s.hw_queue_id, s.command_buffer, s.render_pass, s.ts, s.dur, s.id, s.name, depth, arg_set_id, track_id, t.name " +
		"FROM gpu_track t LEFT JOIN gpu_slice s " +
		"ON s.track_id = t.id WHERE t.scope = 'gpu_render_stage' ORDER BY s.ts"
	argsQueryFmt = "" +
		"SELECT key, string_value FROM args WHERE args.arg_set_id = %d"
)

type Slice struct {
	Context       int64
	RenderTarget  int64
	Frame         int64
	Submission    int64
	HardwareQueue int64
	CommandBuffer int64
	Renderpass    int64
	Timestamp     uint64
	Duration      uint64
	SliceID       uint64
	Name          string
	Depth         int32
	ArgSet        int64
	Track         int32
	TrackName     string
	GroupID       int32 // To be filled in by caller.
}

type SliceData []Slice

func ExtractSliceData(ctx context.Context, processor *perfetto.Processor) (SliceData, error) {
	slicesQueryResult, err := processor.Query(slicesQuery)
	if err != nil {
		return nil, log.Errf(ctx, err, "SQL query failed: %v", slicesQuery)
	}

	slicesColumns := slicesQueryResult.GetColumns()
	data := make([]Slice, slicesQueryResult.GetNumRecords())
	for i := range data {
		data[i] = Slice{
			Context:       slicesColumns[0].GetLongValues()[i],
			RenderTarget:  slicesColumns[1].GetLongValues()[i],
			Frame:         slicesColumns[2].GetLongValues()[i],
			Submission:    slicesColumns[3].GetLongValues()[i],
			HardwareQueue: slicesColumns[4].GetLongValues()[i],
			CommandBuffer: slicesColumns[5].GetLongValues()[i],
			Renderpass:    slicesColumns[6].GetLongValues()[i],
			Timestamp:     uint64(slicesColumns[7].GetLongValues()[i]),
			Duration:      uint64(slicesColumns[8].GetLongValues()[i]),
			SliceID:       uint64(slicesColumns[9].GetLongValues()[i]),
			Name:          slicesColumns[10].GetStringValues()[i],
			Depth:         int32(slicesColumns[11].GetLongValues()[i]),
			ArgSet:        slicesColumns[12].GetLongValues()[i],
			Track:         int32(slicesColumns[13].GetLongValues()[i]),
			TrackName:     slicesColumns[14].GetStringValues()[i],
		}
	}

	return data, nil
}

func (d SliceData) MapIdentifiers(ctx context.Context, handleMapping map[uint64][]service.VulkanHandleMappingItem) {
	for i := range d {
		extractTraceHandle(ctx, &d[i].Context, "VkDevice", handleMapping)
		extractTraceHandle(ctx, &d[i].RenderTarget, "VkFramebuffer", handleMapping)
		extractTraceHandle(ctx, &d[i].CommandBuffer, "VkCommandBuffer", handleMapping)
		extractTraceHandle(ctx, &d[i].Renderpass, "VkRenderPass", handleMapping)
	}
}

// extractTraceHandle translates a handle based on the mappings.
func extractTraceHandle(ctx context.Context, replayHandle *int64, replayHandleType string, handleMapping map[uint64][]service.VulkanHandleMappingItem) {
	handles, ok := handleMapping[uint64(*replayHandle)]
	if !ok {
		// On some devices, when running in 32bit app compat mode, the handles
		// reported through Perfetto have this extra bit set in the last nibble,
		// which is typically all zeros. I.e. handles in the profiling data are
		// of the form 0x???????4, while exposed by the API they are 0x???????0.
		if (*replayHandle & 0xf) == 4 {
			handles, ok = handleMapping[uint64(*replayHandle&^4)]
		}
		if !ok {
			log.E(ctx, "%v not found in replay: %v", replayHandleType, *replayHandle)
			return
		}
	}
	for _, handle := range handles {
		if handle.HandleType == replayHandleType {
			*replayHandle = int64(handle.TraceValue)
			return
		}
	}

	log.E(ctx, "Incorrect Handle type for %v: %v", replayHandleType, *replayHandle)
}

func (d SliceData) ToService(ctx context.Context, processor *perfetto.Processor) *service.ProfilingData_GpuSlices {
	extraCache := newExtras(processor)

	tracks := map[int32]*service.ProfilingData_GpuSlices_Track{}
	slices := make([]*service.ProfilingData_GpuSlices_Slice, len(d))

	for i := range d {
		extras := fillInExtras(&d[i], extraCache.get(ctx, d[i].ArgSet))

		slices[i] = &service.ProfilingData_GpuSlices_Slice{
			Ts:      d[i].Timestamp,
			Dur:     d[i].Duration,
			Id:      d[i].SliceID,
			Label:   d[i].Name,
			Depth:   d[i].Depth,
			Extras:  extras,
			TrackId: d[i].Track,
			GroupId: d[i].GroupID,
		}

		if _, ok := tracks[d[i].Track]; !ok {
			tracks[d[i].Track] = &service.ProfilingData_GpuSlices_Track{
				Id:   d[i].Track,
				Name: d[i].TrackName,
			}
		}
	}

	return &service.ProfilingData_GpuSlices{
		Slices: slices,
		Tracks: flattenTracks(tracks),
	}
}

func fillInExtras(slice *Slice, extras []*service.ProfilingData_GpuSlices_Slice_Extra) []*service.ProfilingData_GpuSlices_Slice_Extra {
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "contextId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.Context)},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "renderTarget",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.RenderTarget)},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "commandBuffer",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.CommandBuffer)},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "renderPass",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.Renderpass)},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "frameId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.Frame)},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "submissionId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.Submission)},
	})
	extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
		Name:  "hwQueueId",
		Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(slice.HardwareQueue)},
	})
	return extras
}

func flattenTracks(tracks map[int32]*service.ProfilingData_GpuSlices_Track) []*service.ProfilingData_GpuSlices_Track {
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
