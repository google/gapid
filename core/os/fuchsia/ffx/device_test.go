// Copyright (C) 2021 Google Inc.
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

package ffx_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/fuchsia/ffx"
)

func TestParseDevices(t_ *testing.T) {
	ctx := log.Testing(t_)

	// Concatenate IP addresses and device names together as a facsimile of the real stdout from ffx
	// with 3 available devices.
	ipAddrs := []string{"fe80::5054:ff:fe63:5e7a%1", "fe80::5054:ff:fe63:5e7a%1", "fe80::5054:ff:fe63:5e7a%1"}
	deviceNames := []string{"fuchsia-5254-0063-5e7a", "fuchsia-5254-0063-5e7b", "fuchsia-5254-0063-5e7c"}

	var devicesStdOut string
	for i := range ipAddrs {
		devicesStdOut += ipAddrs[i]
		devicesStdOut += " "
		devicesStdOut += deviceNames[i]
		devicesStdOut += "\n"
	}

	deviceMap, err := ffx.ParseDevices(ctx, devicesStdOut)
	if assert.For(ctx, "Valid devices").ThatError(err).Succeeded() {
		devicesFound := 0
		for deviceName := range deviceMap {
			for _, currDeviceName := range deviceNames {
				if currDeviceName == deviceName {
					devicesFound++
					break
				}
			}
		}
		assert.For(ctx, "Valid devices").ThatInteger(devicesFound).Equals(len(ipAddrs))
	}

	// Verify empty device list.
	devicesStdOut = "\nNo devices found.\n\n"
	deviceMap, err = ffx.ParseDevices(ctx, devicesStdOut)
	assert.For(ctx, "Empty devices").ThatError(err).Succeeded()
	assert.For(ctx, "Empty devices").ThatSlice(deviceMap).IsEmpty()

	// Verify error state with garbage input.
	devicesStdOut = "\nFile not found.\n\n"
	deviceMap, err = ffx.ParseDevices(ctx, devicesStdOut)
	assert.For(ctx, "Garbage input").ThatError(err).Failed()
	assert.For(ctx, "Garbage input").ThatSlice(deviceMap).IsEmpty()
}
