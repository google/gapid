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
)

// Device represents a connection to an attached device.
type Device interface {
	// Instance returns the instance information for this device.
	Instance() *device.Instance
	// State returns the last known connected status of the device.
	Status() Status
	// Shell is a helper that builds a shell.Cmd with d.ShellTarget() as its target
	Shell(name string, args ...string) shell.Cmd
	// TempFile creates a temporary file on the given Device. It returns the
	// path to the file, and a function that can be called to clean it up.
	TempFile(ctx context.Context) (string, func(ctx context.Context), error)
	// FileContents returns the contents of a given file on the Device.
	FileContents(ctx context.Context, path string) (string, error)
	// RemoveFile removes the given file from the device
	RemoveFile(ctx context.Context, path string) error
	// GetEnv returns the default environment for the Device.
	GetEnv(ctx context.Context) (*shell.Env, error)
	// SetupLocalPort makes sure that the given port can be accessed on localhost
	// It returns a new port number to connect to on localhost
	SetupLocalPort(ctx context.Context, port int) (int, error)
}
