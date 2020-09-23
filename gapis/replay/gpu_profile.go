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

package replay

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace"

	perfetto_pb "protos/perfetto/config"
)

// Eyeball some generous trace config parameters
const (
	counterPeriodNs                         = uint64(1000000)
	bufferSizeKb                            = uint32(256 * 1024)
	durationMs                              = 30000
	gpuCountersDataSourceDescriptorName     = "gpu.counters"
	gpuRenderStagesDataSourceDescriptorName = "gpu.renderstages"
)

func getPerfettoConfig(ctx context.Context, device *path.Device) (*perfetto_pb.TraceConfig, error) {
	t, err := trace.GetTracer(ctx, device)
	if err != nil {
		err = log.Errf(ctx, err, "Failed to find tracer for %v", device)
		return nil, err
	}
	d := t.GetDevice()
	specs := d.Instance().GetConfiguration().GetPerfettoCapability().GetGpuProfiling().GetGpuCounterDescriptor().GetSpecs()
	ids := make([]uint32, len(specs))
	for i, s := range specs {
		ids[i] = s.GetCounterId()
	}
	conf := &perfetto_pb.TraceConfig{
		Buffers: []*perfetto_pb.TraceConfig_BufferConfig{
			{SizeKb: proto.Uint32(bufferSizeKb)},
		},
		DurationMs: proto.Uint32(durationMs),
		DataSources: []*perfetto_pb.TraceConfig_DataSource{
			{
				Config: &perfetto_pb.DataSourceConfig{
					Name: proto.String(gpuRenderStagesDataSourceDescriptorName),
				},
			},
			{
				Config: &perfetto_pb.DataSourceConfig{
					Name: proto.String(gpuCountersDataSourceDescriptorName),
					GpuCounterConfig: &perfetto_pb.GpuCounterConfig{
						CounterPeriodNs: proto.Uint64(counterPeriodNs),
						CounterIds:      ids,
					},
				},
			},
		},
	}
	return conf, nil
}

// GpuProfile replays the trace and writes a Perfetto trace of the replay
func GpuProfile(ctx context.Context, capturePath *path.Capture, device *path.Device, disabledCmds []*path.Command) (*service.ProfilingData, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return nil, err
	}

	if device != nil {
		intent := Intent{
			Capture: capturePath,
			Device:  device,
		}

		conf, err := getPerfettoConfig(ctx, device)
		if err != nil {
			return nil, err
		}

		opts := &service.TraceOptions{
			Device:         device,
			Type:           service.TraceType_Perfetto,
			PerfettoConfig: conf,
		}

		var disabledCmdsIndices [][]uint64
		if disabledCmds != nil {
			disabledCmdsIndices = make([][]uint64, 0, len(disabledCmds))
			for _, cmd := range disabledCmds {
				disabledCmdsIndices = append(disabledCmdsIndices, cmd.Indices)
			}
		}

		mgr := GetManager(ctx)
		hints := &path.UsageHints{Background: true}
		for _, a := range c.APIs {
			if pf, ok := a.(Profiler); ok {
				data, err := pf.QueryProfile(ctx, intent, mgr, hints, opts, disabledCmdsIndices)
				if err != nil {
					log.E(ctx, "Replay profiling failed.")
					return nil, err
				}
				log.I(ctx, "Replay profiling finished.")
				return data, nil
			}
		}
	} else {
		err = log.Err(ctx, nil, "Failed to find replay device.")
		return nil, err
	}
	err = log.Err(ctx, nil, "Failed to profile replay")
	return nil, err
}
