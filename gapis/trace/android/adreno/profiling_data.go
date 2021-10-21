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
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_service "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace/android/profile"
	"github.com/google/gapid/gapis/trace/android/utils"
)

var (
	slicesQuery = "" +
		"SELECT s.context_id, s.render_target, s.frame_id, s.submission_id, s.hw_queue_id, s.command_buffer, s.render_pass, s.ts, s.dur, s.id, s.name, depth, arg_set_id, track_id, t.name " +
		"FROM gpu_track t LEFT JOIN gpu_slice s " +
		"ON s.track_id = t.id WHERE t.scope = 'gpu_render_stage' ORDER BY s.ts"
	argsQueryFmt = "" +
		"SELECT key, string_value FROM args WHERE args.arg_set_id = %d"
	queueSubmitQuery = "" +
		"SELECT submission_id FROM gpu_slice s JOIN track t ON s.track_id = t.id WHERE s.name = 'vkQueueSubmit' AND t.name = 'Vulkan Events' ORDER BY submission_id"
	counterTracksQuery = "" +
		"SELECT id, name, unit, description FROM gpu_counter_track ORDER BY id"
	countersQueryFmt = "" +
		"SELECT ts, value FROM counter c WHERE c.track_id = %d ORDER BY ts"
	renderPassSliceName = "Surface"
)

func ProcessProfilingData(ctx context.Context, processor *perfetto.Processor, capture *path.Capture, desc *device.GpuCounterDescriptor, handleMapping map[uint64][]service.VulkanHandleMappingItem, syncData *sync.Data) (*service.ProfilingData, error) {
	slices, err := processGpuSlices(ctx, processor, capture, handleMapping, syncData)
	if err != nil {
		log.Err(ctx, err, "Failed to get GPU slices")
	}
	counters, err := processCounters(ctx, processor, desc)
	if err != nil {
		log.Err(ctx, err, "Failed to get GPU counters")
	}
	gpuCounters, err := profile.ComputeCounters(ctx, slices, counters)
	if err != nil {
		log.Err(ctx, err, "Failed to calculate performance data based on GPU slices and counters")
	}

	return &service.ProfilingData{
		Slices:      slices,
		Counters:    counters,
		GpuCounters: gpuCounters,
	}, nil
}

func fixContextIds(contextIDs []int64) {
	// This is a workaround a QC bug(b/192546534)
	// that causes first deviceID to be zero after a
	// renderpass change in the same queue submit.
	// So, we fill the zero devices with the existing
	// device id, where there is only one device id.

	zeroIndices := make([]int, 0)
	contextID := int64(0)

	for i, v := range contextIDs {
		if v == 0 {
			zeroIndices = append(zeroIndices, i)
			continue
		}

		if contextID == 0 {
			contextID = v
			continue
		}

		if contextID != v {
			// There are multiple devices
			// We cannot know which one to fill
			return
		}
	}

	for _, v := range zeroIndices {
		// If there is only one device in entire trace
		// We can assume that we possibly have only one device
		contextIDs[v] = contextID
	}
}

