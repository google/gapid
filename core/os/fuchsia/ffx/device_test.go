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
	"fmt"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/fuchsia/ffx"
)

const deviceJSONPattern = `{
	"nodename": "%s",
	"rcs_state": "%s",
	"serial": "<unknown>",
	"target_type": "Unknown",
	"target_state": "Product",
	"addresses":["%s"]
}
`

func TestParseDevices(t_ *testing.T) {
	ctx := log.Testing(t_)

	// Concatenate IP addresses and device names together as a facsimile of the real stdout from ffx
	// with 3 available devices.
	ipAddrs := []string{"fe80::5054:ff:fe63:5e7a%1", "fe80::5054:ff:fe63:5e7a%1", "fe80::5054:ff:fe63:5e7a%1"}
	deviceNames := []string{"fuchsia-5254-0063-5e7a", "fuchsia-5254-0063-5e7b", "fuchsia-5254-0063-5e7c"}

	var deviceJSON []string
	for i := range ipAddrs {
		deviceJSON = append(deviceJSON, fmt.Sprintf(deviceJSONPattern, deviceNames[i], "Y", ipAddrs[i]))
	}

	deviceMap, err := ffx.ParseDevices(ctx, "["+strings.Join(deviceJSON, ",")+"]")
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

	deviceMap, err = ffx.ParseDevices(ctx, "\nFile not found.\n\n")
	assert.For(ctx, "Garbage input").ThatError(err).Failed()
	assert.For(ctx, "Garbage input").ThatSlice(deviceMap).IsEmpty()
}
