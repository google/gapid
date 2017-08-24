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

package host_test

import (
	"sync"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/host"
)

func TestHost(t *testing.T) {
	ctx := log.Testing(t)
	h := host.Instance(ctx)
	assert.For(ctx, "Host.ID").
		That(h.Id.ID()).NotEquals(id.ID{})
	assert.For(ctx, "Host.Name").
		That(h.Name).NotEquals("")
	assert.For(ctx, "Host.Configuration.OS.Kind").
		That(h.Configuration.OS.Kind).NotEquals(device.UnknownOS)
	assert.For(ctx, "Host.Configuration.Hardware.CPU").
		ThatString(h.Configuration.Hardware.CPU.Name).NotEquals("")
}

func TestHostConcurrent(t *testing.T) {
	ctx := log.Testing(t)
	hosts := make([]*device.Instance, 1000)
	wg := sync.WaitGroup{}
	for i := range hosts {
		i := i
		wg.Add(1)
		go func() {
			hosts[i] = host.Instance(ctx)
			wg.Done()
		}()
	}
	wg.Wait()
	for i, h := range hosts[1:] {
		assert.For(ctx, "i=%v", i).That(h).DeepEquals(hosts[0])
	}
}
