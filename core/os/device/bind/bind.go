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

package bind

import (
	"context"
	"sync"

	"github.com/google/gapid/core/os/device"
)

var (
	hostMutex sync.Mutex
	host      Device
)

// Host returns the Device to the host.
func Host(ctx context.Context) Device {
	hostMutex.Lock()
	defer hostMutex.Unlock()
	if host == nil {
		host = &Simple{
			To: device.Host(ctx),
		}
	}
	return host
}
