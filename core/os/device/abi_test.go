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

package device_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/device"
)

func TestAndroidABIByName(t *testing.T) {
	assert := assert.To(t)
	abi := device.AndroidABIByName("invalid")
	assert.For("ABI.Name").That(abi.Name).Equals("invalid")
	assert.For("ABI.Architecture").That(abi.Architecture).Equals(device.UnknownArchitecture)
	assert.For("ABI.OS").That(abi.OS).Equals(device.UnknownOS)
}
