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

package vulkan

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/service/path"
)

// StaticAnalysisProfileData is the result of the profiling static analysis.
// TODO(pmuetschard): this is a temporary struct here, that will be replaced once the
// service.ProfilingData is cleaned up/refactored for the new PerfTab.
type StaticAnalysisProfileData struct {
	CounterSpecs []CounterSpec
	CounterData  []CounterData
}

type CounterSpec struct {
	ID          uint32
	Name        string
	Description string
	Unit        string // this should match the unit from the Perfetto data.
}

type CounterData struct {
	Index   api.SubCmdIdx
	Samples []struct {
		Counter uint32
		Value   float64
	}
}

type counterType uint32

const (
	vertexCount counterType = iota
)

var (
	counters = map[counterType]CounterSpec{
		vertexCount: {
			ID:          uint32(vertexCount),
			Name:        "Vertex Count",
			Description: "The number of input vertices processed.",
			Unit:        "25", // VERTEX
		},
	}
)

type profilingDataBuilder struct {
	data         *StaticAnalysisProfileData
	seenCounters map[counterType]struct{}
}

// ProfileStaticAnalysis computes the static analysis profiling data. It processes each command
// and for each submitted draw call command, it computes statistics to be shown as counters in
// PerfTab, combined with the hardware counters.
func (API) ProfileStaticAnalysis(ctx context.Context, p *path.Capture) (interface{}, error) {
	data := profilingDataBuilder{
		data:         &StaticAnalysisProfileData{},
		seenCounters: map[counterType]struct{}{},
	}
	postSubCmdCb := func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, ref interface{}) {
		state := GetState(s)
		cmdRef := ref.(CommandReferenceʳ)
		cmdArgs := GetCommandArgs(ctx, cmdRef, state)

		switch args := cmdArgs.(type) {
		case VkCmdDrawArgsʳ:
			sampler := data.newSampler(idx)
			data.addSample(sampler, vertexCount, float64(args.InstanceCount()*args.VertexCount()))
		case VkCmdDrawIndexedArgsʳ:
			sampler := data.newSampler(idx)
			data.addSample(sampler, vertexCount, float64(args.InstanceCount()*args.IndexCount()))
		case VkCmdDrawIndexedIndirectArgsʳ:
		case VkCmdDrawIndirectArgsʳ:
		case VkCmdDrawIndirectCountKHRArgsʳ:
		case VkCmdDrawIndexedIndirectCountKHRArgsʳ:
		case VkCmdDrawIndirectCountAMDArgsʳ:
		case VkCmdDrawIndexedIndirectCountAMDArgsʳ:
		}
	}

	if err := sync.MutateWithSubcommands(ctx, p, nil, nil, nil, postSubCmdCb); err != nil {
		return nil, err
	}
	return data.data, nil
}

func (db profilingDataBuilder) newSampler(idx api.SubCmdIdx) *CounterData {
	db.data.CounterData = append(db.data.CounterData, CounterData{Index: idx})
	return &db.data.CounterData[len(db.data.CounterData)-1]
}

func (db profilingDataBuilder) addSample(sampler *CounterData, counter counterType, value float64) {
	if _, ok := db.seenCounters[counter]; !ok {
		db.seenCounters[counter] = struct{}{}
		db.data.CounterSpecs = append(db.data.CounterSpecs, counters[counter])
	}

	sampler.Samples = append(sampler.Samples, struct {
		Counter uint32
		Value   float64
	}{uint32(counter), value})
}
