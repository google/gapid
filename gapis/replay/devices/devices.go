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

// Package devices contains functions for gathering devices that can replay a
// capture.
package devices

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service/path"
)

// ForReplay returns a priority-sorted path list of devices that are capable of
// replaying the capture c.
func ForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, p)
	if err != nil {
		return nil, err
	}

	apis := make([]replay.Support, 0, len(c.APIs))
	for _, i := range c.APIs {
		api := api.Find(api.ID(i.ID()))
		if f, ok := api.(replay.Support); ok {
			apis = append(apis, f)
		}
	}

	all := Sorted(ctx)
	filtered := make([]prioritizedDevice, 0, len(all))
	for _, device := range all {
		instance := device.Instance()
		p := uint32(1)
		for _, api := range apis {
			// TODO: Check if device is a LAD, and if so filter by supportsLAD.
			ctx := log.V{
				"api":    fmt.Sprintf("%T", api),
				"device": instance.Name,
			}.Bind(ctx)
			priority := api.GetReplayPriority(ctx, instance, c.Header)
			p = p * priority
			if priority != 0 {
				log.D(ctx, "Compatible %d", priority)
			} else {
				log.D(ctx, "Incompatible")
			}
		}
		if p > 0 {
			ctx := log.V{
				"device": instance,
			}.Bind(ctx)
			log.D(ctx, "Priority %d", p)
			filtered = append(filtered, prioritizedDevice{device, p})
		}
	}

	sort.Sort(prioritizedDevices(filtered))

	paths := make([]*path.Device, len(filtered))
	for i, d := range filtered {
		paths[i] = path.NewDevice(d.device.Instance().ID.ID())
	}
	return paths, nil
}

// Sorted returns all devices, sorted by Android first, and then Host.
func Sorted(ctx context.Context) []bind.Device {
	all := bind.GetRegistry(ctx).Devices()
	androidDevices := make([]bind.Device, 0, len(all))
	nonAndroidDevices := make([]bind.Device, 0, len(all))
	for _, dev := range all {
		instance := dev.Instance()
		if instance.GetConfiguration().GetOS().GetKind() == device.Android {
			androidDevices = append(androidDevices, dev)
		} else {
			nonAndroidDevices = append(nonAndroidDevices, dev)
		}
	}
	return append(androidDevices, nonAndroidDevices...)
}

type prioritizedDevice struct {
	device   bind.Device
	priority uint32
}

type prioritizedDevices []prioritizedDevice

func (p prioritizedDevices) Len() int {
	return len(p)
}

func (p prioritizedDevices) Less(i, j int) bool {
	return p[i].priority < p[j].priority
}

func (p prioritizedDevices) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
