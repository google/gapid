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
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/shell"
)

// Simple is a very short implementation of the Device interface.
// It directly holds the devices Information struct, and it's last known Status, but provides no other active
// functionality. It can be used for fake devices, or as a building block to create a more complete device.
type Simple struct {
	To         *device.Instance
	LastStatus Status
}

const (
	// ErrShellNotSupported may be returned by Start if the target does not support a shell.
	ErrShellNotSupported = fault.Const("bind.Simple does not support shell commands")
)

func (b *Simple) String() string {
	if len(b.To.Name) > 0 {
		return b.To.Name
	}
	return b.To.Serial
}

// Instance implements the Device interface returning the Information in the To field.
func (b *Simple) Instance() *device.Instance { return b.To }

// Status implements the Device interface returning the Status from the LastStatus field.
func (b *Simple) Status() Status { return b.LastStatus }

// Shell implements the Device interface returning commands that will error if run.
func (b *Simple) Shell(name string, args ...string) shell.Cmd {
	return shell.Command(name, args...).On(simpleTarget{})
}

// ABI implements the Device interface returning the first ABI from the Information, or UnknownABI if it has none.
func (b *Simple) ABI() *device.ABI {
	if len(b.To.Configuration.ABIs) <= 0 {
		return device.UnknownABI
	}
	return b.To.Configuration.ABIs[0]
}

type simpleTarget struct{}

func (t simpleTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	return nil, ErrShellNotSupported
}
