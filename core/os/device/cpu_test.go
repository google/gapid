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

func TestCPUByName(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		name         string
		architecture device.Architecture
	}{
		{"Scorpion", device.ARMv7a},
		{"Krait", device.ARMv7a},
		{"Denver", device.ARMv8a},
		{"invalid", device.UnknownArchitecture},
	} {
		ctx := log.Enter(ctx, test.name)
		cpu := device.CPUByName(test.name)
		assert.For(ctx, "Name").That(cpu.Name).Equals(test.name)
		assert.For(ctx, "Architecture").That(cpu.Architecture).Equals(test.architecture)
	}
}
