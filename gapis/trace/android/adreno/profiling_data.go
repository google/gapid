// Copyright (C) 2019 Google Inc.
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

package adreno

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_service "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/service"
)

var (
	slicesQuery = "" +
		"SELECT s.context_id, s.render_target, s.frame_id, s.submission_id, s.hw_queue_id, s.ts, s.dur, s.name, depth, arg_set_id, track_id, t.name " +
		"FROM gpu_track t LEFT JOIN gpu_slice s " +
		"ON s.track_id = t.id AND t.scope = 'gpu_render_stage'"
	argsQueryFmt = "" +
		"SELECT key, string_value FROM args WHERE args.arg_set_id = %d"
)

func ProcessProfilingData(ctx context.Context, processor *perfetto.Processor, handleMapping *map[uint64][]service.VulkanHandleMappingItem) (*service.ProfilingData, error) {
	slicesQueryResult, err := processor.Query(slicesQuery)
	if err != nil {
		return nil, log.Errf(ctx, err, "SQL query failed: %v", slicesQuery)
	}

	trackIdCache := make(map[int64]bool)
	argsQueryCache := make(map[int64]*perfetto_service.QueryResult)
	slicesColumns := slicesQueryResult.GetColumns()
	numSliceRows := slicesQueryResult.GetNumRecords()
	slices := make([]*service.ProfilingData_GpuSlices_Slice, numSliceRows)
	var tracks []*service.ProfilingData_GpuSlices_Track
	// Grab all the column values. Depends on the order of columns selected in slicesQuery

	contextIds := slicesColumns[0].GetLongValues()
	for i, v := range contextIds {
		if m, ok := (*handleMapping)[uint64(v)]; ok {
			contextIds[i] = int64(m[0].TraceValue)
		} else {
			log.E(ctx, "Context Id could not found: %v", v)
		}
	}

	renderTargets := slicesColumns[1].GetLongValues()
	for i, v := range renderTargets {
		if m, ok := (*handleMapping)[uint64(v)]; ok {
			renderTargets[i] = int64(m[0].TraceValue)
		} else {
			log.E(ctx, "Render Target could not found: %v", v)
		}
	}

	frameIds := slicesColumns[2].GetLongValues()
	submissionIds := slicesColumns[3].GetLongValues()
	hwQueueIds := slicesColumns[4].GetLongValues()
	timestamps := slicesColumns[5].GetLongValues()
	durations := slicesColumns[6].GetLongValues()
	names := slicesColumns[7].GetStringValues()
	depths := slicesColumns[8].GetLongValues()
	argSetIds := slicesColumns[9].GetLongValues()
	trackIds := slicesColumns[10].GetLongValues()
	trackNames := slicesColumns[11].GetStringValues()

	for i := uint64(0); i < numSliceRows; i++ {
		var argsQueryResult *perfetto_service.QueryResult
		var ok bool
		if argsQueryResult, ok = argsQueryCache[argSetIds[i]]; !ok {
			argsQuery := fmt.Sprintf(argsQueryFmt, argSetIds[i])
			argsQueryResult, err = processor.Query(argsQuery)
			if err != nil {
				log.W(ctx, "SQL query failed: %v", argsQuery)
			}
			argsQueryCache[argSetIds[i]] = argsQueryResult
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
		extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
			Name:  "contextId",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(contextIds[i])},
		})
		extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
			Name:  "renderTarget",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(renderTargets[i])},
		})
		extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
			Name:  "frameId",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(frameIds[i])},
		})
		extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
			Name:  "submissionId",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(submissionIds[i])},
		})
		extras = append(extras, &service.ProfilingData_GpuSlices_Slice_Extra{
			Name:  "hwQueueId",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(hwQueueIds[i])},
		})

		slices[i] = &service.ProfilingData_GpuSlices_Slice{
			Ts:      uint64(timestamps[i]),
			Dur:     uint64(durations[i]),
			Label:   names[i],
			Depth:   int32(depths[i]),
			Extras:  extras,
			TrackId: int32(trackIds[i]),
		}

		if _, ok := trackIdCache[trackIds[i]]; !ok {
			trackIdCache[trackIds[i]] = true
			tracks = append(tracks, &service.ProfilingData_GpuSlices_Track{
				Id:   int32(trackIds[i]),
				Name: trackNames[i],
			})
		}
	}

	return &service.ProfilingData{Slices: &service.ProfilingData_GpuSlices{
		Slices: slices,
		Tracks: tracks,
	}}, nil
}
