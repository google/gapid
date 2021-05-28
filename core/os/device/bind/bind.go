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
	"io/ioutil"
	"os"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/shell"
)

type binding struct {
	Simple
}

var (
	hostMutex sync.Mutex
	hostDev   Desktop
)

// Host returns the Device to the host.
func Host(ctx context.Context) Desktop {
	hostMutex.Lock()
	defer hostMutex.Unlock()
	if hostDev == nil {
		hostDev = &binding{
			Simple{
				To: host.Instance(ctx),
			},
		}
	}
	return hostDev
}

// Shell implements the Device interface returning commands that will error if run.
func (b *binding) Shell(name string, args ...string) shell.Cmd {
	return shell.Command(name, args...).On(hostTarget{}).Verbose()
}

// TempFile creates a temporary file on the given Device. It returns the
// path to the file, and a function that can be called to clean it up.
func (b *binding) TempFile(ctx context.Context) (string, func(ctx context.Context), error) {
	fl, e := ioutil.TempFile("", "")
	if e != nil {
		return "", nil, e
	}

	f := fl.Name()
	fl.Close()
	return f, func(ctx context.Context) {
		os.Remove(f)
	}, nil
}

// FileContents returns the contents of a given file on the Device.
func (b *binding) FileContents(ctx context.Context, path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func (b *binding) PushFile(ctx context.Context, sourcePath, destPath string) error {
	in, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer in.Close()
	return b.WriteFile(ctx, in, 0666, destPath)
}

func (b *binding) TempDir(ctx context.Context) (string, app.Cleanup, error) {
	fl, e := ioutil.TempDir("", "")
	if e != nil {
		return "", nil, e
	}

	return fl, func(ctx context.Context) {
		os.RemoveAll(fl)
	}, nil
}

func (b *binding) WriteFile(ctx context.Context, contents io.Reader, mode os.FileMode, destPath string) error {
	out, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, contents)
	if err != nil {
		return err
	}
	return out.Close()
}

// RemoveFile removes the given file from the device
func (b *binding) RemoveFile(ctx context.Context, path string) error {
	return os.Remove(path)
}

// GetEnv returns the default environment for the Device.
func (b *binding) GetEnv(ctx context.Context) (*shell.Env, error) {
	return shell.CloneEnv(), nil
}

// SetupLocalPort makes sure that the given port can be accessed on localhost
// It returns a new port number to connect to on localhost
func (b *binding) SetupLocalPort(ctx context.Context, port int) (int, error) {
	return port, nil
}

// IsFile returns true if the given path refers to a file
func (b *binding) IsFile(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return false, nil
	}

	f, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if f.Mode().IsRegular() || (f.Mode()&(os.ModeNamedPipe|os.ModeSocket)) != 0 {
		return true, nil
	}
	return false, nil
}

// IsDirectory returns true if the given path refers to a directory
func (b *binding) IsDirectory(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return true, nil
	}

	f, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if f.Mode().IsDir() {
		return true, nil
	}
	return false, nil
}

// GetWorkingDirectory returns the directory that this device considers CWD
func (b *binding) GetWorkingDirectory(ctx context.Context) (string, error) {
	return os.Getwd()
}

func (b *binding) IsLocal(ctx context.Context) (bool, error) {
	return true, nil
}

type hostTarget struct{}

func (t hostTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	return shell.LocalTarget.Start(cmd)
}
