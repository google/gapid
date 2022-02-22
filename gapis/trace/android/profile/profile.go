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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/f64"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
)

const (
	gpuTimeMetricId       int32 = 0
	gpuWallTimeMetricId   int32 = 1
	counterMetricIdOffset int32 = 2
)

type ProfilingData struct {
	Groups        *GroupTree
	Slices        SliceData
	Counters      []*service.ProfilingData_Counter
	GpuCounters   *service.ProfilingData_GpuCounters
	CounterGroups []*service.ProfilingData_CounterGroup
}

func NewProfilingData() *ProfilingData {
	return &ProfilingData{
		Groups: NewGroupTree(),
	}
}

// ComputeCounters calculates the summarized GPU performance and fills in the GpuCounters field.
func (pd *ProfilingData) ComputeCounters(ctx context.Context) {
	metrics := []*service.ProfilingData_GpuCounters_Metric{}

	// Filter out the slices that are at depth 0 and belong to a command,
	// then sort them based on the start time.
	groupToEntry := map[int32]*service.ProfilingData_GpuCounters_Entry{}
	groupToParent := map[int32]int32{}
	pd.Groups.Visit(func(parent int32, node *groupTreeNode) {
		groupToEntry[node.id] = &service.ProfilingData_GpuCounters_Entry{
			GroupId:       node.id,
			MetricToValue: map[int32]*service.ProfilingData_GpuCounters_Perf{},
		}
		groupToParent[node.id] = parent
	})
	filteredSlices := []*Slice{}
	for i := 0; i < len(pd.Slices); i++ {
		if pd.Slices[i].Depth == 0 && groupToEntry[pd.Slices[i].GroupID] != nil {
			filteredSlices = append(filteredSlices, &pd.Slices[i])
		}
	}
	sort.Slice(filteredSlices, func(i, j int) bool {
		return filteredSlices[i].Timestamp < filteredSlices[j].Timestamp
	})

	// Group slices based on their group id.
	groupToSlices := map[int32][]*Slice{}
	for i := 0; i < len(filteredSlices); i++ {
		e, _ := groupToEntry[filteredSlices[i].GroupID]
		// Attribute a slice to its direct group and all ancestor groups.
		for e != nil {
			groupToSlices[e.GroupId] = append(groupToSlices[e.GroupId], filteredSlices[i])
			parent, _ := groupToParent[e.GroupId]
			e, _ = groupToEntry[parent]
		}
	}

	// Calculate GPU Time Performance and GPU Wall Time Performance for all leaf groups/commands.
	setTimeMetrics(ctx, groupToSlices, &metrics, groupToEntry)

	// Calculate GPU Counter Performances for all leaf groups/commands.
	setGpuCounterMetrics(ctx, groupToSlices, pd.Counters, filteredSlices, &metrics, groupToEntry)

	// Collect the entries.
	entries := []*service.ProfilingData_GpuCounters_Entry{}
	for _, entry := range groupToEntry {
		entries = append(entries, entry)
	}

	pd.GpuCounters = &service.ProfilingData_GpuCounters{
		Metrics: metrics,
		Entries: entries,
	}
}

