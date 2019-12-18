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

	perfetto_pb "protos/perfetto/config"
)

// Eyeball some generous trace config parameters
const (
	bufferSizeKb                            = uint32(256 * 1024)
	durationMs                              = 30000
	gpuRenderStagesDataSourceDescriptorName = "gpu.renderstages"
)

// GpuProfile replays the trace and writes a Perfetto trace of the replay
func GpuProfile(ctx context.Context, capturePath *path.Capture, device *path.Device) (*service.ProfilingData, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return nil, err
	}

	if device != nil {
		intent := Intent{
			Capture: capturePath,
			Device:  device,
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
			},
		}

		opts := &service.TraceOptions{
			Device:         device,
			Type:           service.TraceType_Perfetto,
			PerfettoConfig: conf,
		}

		mgr := GetManager(ctx)
		hints := &service.UsageHints{Background: true}
		for _, a := range c.APIs {
			if pf, ok := a.(Profiler); ok {
				data, err := pf.Profile(ctx, intent, mgr, hints, opts, nil)
				if err != nil {
					log.E(ctx, "Replay profiling failed.")
					continue
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
