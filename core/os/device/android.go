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

package device

import "fmt"

const AndroidMinimalSupportedAPIVersion = 23

// AndroidOS returns the full OS structure for the supplied android os version.
func AndroidOS(major, minor, point int32) *OS {
	os := &OS{
		Kind:         Android,
		MajorVersion: major,
		MinorVersion: minor,
		PointVersion: point,
	}
	switch {
	case major == 11:
		os.Name = "Android 11"
	case major == 10:
		os.Name = "Android 10"
	case major == 9:
		os.Name = "Pie"
	case major == 8:
		os.Name = "Oreo"
	case major == 7:
		os.Name = "Nougat"
	case major == 6:
		os.Name = "Marshmallow"
	case major == 5:
		os.Name = "Lollipop"
	case major == 4 && minor >= 4:
		os.Name = "KitKat"
	case major == 4 && minor >= 1:
		os.Name = "Jelly Bean"
	case major == 4:
		os.Name = "Ice Cream Sandwich"
	case major == 3:
		os.Name = "Honeycomb"
	case major == 2 && minor >= 3:
		os.Name = "Gingerbread"
	case major == 2 && minor >= 2:
		os.Name = "Froyo"
	case major == 2:
		os.Name = "Eclair"
	case major == 1 && minor >= 6:
		os.Name = "Donut"
	case major == 1:
		os.Name = "Cupcake"
	default:
		os.Name = fmt.Sprintf("Android %d.%d.%d", major, minor, point)
	}
	return os
}