// Create GPU time metric metadata, calculate time performance for each GPU
// slice group, and append the result to corresponding entries.
func setTimeMetrics(ctx context.Context, groupToSlices map[int32][]*Slice,
	metrics *[]*service.ProfilingData_GpuCounters_Metric,
	groupToEntry map[int32]*service.ProfilingData_GpuCounters_Entry) {

	gpuTimeMetric := &service.ProfilingData_GpuCounters_Metric{
		Id:              gpuTimeMetricId,
		Name:            "GPU Time",
		Unit:            strconv.Itoa(int(device.GpuCounterDescriptor_NANOSECOND)),
		Op:              service.ProfilingData_GpuCounters_Metric_Summation,
		Description:     "Total time spent on the GPU, computed by summing the duration of all the GPU activity slices.",
		SelectByDefault: true,
		Type:            service.ProfilingData_GpuCounters_Metric_Hardware,
	}
	*metrics = append(*metrics, gpuTimeMetric)
	wallTimeMetric := &service.ProfilingData_GpuCounters_Metric{
		Id:              gpuWallTimeMetricId,
		Name:            "GPU Wall Time",
		Unit:            strconv.Itoa(int(device.GpuCounterDescriptor_NANOSECOND)),
		Op:              service.ProfilingData_GpuCounters_Metric_Summation,
		Description:     "Total wall time spent on the GPU, computed by summing the duration of all the GPU activity slices, but accounting for overlapping slices.",
		SelectByDefault: false,
		Type:            service.ProfilingData_GpuCounters_Metric_Hardware,
	}
	*metrics = append(*metrics, wallTimeMetric)
	gpuTimeSum, wallTimeSum := float64(0), float64(0)
	gpuTimeAvg, wallTimeAvg := float64(-1), float64(-1)
	for groupId, slices := range groupToSlices {
		gpuTime, wallTime := gpuTimeForGroup(slices)
		gpuTimeSum += gpuTime
		wallTimeSum += wallTime
		entry := groupToEntry[groupId]
		if entry == nil {
			log.W(ctx, "Didn't find corresponding counter performance entry for GPU slice group %v.", groupId)
			continue
		}
		entry.MetricToValue[gpuTimeMetricId] = &service.ProfilingData_GpuCounters_Perf{
			Estimate: gpuTime,
			Min:      gpuTime,
			Max:      gpuTime,
		}
		entry.MetricToValue[gpuWallTimeMetricId] = &service.ProfilingData_GpuCounters_Perf{
			Estimate: wallTime,
			Min:      wallTime,
			Max:      wallTime,
		}
	}
	if len(groupToSlices) > 0 {
		gpuTimeAvg = gpuTimeSum / float64(len(groupToSlices))
		wallTimeAvg = wallTimeSum / float64(len(groupToSlices))
	}
	gpuTimeMetric.Average = gpuTimeAvg
	wallTimeMetric.Average = wallTimeAvg
}

// Calculate GPU-time and wall-time for a specific GPU slice group.
func gpuTimeForGroup(slices []*Slice) (float64, float64) {
	gpuTime, wallTime := uint64(0), uint64(0)
	lastEnd := uint64(0)
	for _, slice := range slices {
		duration := slice.Duration
		gpuTime += duration
		if slice.Timestamp < lastEnd {
			if slice.Timestamp+slice.Duration <= lastEnd {
				continue // completely contained within the other, can ignore it.
			}
			duration -= lastEnd - slice.Timestamp
		}
		wallTime += duration
		lastEnd = slice.Timestamp + slice.Duration
	}
	return float64(gpuTime), float64(wallTime)
}

// Create GPU counter metric metadata, calculate counter performance for each
// GPU slice group, and append the result to corresponding entries.
func setGpuCounterMetrics(ctx context.Context, groupToSlices map[int32][]*Slice,
	counters []*service.ProfilingData_Counter, globalSlices []*Slice,
	metrics *[]*service.ProfilingData_GpuCounters_Metric,
	groupToEntry map[int32]*service.ProfilingData_GpuCounters_Entry) {

	for i, counter := range counters {
		metricId := counterMetricIdOffset + int32(i)
		op := getCounterAggregationMethod(counter)
		description := ""
		selectByDefault := false
		if counter.Spec != nil {
			description = counter.Spec.Description
			selectByDefault = counter.Spec.SelectByDefault
		}
		counterMetric := &service.ProfilingData_GpuCounters_Metric{
			Id:              metricId,
			CounterId:       counter.Id,
			Name:            counter.Name,
			Unit:            counter.Unit,
			Op:              op,
			Description:     description,
			SelectByDefault: selectByDefault,
		}
		*metrics = append(*metrics, counterMetric)
		if op != service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg {
			log.E(ctx, "Counter aggregation method not implemented yet. Operation: %v", op)
			continue
		}
		concurrentSlicesCount := scanConcurrency(globalSlices, counter)
		counterPerfSum, counterPerfAvg := float64(0), float64(-1)
		for groupId, slices := range groupToSlices {
			estimateSet, minSet, maxSet := mapCounterSamples(slices, counter, concurrentSlicesCount)
			estimate := aggregateCounterSamples(estimateSet, counter)
			counterPerfSum += estimate
			// Extra comparison here because minSet/maxSet only denote minimal/maximal
			// number of counter samples inclusion strategy, the aggregation result
			// may not be the smallest/largest actually.
			min, max := estimate, estimate
			if minSetRes := aggregateCounterSamples(minSet, counter); minSetRes != -1 {
				min = f64.MinOf(min, minSetRes)
				max = f64.MaxOf(max, minSetRes)
			}
			if maxSetRes := aggregateCounterSamples(maxSet, counter); maxSetRes != -1 {
				min = f64.MinOf(min, maxSetRes)
				max = f64.MaxOf(max, maxSetRes)
			}
			groupToEntry[groupId].MetricToValue[metricId] = &service.ProfilingData_GpuCounters_Perf{
				Estimate:        estimate,
				Min:             min,
				Max:             max,
				EstimateSamples: estimateSet,
				MinSamples:      minSet,
				MaxSamples:      maxSet,
			}
		}
		if len(groupToSlices) > 0 {
			counterPerfAvg = counterPerfSum / float64(len(groupToSlices))
		}
		counterMetric.Average = counterPerfAvg
	}
}

