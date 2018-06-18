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

// OSXOS converts from an OSX version to a full OS structure.
func OSXOS(major, minor, point int32) *OS {
	var name string
	switch {
	case major == 10 && minor == 11:
		name = "El Capitan"
	case major == 10 && minor == 10:
		name = "Yosemite"
	case major == 10 && minor == 9:
		name = "Mavericks"
	case major == 10 && minor == 8:
		name = "Mountain Lion"
	case major == 10 && minor == 7:
		name = "Lion"
	case major == 10 && minor == 6:
		name = "Snow Leopard"
	case major == 10 && minor == 5:
		name = "Leopard"
	case major == 10 && minor == 4:
		name = "Tiger"
	case major == 10 && minor == 3:
		name = "Panther"
	case major == 10 && minor == 2:
		name = "Jaguar"
	case major == 10 && minor == 1:
		name = "Puma"
	default:
		name = "OSX"
	}
	return &OS{
		Kind:         OSX,
		Build:        fmt.Sprintf("%s %d.%d.%d", name, major, minor, point),
		Name:         name,
		MajorVersion: major,
		MinorVersion: minor,
		PointVersion: point,
	}
}
