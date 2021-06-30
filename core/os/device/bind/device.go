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

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/gapis/perfetto"
)

type Status int

const (
	UnknownStatus Status = iota
	Offline
	Online
	Unauthorized
)

// Device represents a connection to an attached device.
type Device interface {
	// Instance returns the instance information for this device.
	Instance() *device.Instance
	// State returns the last known connected status of the device.
	Status(ctx context.Context) Status
	// CanTrace returns true if this device can be used to take a trace
	CanTrace() bool
	// SupportsPerfetto returns true if this device will work with perfetto
	SupportsPerfetto(ctx context.Context) bool
	// ConnectPerfetto connects to a Perfetto service running on this device
	// and returns an open socket connection to the service.
	ConnectPerfetto(ctx context.Context) (*perfetto.Client, error)
	// InstallApp installs the given application to this device
	InstallApp(ctx context.Context, app string) error
}

type DeviceWithShell interface {
	Device
	// Shell is a helper that builds a shell.Cmd with d.ShellTarget() as its target
	Shell(name string, args ...string) shell.Cmd
}