// Scan global slices and count concurrent slices for each counter sample.
func scanConcurrency(globalSlices []*Slice, counter *service.ProfilingData_Counter) []int {
	slicesCount := make([]int, len(counter.Timestamps))
	for _, slice := range globalSlices {
		sStart, sEnd := slice.Timestamp, slice.Timestamp+slice.Duration
		for i := 1; i < len(counter.Timestamps); i++ {
			cStart, cEnd := counter.Timestamps[i-1], counter.Timestamps[i]
			if cEnd < sStart { // Sample earlier than GPU slice's span.
				continue
			} else if cStart > sEnd { // Sample later than GPU slice's span.
				break
			} else { // Sample overlaps with GPU slice's span.
				slicesCount[i]++
			}
		}
	}
	return slicesCount
}

// Map counter samples to GPU slice. When collecting samples, three sets will
// be maintained based on attribution strategy: the minimum set,
// the best guess set, and the maximum set.
// The returned results map {sample index} to {sample weight}.
func mapCounterSamples(slices []*Slice, counter *service.ProfilingData_Counter, concurrentSlicesCount []int) (map[int32]float64, map[int32]float64, map[int32]float64) {
	estimateSet, minSet, maxSet := map[int32]float64{}, map[int32]float64{}, map[int32]float64{}
	for _, slice := range slices {
		sStart, sEnd := slice.Timestamp, slice.Timestamp+slice.Duration
		for i := int32(1); i < int32(len(counter.Timestamps)); i++ {
			cStart, cEnd := counter.Timestamps[i-1], counter.Timestamps[i]
			concurrencyWeight := 1.0
			if concurrentSlicesCount[i] > 1 {
				concurrencyWeight = 1 / float64(concurrentSlicesCount[i])
			}
			if cEnd < sStart { // Sample earlier than GPU slice's span.
				continue
			} else if cStart > sEnd { // Sample later than GPU slice's span.
				break
			} else if cStart > sStart && cEnd < sEnd { // Sample is contained inside GPU slice's span.
				estimateSet[i] = 1 * concurrencyWeight
				// Only add to minSet when there's no concurrent slices, because of the
				// possibility that the sample belongs entirely to one of the slices.
				if concurrencyWeight == 1.0 {
					minSet[i] = 1
				}
				maxSet[i] = 1
			} else { // Sample contains, or partially overlap with GPU slice's span.
				percent := float64(0)
				if cEnd != cStart {
					percent = float64(u64.Min(cEnd, sEnd)-u64.Max(cStart, sStart)) / float64(cEnd-cStart) // Time overlap weight.
					percent *= concurrencyWeight
				}
				if _, ok := estimateSet[i]; !ok {
					estimateSet[i] = 0
				}
				estimateSet[i] += percent
				maxSet[i] = 1
			}
		}
	}
	return estimateSet, minSet, maxSet
}

