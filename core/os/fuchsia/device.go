// Copyright (C) 2021 Google Inc.
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

package fuchsia

import (
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/shell"
)

// Device extends the bind.Device interface with capabilities specific to Fuchsia devices.
type Device interface {
	bind.Device
	// Command is a helper that builds a shell.Cmd with the device as its target.
	Command(name string, args ...string) shell.Cmd
}
