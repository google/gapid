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
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_service "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

var (
	slicesQuery = "" +
		"SELECT s.context_id, s.render_target, s.frame_id, s.submission_id, s.hw_queue_id, s.command_buffer, s.ts, s.dur, s.name, depth, arg_set_id, track_id, t.name " +
		"FROM gpu_track t LEFT JOIN gpu_slice s " +
		"ON s.track_id = t.id WHERE t.scope = 'gpu_render_stage'"
	argsQueryFmt = "" +
		"SELECT key, string_value FROM args WHERE args.arg_set_id = %d"
	queueSubmitQuery = "" +
		"SELECT submission_id FROM gpu_slice s JOIN track t ON s.track_id = t.id WHERE s.name = 'vkQueueSubmit' AND t.name = 'Vulkan Events' ORDER BY submission_id"
	counterTracksQuery = "" +
		"SELECT id, name, unit, description FROM gpu_counter_track ORDER BY id"
	countersQueryFmt = "" +
		"SELECT ts, value FROM counter c WHERE c.track_id = %d ORDER BY ts"
)

func ProcessProfilingData(ctx context.Context, processor *perfetto.Processor, capture *path.Capture, desc *device.GpuCounterDescriptor, handleMapping *map[uint64][]service.VulkanHandleMappingItem, submissionIds *map[api.CommandSubmissionKey][]uint64) (*service.ProfilingData, error) {
	slices, err := processGpuSlices(ctx, processor, capture, handleMapping, submissionIds)
	if err != nil {
		log.Err(ctx, err, "Failed to get GPU slices")
	}
	counters, err := processCounters(ctx, processor, desc)
	if err != nil {
		log.Err(ctx, err, "Failed to get GPU counters")
	}
	return &service.ProfilingData{Slices: slices, Counters: counters}, nil
}

func processGpuSlices(ctx context.Context, processor *perfetto.Processor, capture *path.Capture, handleMapping *map[uint64][]service.VulkanHandleMappingItem, subCommandIndicesMap *map[api.CommandSubmissionKey][]uint64) (*service.ProfilingData_GpuSlices, error) {
	slicesQueryResult, err := processor.Query(slicesQuery)
	if err != nil {
		return nil, log.Errf(ctx, err, "SQL query failed: %v", slicesQuery)
	}

	queueSubmitQueryResult, err := processor.Query(queueSubmitQuery)
	if err != nil {
		return nil, log.Errf(ctx, err, "SQL query failed: %v", queueSubmitQuery)
	}
	queueSubmitColumns := queueSubmitQueryResult.GetColumns()
	queueSubmitIds := queueSubmitColumns[0].GetLongValues()
	submissionOrdering := make(map[int64]uint64)

	for i, v := range queueSubmitIds {
		submissionOrdering[v] = uint64(i)
	}

	trackIdCache := make(map[int64]bool)
	argsQueryCache := make(map[int64]*perfetto_service.QueryResult)
	slicesColumns := slicesQueryResult.GetColumns()
	numSliceRows := slicesQueryResult.GetNumRecords()
	slices := make([]*service.ProfilingData_GpuSlices_Slice, numSliceRows)
	groups := make([]*service.ProfilingData_GpuSlices_Group, 0)
	groupIds := make([]int32, numSliceRows)
	var tracks []*service.ProfilingData_GpuSlices_Track
	// Grab all the column values. Depends on the order of columns selected in slicesQuery

	contextIds := slicesColumns[0].GetLongValues()
	for i, v := range contextIds {
		if m, ok := (*handleMapping)[uint64(v)]; ok {
			contextIds[i] = int64(m[0].TraceValue)
		} else {
			log.E(ctx, "Context Id not found: %v", v)
		}
	}

	renderTargets := slicesColumns[1].GetLongValues()
	for i, v := range renderTargets {
		if m, ok := (*handleMapping)[uint64(v)]; ok {
			renderTargets[i] = int64(m[0].TraceValue)
		} else {
			log.E(ctx, "Render Target not found: %v", v)
		}
	}

	commandBuffers := slicesColumns[5].GetLongValues()
	for i, v := range commandBuffers {
		if m, ok := (*handleMapping)[uint64(v)]; ok {
			commandBuffers[i] = int64(m[0].TraceValue)
		} else {
			log.E(ctx, "Command Buffer not found: %v", v)
		}
	}

	submissionIds := slicesColumns[3].GetLongValues()
	currentSubId := int64(-1)
	subCommandGroupMap := make(map[api.CommandSubmissionKey]bool)
	for i, v := range submissionIds {
		if v != currentSubId {
			currentSubId = v
			// We'll never need the previous keys again
			subCommandGroupMap = make(map[api.CommandSubmissionKey]bool)
		}
		subOrder, ok := submissionOrdering[v]
		if ok {
			cb := commandBuffers[i]
			key := api.CommandSubmissionKey{SubmissionOrder: subOrder, CommandBuffer: cb}
			if _, ok := subCommandGroupMap[key]; !ok {
				if indices, ok := (*subCommandIndicesMap)[key]; ok {
					group := &service.ProfilingData_GpuSlices_Group{
						Id:   int32(len(groups)),
						Link: &path.Command{Capture: capture, Indices: indices},
					}
					groups = append(groups, group)
					subCommandGroupMap[key] = true
				}
			}
		} else {
			log.W(ctx, "Encountered submission ID mismatch %v", v)
		}

		groupIds[i] = int32(len(groups)) - 1
	}

	frameIds := slicesColumns[2].GetLongValues()
	hwQueueIds := slicesColumns[4].GetLongValues()
	timestamps := slicesColumns[6].GetLongValues()
	durations := slicesColumns[7].GetLongValues()
	names := slicesColumns[8].GetStringValues()
	depths := slicesColumns[9].GetLongValues()
	argSetIds := slicesColumns[10].GetLongValues()
	trackIds := slicesColumns[11].GetLongValues()
	trackNames := slicesColumns[12].GetStringValues()

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
			Name:  "commandBuffer",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(commandBuffers[i])},
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

		if names[i] == "Surface" && groupIds[i] != -1 {
			names[i] = fmt.Sprintf("%v", groups[groupIds[i]].Link.Indices)
		}

		slices[i] = &service.ProfilingData_GpuSlices_Slice{
			Ts:      uint64(timestamps[i]),
			Dur:     uint64(durations[i]),
			Label:   names[i],
			Depth:   int32(depths[i]),
			Extras:  extras,
			TrackId: int32(trackIds[i]),
			GroupId: groupIds[i],
		}

		if _, ok := trackIdCache[trackIds[i]]; !ok {
			trackIdCache[trackIds[i]] = true
			tracks = append(tracks, &service.ProfilingData_GpuSlices_Track{
				Id:   int32(trackIds[i]),
				Name: trackNames[i],
			})
		}
	}

	return &service.ProfilingData_GpuSlices{
		Slices: slices,
		Tracks: tracks,
		Groups: groups,
	}, nil
}

