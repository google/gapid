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

package profile

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/gapis/service"
)

const (
	gpuTimeMetricId       int32 = 0
	gpuWallTimeMetricId   int32 = 1
	counterMetricIdOffset int32 = 2
)

// For CPU commands, calculate their summarized GPU performance.
func ComputeCounters(ctx context.Context, slices *service.ProfilingData_GpuSlices, counters []*service.ProfilingData_Counter) (*service.ProfilingData_GpuCounters, error) {
	metrics := []*service.ProfilingData_GpuCounters_Metric{}

	// Filter out the slices that are at depth 0 and belong to a command,
	// then sort them based on the start time.
	groupToEntry := map[int32]*service.ProfilingData_GpuCounters_Entry{}
	for _, group := range slices.Groups {
		groupToEntry[group.Id] = &service.ProfilingData_GpuCounters_Entry{
			CommandIndex:  group.Link.Indices,
			MetricToValue: map[int32]float64{},
		}
	}
	filteredSlices := []*service.ProfilingData_GpuSlices_Slice{}
	for i := 0; i < len(slices.Slices); i++ {
		if slices.Slices[i].Depth == 0 && groupToEntry[slices.Slices[i].GroupId] != nil {
			filteredSlices = append(filteredSlices, slices.Slices[i])
		}
	}
	sort.Slice(filteredSlices, func(i, j int) bool {
		return filteredSlices[i].Ts < filteredSlices[j].Ts
	})

	// Group slices based on their group id.
	groupToSlices := map[int32][]*service.ProfilingData_GpuSlices_Slice{}
	for i := 0; i < len(filteredSlices); i++ {
		groupId := filteredSlices[i].GroupId
		groupToSlices[groupId] = append(groupToSlices[groupId], filteredSlices[i])
	}

	// Calculate GPU Time Performance and GPU Wall Time Performance for all leaf groups/commands.
	setTimeMetrics(groupToSlices, &metrics, groupToEntry)

	// Calculate GPU Counter Performances for all leaf groups/commands.
	setGpuCounterMetrics(ctx, groupToSlices, counters, &metrics, groupToEntry)

	// Merge and organize the leaf entries.
	entries := mergeLeafEntries(metrics, groupToEntry)

	return &service.ProfilingData_GpuCounters{
		Metrics: metrics,
		Entries: entries,
	}, nil
}

// Create GPU time metric metadata, calculate time performance for each GPU
// slice group, and append the result to corresponding entries.
func setTimeMetrics(groupToSlices map[int32][]*service.ProfilingData_GpuSlices_Slice, metrics *[]*service.ProfilingData_GpuCounters_Metric, groupToEntry map[int32]*service.ProfilingData_GpuCounters_Entry) {
	*metrics = append(*metrics, &service.ProfilingData_GpuCounters_Metric{
		Id:   gpuTimeMetricId,
		Name: "GPU Time",
		Unit: "ns",
		Op:   service.ProfilingData_GpuCounters_Metric_Summation,
	})
	*metrics = append(*metrics, &service.ProfilingData_GpuCounters_Metric{
		Id:   gpuWallTimeMetricId,
		Name: "GPU Wall Time",
		Unit: "ns",
		Op:   service.ProfilingData_GpuCounters_Metric_Summation,
	})
	for groupId, slices := range groupToSlices {
		gpuTime, wallTime := gpuTimeForGroup(slices)
		entry := groupToEntry[groupId]
		entry.MetricToValue[gpuTimeMetricId] = float64(gpuTime)
		entry.MetricToValue[gpuWallTimeMetricId] = float64(wallTime)
	}
}

// Calculate GPU-time and wall-time for a specific GPU slice group.
func gpuTimeForGroup(slices []*service.ProfilingData_GpuSlices_Slice) (uint64, uint64) {
	gpuTime, wallTime := uint64(0), uint64(0)
	lastEnd := uint64(0)
	for _, slice := range slices {
		duration := slice.Dur
		gpuTime += duration
		if slice.Ts < lastEnd {
			if slice.Ts+slice.Dur <= lastEnd {
				continue // completely contained within the other, can ignore it.
			}
			duration -= lastEnd - slice.Ts
		}
		wallTime += duration
		lastEnd = slice.Ts + slice.Dur
	}
	return gpuTime, wallTime
}

// Create GPU counter metric metadata, calculate counter performance for each
// GPU slice group, and append the result to corresponding entries.
func setGpuCounterMetrics(ctx context.Context, groupToSlices map[int32][]*service.ProfilingData_GpuSlices_Slice, counters []*service.ProfilingData_Counter, metrics *[]*service.ProfilingData_GpuCounters_Metric, groupToEntry map[int32]*service.ProfilingData_GpuCounters_Entry) {
	for i, counter := range counters {
		metricId := counterMetricIdOffset + int32(i)
		op := getCounterAggregationMethod(counter)
		*metrics = append(*metrics, &service.ProfilingData_GpuCounters_Metric{
			Id:   metricId,
			Name: counter.Name,
			Unit: counter.Unit,
			Op:   op,
		})
		if op != service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg {
			log.E(ctx, "Counter aggregation method not implemented yet. Operation: %v", op)
			continue
		}
		for groupId, slices := range groupToSlices {
			counterPerf := counterPerfForGroup(slices, counter)
			entry := groupToEntry[groupId]
			entry.MetricToValue[metricId] = counterPerf
		}
	}
}

