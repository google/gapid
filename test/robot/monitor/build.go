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

package monitor

import (
	"context"

	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/build"
)

// Package is the in memory representation/wrapper for a build.Package
type Package struct {
	build.Package
	parent *Package
}

// Packages is the type that manages a set of Package objects.
type Packages struct {
	entries []*Package
}

// Track is the in memory representation/wrapper for a build.Track
type Track struct {
	build.Track
	head *Package
}

// Tracks is the type that manages a set of Track objects.
type Tracks struct {
	entries []*Track
}

// All returns the complete set of Package objects we have seen so far.
func (p *Packages) All() []*Package {
	return p.entries
}

// All returns the complete set of Track objects we have seen so far.
func (t *Tracks) All() []*Track {
	return t.entries
}

// FindTools returns the tool set that matches the supplied device, if the package has one.
func (p *Package) FindTools(ctx context.Context, d *Device) *build.ToolSet {
	if p == nil || d == nil {
		return nil
	}
	for _, abi := range d.Information.Configuration.ABIs {
		if tools := p.Package.GetHostTools(abi); tools != nil {
			return tools
		}
	}
	return nil
}

// FindToolsForAPK returns the best matching tool set for a certain apk on a device,
// if present in the package.
func (p *Package) FindToolsForAPK(ctx context.Context, host *Device, target *Device, apkInfo *apk.Information) *build.AndroidToolSet {
	targetToolsAbi := target.GetInformation().GetConfiguration().PreferredABI(apkInfo.ABI)
	if targetToolsAbi == nil {
		return nil
	}
	for _, abi := range host.Information.Configuration.ABIs {
		if tools := p.Package.GetTargetTools(abi, targetToolsAbi); tools != nil {
			return tools
		}
	}
	return nil
}

// FindToolsForDevice returns the best matching tool set for a certain device.
func (p *Package) FindToolsForDevice(ctx context.Context, host *Device, target *Device) *build.AndroidToolSet {
	targetToolsAbi := target.GetInformation().GetConfiguration().PreferredABI([]*device.ABI{})
	if targetToolsAbi == nil {
		return nil
	}
	for _, abi := range host.Information.Configuration.ABIs {
		if tools := p.Package.GetTargetTools(abi, targetToolsAbi); tools != nil {
			return tools
		}
	}
	return nil
}

func (o *DataOwner) updateTrack(ctx context.Context, track *build.Track) error {
	o.Write(func(data *Data) {
		for i, e := range data.Tracks.entries {
			if track.Id == e.Id {
				data.Tracks.entries[i].Track = *track
				return
			}
		}
		data.Tracks.entries = append(data.Tracks.entries, &Track{Track: *track})
	})
	return nil
}

func (o *DataOwner) updatePackage(ctx context.Context, pkg *build.Package) error {
	o.Write(func(data *Data) {
		for i, e := range data.Packages.entries {
			if pkg.Id == e.Id {
				data.Packages.entries[i].Package = *pkg
				return
			}
		}
		data.Packages.entries = append(data.Packages.entries, &Package{Package: *pkg})
	})
	return nil
}