func processCounters(ctx context.Context, processor *perfetto.Processor, desc *device.GpuCounterDescriptor) ([]*service.ProfilingData_Counter, error) {
	counterTracksQueryResult, err := processor.Query(counterTracksQuery)
	if err != nil {
		return nil, log.Errf(ctx, err, "SQL query failed: %v", counterTracksQuery)
	}
	// t.id, name, unit, description, ts, value
	tracksColumns := counterTracksQueryResult.GetColumns()
	numTracksRows := counterTracksQueryResult.GetNumRecords()
	counters := make([]*service.ProfilingData_Counter, numTracksRows)
	// Grab all the column values. Depends on the order of columns selected in countersQuery
	trackIds := tracksColumns[0].GetLongValues()
	names := tracksColumns[1].GetStringValues()
	units := tracksColumns[2].GetStringValues()
	descriptions := tracksColumns[3].GetStringValues()

	for i := uint64(0); i < numTracksRows; i++ {
		countersQuery := fmt.Sprintf(countersQueryFmt, trackIds[i])
		countersQueryResult, err := processor.Query(countersQuery)
		countersColumns := countersQueryResult.GetColumns()
		if err != nil {
			return nil, log.Errf(ctx, err, "SQL query failed: %v", counterTracksQuery)
		}
		timestampsLong := countersColumns[0].GetLongValues()
		timestamps := make([]uint64, len(timestampsLong))
		for i, t := range timestampsLong {
			timestamps[i] = uint64(t)
		}
		values := countersColumns[1].GetDoubleValues()
		// TODO(apbodnar) Populate the `default` field once the trace processor supports it (b/147432390)
		counters[i] = &service.ProfilingData_Counter{
			Id:          uint32(trackIds[i]),
			Name:        names[i],
			Unit:        units[i],
			Description: descriptions[i],
			Timestamps:  timestamps,
			Values:      values,
		}
	}
	return counters, nil
}
