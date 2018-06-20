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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

func TestOSXOS(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		major int32
		minor int32
		point int32
		name  string
		build string
	}{
		{1, 3, 0, "OSX", "OSX 1.3.0"},
		{10, 11, 0, "El Capitan", "El Capitan 10.11.0"},
		{10, 10, 0, "Yosemite", "Yosemite 10.10.0"},
		{10, 9, 0, "Mavericks", "Mavericks 10.9.0"},
		{10, 8, 0, "Mountain Lion", "Mountain Lion 10.8.0"},
		{10, 7, 0, "Lion", "Lion 10.7.0"},
		{10, 6, 0, "Snow Leopard", "Snow Leopard 10.6.0"},
		{10, 5, 0, "Leopard", "Leopard 10.5.0"},
		{10, 4, 0, "Tiger", "Tiger 10.4.0"},
		{10, 3, 0, "Panther", "Panther 10.3.0"},
		{10, 2, 0, "Jaguar", "Jaguar 10.2.0"},
		{10, 1, 0, "Puma", "Puma 10.1.0"},
	} {
		ctx := log.Enter(ctx, test.name)
		os := device.OSXOS(test.major, test.minor, 0)
		assert.For(ctx, "Kind").That(os.Kind).Equals(device.OSX)
		assert.For(ctx, "Name").That(os.Name).Equals(test.name)
		assert.For(ctx, "Build").That(os.Build).Equals(test.build)
		assert.For(ctx, "Major").That(os.MajorVersion).Equals(test.major)
		assert.For(ctx, "Minor").That(os.MinorVersion).Equals(test.minor)
		assert.For(ctx, "Point").That(os.PointVersion).Equals(test.point)
	}
}
