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
	"github.com/google/gapid/core/os/device/host"
)

func TestSamePhysicalDevice(t *testing.T) {
	ctx := log.Testing(t)
	a := host.Instance(ctx)
	var b *device.Instance
	c := a
	log.I(ctx, "%#+v %#+v", a, b)
	assert.For(ctx, "Device must be itself").That(a.SameAs(a)).Equals(true)
	assert.For(ctx, "Device must match a copy").That(a.SameAs(c)).Equals(true)
	assert.For(ctx, "Nil matches itself").That(b.SameAs(b)).Equals(true)
	assert.For(ctx, "Device must not match nil").That(a.SameAs(b)).Equals(false)
}
