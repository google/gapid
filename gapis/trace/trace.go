// Copyright (C) 2018 Google Inc.
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

package trace

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace/tracer"
)

func trace(ctx context.Context, device *path.Device, start task.Signal, stop task.Signal, ready task.Task, options *service.TraceOptions, written *int64, buffer *bytes.Buffer) error {
	gapiiOpts := tracer.GapiiOptions(options)
	var process tracer.Process
	var cleanup app.Cleanup

	t, err := GetTracer(ctx, device)
	if err != nil {
		return err
	}
	conf, err := t.TraceConfiguration(ctx)
	if err != nil {
		return err
	}

	if !isSupported(conf, options) {
		return log.Errf(ctx, nil, "Cannot take the requested type of trace on this device")
	}

	if port := options.GetPort(); port != 0 {
		if !conf.ServerLocalPath {
			return log.Errf(ctx, nil, "Cannot attach to a remote device by port")
		}
		process = &gapii.Process{Port: int(port), Device: t.GetDevice(), Options: gapiiOpts}
	} else {
		process, cleanup, err = t.SetupTrace(ctx, options)
	}

	if err != nil {
		return log.Errf(ctx, err, "Could not start trace")
	}
	defer cleanup.Invoke(ctx)

	var writer io.Writer
	if buffer != nil {
		writer = buffer
	} else {
		os.MkdirAll(filepath.Dir(options.ServerLocalSavePath), 0755)
		writer, err = os.Create(options.ServerLocalSavePath)
		if err != nil {
			return err
		}
		defer writer.(*os.File).Close()
	}

	_, err = process.Capture(ctx, start, stop, ready, writer, written)

	return err
}

func Trace(ctx context.Context, device *path.Device, start task.Signal, stop task.Signal, ready task.Task, options *service.TraceOptions, written *int64) error {
	return trace(ctx, device, start, stop, ready, options, written, nil)
}

func TraceBuffered(ctx context.Context, device *path.Device, start task.Signal, stop task.Signal, ready task.Task, options *service.TraceOptions, buffer *bytes.Buffer) error {
	var written int64 = 0
	err := trace(ctx, device, start, stop, ready, options, &written, buffer)
	if config.DumpReplayProfile {
		dumpTrace(ctx, buffer)
	}
	return err
}

func TraceConfiguration(ctx context.Context, device *path.Device) (*service.DeviceTraceConfiguration, error) {
	t, err := GetTracer(ctx, device)
	if err != nil {
		return nil, err
	}
	return t.TraceConfiguration(ctx)
}

func ProcessProfilingData(ctx context.Context, device *path.Device, capture *path.Capture,
	buffer *bytes.Buffer, staticAnalysisResult chan *api.StaticAnalysisProfileData,
	handleMapping map[uint64][]service.VulkanHandleMappingItem, syncData *sync.Data) (*service.ProfilingData, error) {

	t, err := GetTracer(ctx, device)
	if err != nil {
		return nil, err
	}
	return t.ProcessProfilingData(ctx, buffer, capture, staticAnalysisResult, handleMapping, syncData)
}

func Validate(ctx context.Context, device *path.Device) error {
	t, err := GetTracer(ctx, device)
	if err != nil {
		return err
	}
	return t.Validate(ctx)
}

func GetTracer(ctx context.Context, device *path.Device) (tracer.Tracer, error) {
	mgr := GetManager(ctx)
	if device == nil {
		return nil, log.Errf(ctx, nil, "Invalid device path")
	}
	t, ok := mgr.tracers[device.ID.ID()]
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find tracer for device %d", device.ID.ID())
	}
	return t, nil
}

func dumpTrace(ctx context.Context, buffer *bytes.Buffer) {
	dumpPath, err := filepath.Abs("./dump.perfetto")
	if err != nil {
		log.W(ctx, "Unable to resolve working directory")
	} else {
		dumpFile, err := os.Create(dumpPath)
		if err != nil {
			log.W(ctx, "Unable to create local trace file")
		} else {
			_, err = dumpFile.Write(buffer.Bytes())
			if err != nil {
				log.W(ctx, "Unable write local trace file")
			}
			log.I(ctx, "Saved %v", dumpPath)
		}
	}
}

func isSupported(config *service.DeviceTraceConfiguration, options *service.TraceOptions) bool {
	for _, c := range config.Types {
		if c.Type == options.Type {
			return true
		}
	}
	return false
}
