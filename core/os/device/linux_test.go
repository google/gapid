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

func TestLinuxOS(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		major int32
		minor int32
		name  string
	}{
		{major: 1, minor: 3, name: "Ubuntu"},
	} {
		os := device.LinuxOS(test.name, test.major, test.minor)
		assert.For("OS Kind").That(os.Kind).Equals(device.Linux)
		assert.For("OS Name").That(os.Name).Equals(test.name)
		assert.For("OS Build").That(os.Build).Equals("")
		assert.For("OS Major").That(os.MajorVersion).Equals(test.major)
		assert.For("OS Minor").That(os.MinorVersion).Equals(test.minor)
		assert.For("OS Point").That(os.PointVersion).Equals(int32(0))
	}
}
