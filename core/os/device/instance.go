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

import (
	"fmt"

	"github.com/google/gapid/core/data/id"
)

// GenID assigns a new identifier to the instance from the serial, name and
// configuration.
func (i *Instance) GenID() {
	key := fmt.Sprintf("%v:%v:%v", i.Serial, i.Name, i.Configuration.String())
	i.ID = NewID(id.OfString(key))
}

// SameAs returns true if the two instance objects refer to the same physical
// device.
func (i *Instance) SameAs(o *Instance) bool {
	if i == o {
		// Same pointer must be same device
		return true
	}
	if i == nil || o == nil {
		// only one of them is nil no match
		return false
	}
	// If the serial is set, treat it as an authoratitive comparison point
	if i.Serial != "" && o.Serial != "" {
		return i.Serial == o.Serial
	}
	// If the hardware is different, cannot be the same
	if !i.Configuration.Hardware.SameAs(o.Configuration.Hardware) {
		return false
	}
	// If we get here, and the devices have names, we trust the names
	if i.Name != "" && o.Name != "" {
		return i.Name == o.Name
	}

	// If we get here, we have no strong information that the devices are either different
	// or the same, assume different for safty
	return false
}
