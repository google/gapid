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

package adb_test

import (
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/shell/stub"
)

func TestParseDevices(t_ *testing.T) {
	ctx := log.Testing(t_)
	defer func() { devices.Handlers[0] = validDevices }()
	expected := &device.Instance{
		Serial: "production_device",
		Name:   "flame",
		Configuration: &device.Configuration{
			OS: &device.OS{
				Kind:         device.Android,
				Name:         "Android 10",
				Build:        "flame-user 10 QQ1A.191003.005 5926727 release-keys",
				MajorVersion: 10,
				MinorVersion: 0,
				PointVersion: 0,
				APIVersion:   29,
			},
			Hardware: &device.Hardware{
				Name: "flame",
			},
			ABIs: []*device.ABI{device.AndroidARM64v8a},
		},
	}
	expected.GenID()
	got, err := adb.Devices(ctx)
	assert.For(ctx, "Normal devices").ThatError(err).Succeeded()
	assert.For(ctx, "Normal devices").That(got.FindBySerial(expected.Serial).Instance()).DeepEquals(expected)

	devices.Handlers[0] = emptyDevices
	got, err = adb.Devices(ctx)
	assert.For(ctx, "Empty devices").ThatError(err).Succeeded()
	assert.For(ctx, "Empty devices").ThatSlice(got).IsEmpty()

	devices.Handlers[0] = invalidDevices
	_, err = adb.Devices(ctx)
	assert.For(ctx, "Invalid devices").ThatError(err).HasCause(adb.ErrInvalidDeviceList)

	devices.Handlers[0] = invalidStatus
	_, err = adb.Devices(ctx)
	assert.For(ctx, "Invalid status").ThatError(err).HasCause(adb.ErrInvalidStatus)

	devices.Handlers[0] = notDevices
	_, err = adb.Devices(ctx)
	assert.For(ctx, "Not devices").ThatError(err).HasCause(adb.ErrNoDeviceList)

	devices.Handlers[0] = &stub.Response{WaitErr: fmt.Errorf("Not connected")}
	_, err = adb.Devices(ctx)
	assert.For(ctx, "not connected").ThatError(err).HasMessage(`Process returned error
   Cause: Not connected`)
}

func TestLimitToSerial(t_ *testing.T) {
	ctx := log.Testing(t_)

	// Make sure to remove the limitation so further tests work fine
	defer adb.LimitToSerial("")

	got, err := adb.Devices(ctx)
	assert.For(ctx, "No serial limitation").ThatError(err).Succeeded()
	assert.For(ctx, "No serial limitation").ThatInteger(len(got)).IsAtLeast(2)

	adb.LimitToSerial("serial_do_match")
	got, err = adb.Devices(ctx)
	assert.For(ctx, "LimitToSerial do match").ThatError(err).Succeeded()
	assert.For(ctx, "LimitToSerial do match").ThatInteger(len(got)).Equals(1)
	assert.For(ctx, "LimitToSerial do match").ThatString(got.FindBySerial("serial_do_match").Instance().Serial).Equals("serial_do_match")

	adb.LimitToSerial("serial_do_not_match")
	got, err = adb.Devices(ctx)
	assert.For(ctx, "LimitToSerial do not match").ThatError(err).Succeeded()
	assert.For(ctx, "LimitToSerial do not match").ThatSlice(got).IsEmpty()
}