// Aggregate counter samples to a single value based on counter weight.
func aggregateCounterSamples(sampleWeight map[int32]float64, counter *service.ProfilingData_Counter) float64 {
	switch getCounterAggregationMethod(counter) {
	case service.ProfilingData_GpuCounters_Metric_Summation:
		ValueSum := float64(0)
		for idx, weight := range sampleWeight {
			ValueSum += counter.Values[idx] * weight
		}
		return ValueSum
	case service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg:
		ValueSum, timeSum := float64(0), float64(0)
		for idx, weight := range sampleWeight {
			ValueSum += counter.Values[idx] * float64(counter.Timestamps[idx]-counter.Timestamps[idx-1]) * weight
			timeSum += float64(counter.Timestamps[idx]-counter.Timestamps[idx-1]) * weight
		}
		if timeSum != 0 {
			return ValueSum / timeSum
		} else {
			return -1
		}
	default:
		return -1
	}
}

// Evaluate and return the appropriate aggregation method for a GPU counter.
func getCounterAggregationMethod(counter *service.ProfilingData_Counter) service.ProfilingData_GpuCounters_Metric_AggregationOperator {
	// TODO: Use time-weighted average to aggregate all counters for now. May need vendor's support. Bug tracked with b/158057709.
	return service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg
}

func (pd *ProfilingData) MergeStaticAnalysis(ctx context.Context, staticAnalysis *api.StaticAnalysisProfileData) {
	// Find the highest used counter ID and add ours on top.
	counterOffset := uint32(0)
	for _, counter := range pd.Counters {
		if counter.Id > counterOffset {
			counterOffset = counter.Id
		}
	}
	counterOffset += 1

	// Add the counter definitions.
	for _, counter := range staticAnalysis.CounterSpecs {
		pd.Counters = append(pd.Counters, &service.ProfilingData_Counter{
			Id:          counterOffset + counter.ID,
			Name:        counter.Name,
			Description: counter.Description,
			Unit:        counter.Unit,
		})
		var counterType service.ProfilingData_GpuCounters_Metric_Type
		switch counter.Type {
		case api.CounterType_Ranged:
			counterType = service.ProfilingData_GpuCounters_Metric_StaticAnalysisRanged
		case api.CounterType_Summed:
			counterType = service.ProfilingData_GpuCounters_Metric_StaticAnalysisSummed
		}

		pd.GpuCounters.Metrics = append(pd.GpuCounters.Metrics, &service.ProfilingData_GpuCounters_Metric{
			Id:          counterMetricIdOffset + int32(counterOffset+counter.ID),
			CounterId:   counterOffset + counter.ID,
			Name:        counter.Name,
			Unit:        counter.Unit,
			Op:          service.ProfilingData_GpuCounters_Metric_TimeWeightedAvg,
			Description: counter.Description,
			Type:        counterType,
		})
	}

	// Add in our groups and samples.
	for _, samples := range staticAnalysis.CounterData {
		group := pd.Groups.GetOrCreateDrawCallGroup("draw call", samples.Index)
		if group < 0 {
			// Draw call for which we didn't see a render pass in the data. Ignore for now.
			// TODO(pmuetschard): This shouldn't really happen, so at this point, trying to find the
			// correct command mapping for the draw call's renderpass, command buffer, and submit is not
			// worth it. I expect that with the scheduled command/sub command clean-up this may go away,
			// as we should be able to inferr the correct index.
			log.W(ctx, "Got a draw call with static analysis profiling data, but no GPU data: %v", samples.Index)
			continue
		}

		values := map[int32]*service.ProfilingData_GpuCounters_Perf{}
		for _, sample := range samples.Samples {
			values[counterMetricIdOffset+int32(counterOffset+sample.Counter)] = &service.ProfilingData_GpuCounters_Perf{
				Estimate: sample.Value,
				Min:      sample.Value,
				Max:      sample.Value,
			}
		}
		pd.GpuCounters.Entries = append(pd.GpuCounters.Entries, &service.ProfilingData_GpuCounters_Entry{
			GroupId:       group,
			MetricToValue: values,
		})
	}
}
