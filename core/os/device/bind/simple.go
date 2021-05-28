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
	"errors"
	"fmt"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/perfetto"
)

// Simple is a very short implementation of the Device interface.
// It directly holds the devices Information struct, and its last known Status, but provides no other active
// functionality. It can be used for fake devices, or as a building block to create a more complete device.
type Simple struct {
	To         *device.Instance
	LastStatus Status
}

func (b *Simple) String() string {
	if len(b.To.Name) > 0 {
		return b.To.Name
	}
	return b.To.Serial
}

// CanTrace returns true if this device can be used to take a trace
func (b *Simple) CanTrace() bool { return true }

// Instance implements the Device interface returning the Information in the To field.
func (b *Simple) Instance() *device.Instance { return b.To }

// Status implements the Device interface returning the Status from the LastStatus field.
func (b *Simple) Status(ctx context.Context) Status { return b.LastStatus }

// SupportsPerfetto returns true if this device will work with perfetto.
func (b *Simple) SupportsPerfetto(ctx context.Context) bool {
	return false
}

// ConnectPerfetto connects to a Perfetto service running on this device
// and returns an open socket connection to the service.
func (b *Simple) ConnectPerfetto(ctx context.Context) (*perfetto.Client, error) {
	return nil, fmt.Errorf("Perfetto is not supported on this device")
}

// ABI implements the Device interface returning the first ABI from the Information, or UnknownABI if it has none.
func (b *Simple) ABI() *device.ABI {
	if len(b.To.Configuration.ABIs) <= 0 {
		return device.UnknownABI
	}
	return b.To.Configuration.ABIs[0]
}

// InstallApp implements the Device interface, always returning an error.
func (b *Simple) InstallApp(ctx context.Context, app string) error {
	return errors.New("Installing applications is not supported on this device")
}