// Calculate GPU counter performance for a specific GPU slice group, and a
// specific GPU counter.
func counterPerfForGroup(slices []*service.ProfilingData_GpuSlices_Slice, counter *service.ProfilingData_Counter) float64 {
	// Reduce overlapped counter samples size.
	// Filter out the counter samples whose implicit range collides with `slices`'s gpu time.
	rangeStart, rangeEnd := ^uint64(0), uint64(0)
	ts, vs := []uint64{}, []float64{}
	for _, slice := range slices {
		rangeStart = u64.Min(rangeStart, slice.Ts)
		rangeEnd = u64.Max(rangeEnd, slice.Ts+slice.Dur)
	}
	for i := range counter.Timestamps {
		if i > 0 && counter.Timestamps[i-1] > rangeEnd {
			break
		}
		if counter.Timestamps[i] > rangeStart {
			ts = append(ts, counter.Timestamps[i])
			vs = append(vs, counter.Values[i])
		}
	}
	if len(ts) == 0 {
		return float64(-1)
	}

	// Aggregate counter samples.
	// Contribution time is the overlapped time between a counter sample's implicit range and a gpu slice.
	ctSum := uint64(0)        // Accumulation of contribution time.
	weightedSum := float64(0) // Accumulation of (counter value * counter's contribution time).
	for _, slice := range slices {
		sStart, sEnd := slice.Ts, slice.Ts+slice.Dur
		if ts[0] > sStart {
			ct := u64.Min(ts[0], sEnd) - sStart
			ctSum += ct
			weightedSum += float64(ct) * vs[0]
		}
		for i := 1; i < len(ts); i++ {
			cStart, cEnd := ts[i-1], ts[i]
			if cEnd < sStart { // Sample earlier than GPU slice's span.
				continue
			} else if cEnd < sEnd { // Sample inside GPU slice's span, or sample's latter part overlaps with slice.
				ct := cEnd - u64.Max(cStart, sStart)
				ctSum += ct
				weightedSum += float64(ct) * vs[i]
			} else if cStart < sEnd { // Sample wraps GPU slice's span, or sample's earlier part overlaps with slice.
				ct := sEnd - u64.Max(sStart, cStart)
				ctSum += ct
				weightedSum += float64(ct) * vs[i]
				break
			}
		}
	}

	// Return result.
	if ctSum == 0 {
		return float64(0)
	} else {
		return weightedSum / float64(ctSum)
	}
}

// Merge leaf group entries if they belong to the same command, and also derive
// the parent command nodes' GPU performances based on the leaf entries.
func mergeLeafEntries(metrics []*service.ProfilingData_GpuCounters_Metric, groupToEntry map[int32]*service.ProfilingData_GpuCounters_Entry) []*service.ProfilingData_GpuCounters_Entry {
	mergedEntries := []*service.ProfilingData_GpuCounters_Entry{}

	// Find out all the self/parent command nodes that may need performance merging.
	indexToGroups := map[string][]int32{} // string formatted command index -> a list of contained groups referenced by group id.
	for groupId, entry := range groupToEntry {
		// The performance of one leaf group/command contributes to itself and all the ancestors up to the root command node.
		leafIdx := entry.CommandIndex
		for end := len(leafIdx); end > 0; end-- {
			mergedIdxStr := encodeIndex(leafIdx[0:end])
			indexToGroups[mergedIdxStr] = append(indexToGroups[mergedIdxStr], groupId)
		}
	}

	for commandIndex, leafGroupIds := range indexToGroups {
		mergedEntry := &service.ProfilingData_GpuCounters_Entry{
			CommandIndex:  decodeIndex(commandIndex),
			MetricToValue: map[int32]float64{},
		}
		for _, metric := range metrics {
			perf := float64(0)
			if metric.Op == service.ProfilingData_GpuCounters_Metric_Summation {
				for _, id := range leafGroupIds {
					perf += groupToEntry[id].MetricToValue[metric.Id]
				}
			} else if metric.Op == service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg {
				timeSum, valueSum := float64(0), float64(0)
				for _, id := range leafGroupIds {
					timeSum += groupToEntry[id].MetricToValue[gpuTimeMetricId]
					valueSum += groupToEntry[id].MetricToValue[gpuTimeMetricId] * groupToEntry[id].MetricToValue[metric.Id]
				}
				if timeSum != 0 {
					perf = valueSum / timeSum
				}
			}
			mergedEntry.MetricToValue[metric.Id] = perf
		}
		mergedEntries = append(mergedEntries, mergedEntry)
	}

	return mergedEntries
}

// Evaluate and return the appropriate aggregation method for a GPU counter.
func getCounterAggregationMethod(counter *service.ProfilingData_Counter) service.ProfilingData_GpuCounters_Metric_AggregationOperator {
	// TODO: Use time-weighted average to aggregate all counters for now. May need vendor's support. Bug tracked with b/158057709.
	return service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg
}

// Encode a command index, transform from array format to string format.
func encodeIndex(array_index []uint64) string {
	str := make([]string, len(array_index))
	for i, v := range array_index {
		str[i] = strconv.FormatUint(v, 10)
	}
	return strings.Join(str, ",")
}

// Decode a command index, transform from string format to array format.
func decodeIndex(str_index string) []uint64 {
	indexes := strings.Split(str_index, ",")
	array := make([]uint64, len(indexes))
	for i := range array {
		array[i], _ = strconv.ParseUint(indexes[i], 10, 0)
	}
	return array
}
