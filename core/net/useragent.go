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
	"fmt"
	"strings"

	"github.com/google/gapid/core/os/device"
)

// ApplicationInfo describes an application.
type ApplicationInfo struct {
	Name         string
	VersionMajor int
	VersionMinor int
	VersionPoint int
}

// UserAgent returns a useragent string for the given device and application
// info.
func UserAgent(d *device.Configuration, ai ApplicationInfo) string {
	product := fmt.Sprintf("%v/%v.%v.%v", ai.Name, ai.VersionMajor, ai.VersionMinor, ai.VersionPoint)
	info := []string{}
	os := d.GetOS()
	switch os.GetKind() {
	case device.Windows:
		info = append(info, fmt.Sprintf("Windows NT %v.%v", os.MajorVersion, os.MinorVersion))
		if d.GetHardware().GetCPU().GetArchitecture().Bitness() == 64 {
			info = append(info, "x64")
		}
	case device.OSX:
		info = append(info, "Macintosh", fmt.Sprintf("Intel Mac OS X %v_%v_%v", os.MajorVersion, os.MinorVersion, os.PointVersion))

	case device.Linux:
		info = append(info, "Linux")

	case device.Android:
		info = append(info, "Linux", "U", fmt.Sprintf("Android %v.%v.%v", os.MajorVersion, os.MinorVersion, os.PointVersion))
		if os.Build != "" {
			info = append(info, "Build/"+os.Build)
		}
	}
	if len(info) > 0 {
		return fmt.Sprintf("%v (%v)", product, strings.Join(info, "; "))
	}
	return product
}
