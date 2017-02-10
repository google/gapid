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
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
)

const (
	sendDevInfoAction  = "com.google.android.gapid.action.SEND_DEV_INFO"
	sendDevInfoService = "com.google.android.gapid.DeviceInfoService"
	sendDevInfoPort    = "gapid-devinfo"
)

func init() {
	adb.RegisterDeviceInfoProvider(fetchDeviceInfo)
}

func fetchDeviceInfo(ctx log.Context, d adb.Device) error {
	apk, err := EnsureInstalled(ctx, d, device.UnknownABI)
	if err != nil {
		// The gapid.apk was not found. This can happen with partial builds used
		// for testing.
		// Don't return an error as this will prevent the device from being
		// registered and the device already comes with basic usable
		// information.
		jot.Warning(ctx).Cause(err).Print("Couldn't find gapid.apk for device")
		return nil
	}

	action := apk.ServiceActions.FindByName(sendDevInfoAction, sendDevInfoService)
	if action == nil {
		return cause.Explain(ctx, nil, "Service intent was not found")
	}

	if err := d.StartService(ctx, *action); err != nil {
		return cause.Explain(ctx, err, "Starting service")
	}

	sock, err := adb.ForwardAndConnect(ctx, d, sendDevInfoPort)
	if err != nil {
		return cause.Explain(ctx, err, "Connecting to service port")
	}

	defer sock.Close()

	data, err := ioutil.ReadAll(sock)
	if err != nil {
		return cause.Explain(ctx, err, "Reading data")
	}

	if err := proto.Unmarshal(data, d.Instance()); err != nil {
		return cause.Explain(ctx, err, "Unmarshalling device Instance")
	}

	return nil
}
