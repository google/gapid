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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

type validateGpuProfilingVerb struct{ ValidateGpuProfilingFlags }

func init() {
	verb := &validateGpuProfilingVerb{}

	app.AddVerb(&app.Verb{
		Name:      "validate_gpu_profiling",
		ShortHelp: "Validates the GPU profiling capability of a device",
		Action:    verb,
	})
}

func (verb *validateGpuProfilingVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server.")
	}
	defer client.Close()
	devices, err := filterDevices(ctx, &verb.DeviceFlags, client)
	if err != nil {
		return log.Err(ctx, err, "Failed to get device list.")
	}
	stdout := os.Stdout
	someDeviceFailed := false
	for i, p := range devices {
		fmt.Fprintf(stdout, "-- Device %v: %v --\n", i, p.ID.ID())
		res, err := client.ValidateDevice(ctx, p)
		if err != nil {
			fmt.Fprintf(stdout, "%v\n", log.Errf(ctx, err, "Failed to start device validation: %s", err))
			someDeviceFailed = true
			continue
		} else if len(res.ValidationFailureMsg) > 0 {
			fmt.Fprintf(stdout, "%v\n", log.Errf(ctx, nil, "Device validation failed: %s, trace file: %s", res.ValidationFailureMsg, res.TracePath))
			someDeviceFailed = true
			continue
		}
		fmt.Fprintf(stdout, "Device is validated.\n")
	}
	if someDeviceFailed {
		return errors.New("Some device failed validation")
	}
	return nil
}
