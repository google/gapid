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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
)

const (
	sendDevInfoAction    = "com.google.android.gapid.action.SEND_DEV_INFO"
	sendDevInfoService   = "com.google.android.gapid.DeviceInfoService"
	sendDevInfoPort      = "gapid-devinfo"
	startServiceAttempts = 3
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
	var se error
	var le error
	var listening bool

	action := apk.ServiceActions.FindByName(sendDevInfoAction, sendDevInfoService)
	if action == nil {
		return log.Err(ctx, nil, "Service intent was not found")
	}

	// Start a fresh run.
	if err := d.ForceStop(ctx, apk.InstalledPackage.Name); err != nil {
		return log.Errf(ctx, err, "Can not stop package %s...", apk.InstalledPackage.Name)
	}

	// Try to start service.
	for i := 0; i < startServiceAttempts; i++ {
		time.Sleep(time.Second)
		log.I(ctx, "Attempt to start service: %s", sendDevInfoService)
		if se = d.StartService(ctx, *action); se != nil {
			continue
		}
		listening, le = devInfoPortListening(ctx, d)
		if le != nil {
			continue
		}
		if listening {
			// Services started and port is listening to incoming connections.
			return nil
		}
	}
	if se != nil {
		return se
	}
	if le != nil {
		return le
	}
	return log.Errf(ctx, nil, "Run out of attempts: %v", startServiceAttempts)
}

// Checks the existence of VkGraphicsSpyLayer in the debug.vulkan.layers system
// property. If found, strip the layer, otherwise keep the property unchanged.
// Returns a property value recover callback and error.
func stripVkGraphicsSpyLayer(ctx context.Context, d adb.Device) (func(), error) {
	const propName = "debug.vulkan.layers"
	const layerName = "VkGraphicsSpy"
	ctx = log.Enter(ctx, "stripVkGraphicsSpyLayer")
	log.I(ctx, "Check the existence of %s in %s", layerName, propName)
	old_layer_str, err := d.SystemProperty(ctx, propName)
	if err != nil {
		return nil, log.Errf(ctx, err, "Getting %s.", propName)
	}
	should_strip := false
	old_layers := strings.Split(old_layer_str, ":")
	var new_layer_str string
	for i, l := range old_layers {
		if strings.TrimSpace(l) == layerName {
			should_strip = true
			continue
		}
		new_layer_str += l
		if i+1 != len(old_layers) {
			new_layer_str += ":"
		}
	}
	if should_strip {
		log.I(ctx, "%s layer does exist, set %s to new value: %s", layerName, propName, new_layer_str)
		if err := d.SetSystemProperty(ctx, propName, new_layer_str); err != nil {
			return nil, log.Errf(ctx, err, "Setting %s to %s", propName, new_layer_str)
		}
		return func() { d.SetSystemProperty(ctx, propName, old_layer_str) }, nil
	}
	return nil, nil
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

	// Remove VkGraphicsSpy in the debug.vulkan.layers property to avoid loading
	// our spy layer.
	recover, err := stripVkGraphicsSpyLayer(ctx, d)
	if err != nil {
		log.W(ctx, err.Error())
		return err
	}
	if recover != nil {
		defer recover()
	}

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

	if err := proto.UnmarshalMerge(data, d.Instance()); err != nil {
		return log.Err(ctx, err, "Unmarshalling device Instance")
	}

	return nil
}
