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
	"io"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/gapis/perfetto"
)

// Device represents a connection to an attached device.
type Device interface {
	// Instance returns the instance information for this device.
	Instance() *device.Instance
	// State returns the last known connected status of the device.
	Status(ctx context.Context) Status
	// Shell is a helper that builds a shell.Cmd with d.ShellTarget() as its target
	Shell(name string, args ...string) shell.Cmd
	// TempFile creates a temporary file on the given Device. It returns the
	// path to the file, and a function that can be called to clean it up.
	TempFile(ctx context.Context) (string, func(ctx context.Context), error)
	// TempDir makes a temporary directory, and returns the
	// path, as well as a function to call to clean it up.
	TempDir(ctx context.Context) (string, app.Cleanup, error)
	// FileContents returns the contents of a given file on the Device.
	FileContents(ctx context.Context, path string) (string, error)
	// RemoveFile removes the given file from the device
	RemoveFile(ctx context.Context, path string) error
	// GetEnv returns the default environment for the Device.
	GetEnv(ctx context.Context) (*shell.Env, error)
	// SetupLocalPort makes sure that the given port can be accessed on localhost
	// It returns a new port number to connect to on localhost
	SetupLocalPort(ctx context.Context, port int) (int, error)
	// ListExecutables returns the executables in a particular directory as given by path
	ListExecutables(ctx context.Context, path string) ([]string, error)
	// ListDirectories returns a list of directories rooted at a particular path
	ListDirectories(ctx context.Context, path string) ([]string, error)
	// GetURIRoot returns the root URI for the entire system
	GetURIRoot() string
	// IsFile returns true if the given path is a file
	IsFile(ctx context.Context, path string) (bool, error)
	// IsDirectory returns true if the given path is a directory
	IsDirectory(ctx context.Context, path string) (bool, error)
	// GetWorkingDirectory returns the directory that this device considers CWD
	GetWorkingDirectory(ctx context.Context) (string, error)
	// IsLocal returns true if this tracer is local
	IsLocal(ctx context.Context) (bool, error)
	// CanTrace returns true if this device can be used to take a trace
	CanTrace() bool
	// SupportsPerfetto returns true if this device will work with perfetto
	SupportsPerfetto(ctx context.Context) bool
	// SupportsAngle returns true if this device will work with ANGLE
	SupportsAngle(ctx context.Context) bool
	// ConnectPerfetto connects to a Perfetto service running on this device
	// and returns an open socket connection to the service.
	ConnectPerfetto(ctx context.Context) (*perfetto.Client, error)
	// PushFile will transfer the local file at sourcePath to the remote
	// machine at destPath
	PushFile(ctx context.Context, sourcePath, destPath string) error
	// WriteFile writes the given file into the given location on the remote device
	WriteFile(ctx context.Context, contents io.Reader, mode os.FileMode, destPath string) error
}