func processGpuSlices(ctx context.Context, processor *perfetto.Processor, capture *path.Capture, handleMapping map[uint64][]service.VulkanHandleMappingItem, syncData *sync.Data) (*service.ProfilingData_GpuSlices, error) {
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
	groupParentLookup := map[api.CmdSubmissionKey]*service.ProfilingData_GpuSlices_Group{}
	groups := []*service.ProfilingData_GpuSlices_Group{}
	groupIds := make([]int32, numSliceRows)
	var tracks []*service.ProfilingData_GpuSlices_Track
	// Grab all the column values. Depends on the order of columns selected in slicesQuery

	contextIds := slicesColumns[0].GetLongValues()
	fixContextIds(contextIds)
	profile.ExtractTraceHandles(ctx, contextIds, "VkDevice", handleMapping)

	renderTargets := slicesColumns[1].GetLongValues()
	profile.ExtractTraceHandles(ctx, renderTargets, "VkFramebuffer", handleMapping)

	commandBuffers := slicesColumns[5].GetLongValues()
	profile.ExtractTraceHandles(ctx, commandBuffers, "VkCommandBuffer", handleMapping)

	renderPasses := slicesColumns[6].GetLongValues()
	profile.ExtractTraceHandles(ctx, renderPasses, "VkRenderPass", handleMapping)

	frameIds := slicesColumns[2].GetLongValues()
	submissionIds := slicesColumns[3].GetLongValues()
	hwQueueIds := slicesColumns[4].GetLongValues()
	timestamps := slicesColumns[7].GetLongValues()
	durations := slicesColumns[8].GetLongValues()
	ids := slicesColumns[9].GetLongValues()
	names := slicesColumns[10].GetStringValues()
	depths := slicesColumns[11].GetLongValues()
	argSetIds := slicesColumns[12].GetLongValues()
	trackIds := slicesColumns[13].GetLongValues()
	trackNames := slicesColumns[14].GetStringValues()

	subCommandGroupMap := make(map[api.CmdSubmissionKey]int)
	for i, v := range submissionIds {
		subOrder, ok := submissionOrdering[v]
		if ok {
			cb := uint64(commandBuffers[i])
			key := api.CmdSubmissionKey{subOrder, cb, uint64(renderPasses[i]), uint64(renderTargets[i])}
			// Create a new group for each main renderPass slice.
			if indices, ok := syncData.SubmissionIndices[key]; ok && names[i] == renderPassSliceName {
				var idx []uint64
				if c, ok := subCommandGroupMap[key]; ok { // Sometimes multiple renderPass slices shares the same renderPass and renderTarget.
					idx = indices[c]
				} else {
					idx = indices[0]
					subCommandGroupMap[key] = 0
				}
				names[i] = fmt.Sprintf("%v", idx)

				parent := utils.FindParentGroup(ctx, subOrder, cb, groupParentLookup, &groups, syncData.SubmissionIndices, capture)
				group := &service.ProfilingData_GpuSlices_Group{
					Id:     int32(len(groups)),
					Name:   fmt.Sprintf("RenderPass %v, RenderTarget %v", uint64(renderPasses[i]), uint64(renderTargets[i])),
					Parent: parent,
					Link:   &path.Command{Capture: capture, Indices: idx},
				}
				groups = append(groups, group)
				subCommandGroupMap[key]++
			}
		} else {
			log.W(ctx, "Encountered submission ID mismatch %v", v)
		}
		// Find the group that the current slice belongs to and mark down group id.
		if len(groups) > 0 {
			groupIds[i] = groups[len(groups)-1].Id // Slices were time sorted and main renderPass slice comes first.
		} else {
			log.W(ctx, "Group missing for slice %v at submission %v, commandBuffer %v, renderPass %v, renderTarget %v", names[i], submissionIds[i], commandBuffers[i], renderPasses[i], renderTargets[i])
			groupIds[i] = -1
		}
	}

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
			Name:  "renderPass",
			Value: &service.ProfilingData_GpuSlices_Slice_Extra_IntValue{IntValue: uint64(renderPasses[i])},
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
			Id:      uint64(ids[i]),
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

	nameToSpec := map[string]*device.GpuCounterDescriptor_GpuCounterSpec{}
	if desc != nil {
		for _, spec := range desc.Specs {
			nameToSpec[spec.Name] = spec
		}
	}

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

		spec, _ := nameToSpec[names[i]]
		// TODO(apbodnar) Populate the `default` field once the trace processor supports it (b/147432390)
		counters[i] = &service.ProfilingData_Counter{
			Id:          uint32(trackIds[i]),
			Name:        names[i],
			Unit:        units[i],
			Description: descriptions[i],
			Spec:        spec,
			Timestamps:  timestamps,
			Values:      values,
		}
	}
	return counters, nil
}
