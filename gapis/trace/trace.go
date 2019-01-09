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
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace/tracer"
)

func Trace(ctx context.Context, device *path.Device, start task.Signal, options *service.TraceOptions, written *int64) error {
	gapiiOpts := tracer.GapiiOptions(options)
	var process tracer.Process
	cleanup := func() {}
	mgr := GetManager(ctx)
	if device == nil {
		return log.Errf(ctx, nil, "Invalid device path")
	}
	tracer, ok := mgr.tracers[device.ID.ID()]
	if !ok {
		return log.Errf(ctx, nil, "Could not find tracer for device %d", device.ID.ID())
	}
	config, err := tracer.TraceConfiguration(ctx)
	if err != nil {
		return err
	}

	if port := options.GetPort(); port != 0 {
		if !config.ServerLocalPath {
			return log.Errf(ctx, nil, "Cannot attach to a remote device by port")
		}
		process = &gapii.Process{Port: int(port), Device: tracer.GetDevice(), Options: gapiiOpts}
	} else {
		process, cleanup, err = tracer.SetupTrace(ctx, options)
	}

	if err != nil {
		return log.Errf(ctx, err, "Could not start trace")
	}
	defer cleanup()

	os.MkdirAll(filepath.Dir(options.ServerLocalSavePath), 0755)
	file, err := os.Create(options.ServerLocalSavePath)
	if err != nil {
		return err
	}

	defer file.Close()

	if options.Duration > 0 {
		ctx, _ = task.WithTimeout(ctx, time.Duration(options.Duration)*time.Second)
	}

	_, err = process.Capture(ctx, start, file, written)
	return err
}

func TraceConfiguration(ctx context.Context, device *path.Device) (*service.DeviceTraceConfiguration, error) {
	mgr := GetManager(ctx)
	tracer, ok := mgr.tracers[device.ID.ID()]
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find tracer for device %d", device.ID.ID())
	}

	return tracer.TraceConfiguration(ctx)
}
