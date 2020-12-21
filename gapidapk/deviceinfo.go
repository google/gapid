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

package gapidapk

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/flock"
)

const (
	sendDevInfoAction     = "com.google.android.gapid.action.SEND_DEV_INFO"
	sendDevInfoService    = "com.google.android.gapid.DeviceInfoService"
	sendDevInfoPort       = "gapid-devinfo"
	startServiceAttempts  = 3
	portListeningAttempts = 5
)

func init() {
	adb.RegisterDeviceInfoProvider(fetchDeviceInfo)
}

// Returns true if the device is listening to sendDevInfoPort, false if not.
// Error if failed at getting the port info.
func devInfoPortListening(ctx context.Context, d adb.Device) (bool, error) {
	var stdout bytes.Buffer
	if err := d.Shell("cat", "/proc/net/unix").Capture(&stdout, nil).Run(ctx); err != nil {
		return false, log.Errf(ctx, err, "Getting unix abstract port info...")
	}
	if strings.Contains(stdout.String(), sendDevInfoPort) {
		return true, nil
	}
	return false, nil
}

// startDevInfoService tries to start the fresh run of the package and start
// the service to send device info.
func startDevInfoService(ctx context.Context, d adb.Device, apk *APK) error {
	ctx = log.Enter(ctx, "startDevInfoService")
	var listening bool

	action := apk.ServiceActions.FindByName(sendDevInfoAction, sendDevInfoService)
	if action == nil {
		return log.Err(ctx, nil, "Service intent was not found")
	}

	// Try to start service.
	err := task.Retry(ctx, startServiceAttempts, 100*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			log.I(ctx, "Attempt to start service: %s", sendDevInfoService)
			if err := d.StartService(ctx, *action); err != nil {
				return false, err
			}
			err := task.Retry(ctx, portListeningAttempts, time.Second, func(
				ctx context.Context) (bool, error) {
				var err error
				listening, err = devInfoPortListening(ctx, d)
				return listening, err
			})
			return listening, err
		})
	if listening {
		return nil
	}
	return log.Errf(ctx, err, "Start DevInfo service: Run out of attempts: %v",
		startServiceAttempts)
}

func fetchDeviceInfo(ctx context.Context, d adb.Device) error {
	apk, err := EnsureInstalled(ctx, d, device.UnknownABI)
	if err != nil {
		// The gapid.apk was not found. This can happen with partial builds used
		// for testing.
		// Don't return an error as this will prevent the device from being
		// registered and the device already comes with basic usable
		// information.
		log.W(ctx, "Couldn't find gapid.apk for device. Error: %v", err)
		return nil
	}

	// Close any previous runs of the apk
	apk.Stop(ctx)

	var cleanup app.Cleanup
	instance := d.Instance()

	packages := []string{}
	supported, packageName, nextCleanup, err := d.PrepareGpuProfiling(ctx, apk.InstalledPackage)
	cleanup = cleanup.Then(nextCleanup)
	if err != nil {
		cleanup.Invoke(ctx)
		return err
	}
	if packageName != "" {
		packages = append(packages, packageName)
	}

	if supported {
		// Set driver package
		nextCleanup, err := android.SetupLayers(ctx, d, apk.Name, packages, []string{})
		cleanup = cleanup.Then(nextCleanup)
		defer cleanup.Invoke(ctx)
		if err != nil {
			log.E(ctx, "Failed when settings up layers: %v", err)
			return err
		}

		// Start perfetto producer
		if instance.GetConfiguration().GetOS().GetAPIVersion() >= 29 {
			EnsurePerfettoProducerLaunched(ctx, d)
		}
	}

	// Make sure the device is available to query device info, this is to prevent
	// Vulkan trace from happening at the same time than device info query.
	m := flock.Lock(instance.GetSerial())
	defer m.Unlock()

	// Tries to start the device info service.
	if err := startDevInfoService(ctx, d, apk); err != nil {
		return log.Err(ctx, err, "Starting service")
	}

	sock, err := adb.ForwardAndConnect(ctx, d, sendDevInfoPort)
	if err != nil {
		return log.Err(ctx, err, "Connecting to service port")
	}

	defer sock.Close()

	data, err := ioutil.ReadAll(sock)
	if err != nil {
		return log.Err(ctx, err, "Reading data")
	}

	// Reset the ABIs, so we can properly merge them later.
	abis := instance.GetConfiguration().GetABIs()
	if len(abis) > 0 {
		instance.Configuration.ABIs = nil
	}
	if err := proto.UnmarshalMerge(data, instance); err != nil {
		return log.Err(ctx, err, "Unmarshalling device Instance")
	}

	// Merge ABIs by hand, since proto.Merge appends repeated fields.
	config := instance.GetConfiguration()
	for _, old := range abis {
		found := false
		for _, new := range config.ABIs {
			if old.Name == new.Name {
				proto.Merge(new, old)
				found = true
				break
			}
		}
		if !found {
			config.ABIs = append(config.ABIs, old)
		}
	}

	return nil
}
