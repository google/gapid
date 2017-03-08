// Copyright (C) 2017 Google Inc.
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
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service/path"
)

func getGapis(ctx context.Context, gapisFlags GapisFlags, gapirFlags GapirFlags) (client.Client, error) {
	args := strings.Fields(gapisFlags.Args)
	if gapirFlags.Args != "" {
		// Pass the arguments for gapir further to gapis. Add flag to tag the
		// gapir argument string for gapis.
		args = append(args, "--gapir-args", gapirFlags.Args)
	}
	if gapisFlags.Profile != "" {
		args = append(args, "-cpuprofile", gapisFlags.Profile)
	}

	var token auth.Token
	if gapisFlags.Port == 0 {
		token = auth.GenToken()
	} else {
		token = auth.Token(gapisFlags.Token)
	}
	client, _, err := client.Connect(ctx, client.Config{
		Port:  gapisFlags.Port,
		Args:  args,
		Token: token,
	})
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	return client, nil
}

func getDevice(ctx context.Context, client client.Client, capture *path.Capture, flags GapirFlags) (*path.Device, error) {
	ctx = log.V{"device": flags.Device}.Bind(ctx)
	paths, err := client.GetDevicesForReplay(ctx, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed query list of devices")
	}

	if len(paths) > 0 {
		return paths[0], nil
	}

	return nil, log.Err(ctx, nil, "No compatible devices found")
}

func getADBDevice(ctx context.Context, pattern string) (adb.Device, error) {
	devices, err := adb.Devices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("No devices found")
	}
	log.I(ctx, "Device list:")
	for _, test := range devices {
		log.I(ctx, "  %v", test.Instance().Serial)
	}
	matchingDevices := []adb.Device{}
	if pattern == "" {
		matchingDevices = devices
	} else {
		re := regexp.MustCompile("(?i)" + pattern)
		for _, test := range devices {
			if re.MatchString(test.Instance().Serial) {
				matchingDevices = append(matchingDevices, test)
			}
		}
	}
	if len(matchingDevices) == 0 {
		return nil, fmt.Errorf("No devices matching %q found", pattern)
	} else if len(matchingDevices) > 1 {
		fmt.Fprintln(os.Stderr, "Matching devices:")
		for _, test := range matchingDevices {
			fmt.Fprint(os.Stderr, "    ")
			fmt.Fprintln(os.Stderr, test.Instance().Serial)
		}
		return nil, fmt.Errorf("Multiple devices matching %q found", pattern)
	}
	return matchingDevices[0], nil
}
