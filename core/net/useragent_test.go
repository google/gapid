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

package net

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

var (
	win7 = &device.Configuration{
		OS: &device.OS{
			Kind:  device.Windows,
			Major: 6, Minor: 1, Point: 5,
		},
	}
	win10 = &device.Configuration{
		OS: &device.OS{
			Kind:  device.Windows,
			Major: 10, Minor: 0, Point: 5,
		},
	}
	macOS = &device.Configuration{
		OS: &device.OS{
			Kind:  device.OSX,
			Major: 10, Minor: 12, Point: 6,
		},
	}
	linux = &device.Configuration{
		OS: &device.OS{
			Kind:  device.Linux,
			Major: 1, Minor: 2, Point: 3,
		},
	}
)

func TestUseragent(t *testing.T) {
	ctx := log.Testing(t)
	version := ApplicationInfo{"GAPID", 1, 2, 3}
	for _, test := range []struct {
		name     string
		cfg      *device.Configuration
		expected string
	}{
		{"win7", win7, "GAPID/1.2.3 (Windows NT 6.1)"},
		{"win10", win10, "GAPID/1.2.3 (Windows NT 10.0)"},
		{"macOS", macOS, "GAPID/1.2.3 (Macintosh; Intel Mac OS X 10_12_6)"},
		{"linux", linux, "GAPID/1.2.3 (Linux)"},
	} {
		assert.For(ctx, test.name).ThatString(UserAgent(test.cfg, version)).Equals(test.expected)
	}
}
