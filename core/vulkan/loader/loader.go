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

// Package loader contains utilities for setting up the Vulkan loader.
package loader

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/remotessh"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

// deviceReplaySetup handles getting files from/to the right location
// for a particular Device
type deviceReplaySetup interface {
	// MakeTempDir returns a path to a created temporary
	MakeTempDir(ctx context.Context) (string, func(ctx context.Context), error)

	// InitializeLibrary takes a library, and if necessary copies it
	// into the given temporary directory. It returns the library
	// location if necessary.
	InitializeLibrary(ctx context.Context, tempdir string, library layout.LibraryType) (string, error)

	// FinalizeJSON puts the given JSON content in the given file
	FinalizeJSON(ctx context.Context, jsonName string, content string) (string, error)
}

// remoteSetup describes moving files to a remote device.
type remoteSetup struct {
	device remotessh.Device
	abi    *device.ABI
}

func (r *remoteSetup) MakeTempDir(ctx context.Context) (string, func(ctx context.Context), error) {
	return r.device.MakeTempDir(ctx)
}

func (r *remoteSetup) InitializeLibrary(ctx context.Context, tempdir string, library layout.LibraryType) (string, error) {
	lib, err := layout.Library(ctx, library, r.abi)
	if err != nil {
		return "", err
	}
	libName := layout.GetLibraryName(library, r.abi)
	if err := r.device.PushFile(ctx, lib.System(), tempdir+"/"+libName); err != nil {
		return "", err
	}
	return tempdir + "/" + libName, nil
}

func (r *remoteSetup) FinalizeJSON(ctx context.Context, jsonName string, content string) (string, error) {
	if err := r.device.WriteFile(ctx, bytes.NewReader([]byte(content)), "0644", jsonName); err != nil {
		return "", err
	}
	return jsonName, nil
}

// localSetup sets up the JSON for a local device.
type localSetup struct {
	abi *device.ABI
}

func (*localSetup) MakeTempDir(ctx context.Context) (string, func(ctx context.Context), error) {
	tempdir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return "", nil, err
	}
	return tempdir, func(ctx context.Context) {
		os.RemoveAll(tempdir)
	}, nil
}

func (l *localSetup) InitializeLibrary(ctx context.Context, tempdir string, library layout.LibraryType) (string, error) {
	lib, err := layout.Library(ctx, library, l.abi)
	if err != nil {
		return "", err
	}
	return lib.System(), nil
}

func (*localSetup) FinalizeJSON(ctx context.Context, jsonName string, content string) (string, error) {
	if err := ioutil.WriteFile(jsonName, []byte(content), 0644); err != nil {
		return "", err
	}
	return jsonName, nil
}

// SetupTrace sets up the environment for tracing a local app. Returns a
// clean-up function to be called after the trace completes, and a temporary
// filename that can be used to find the port if stdout fails, or an error.
func SetupTrace(ctx context.Context, d bind.Device, abi *device.ABI, env *shell.Env) (func(ctx context.Context), string, error) {
	var setup deviceReplaySetup
	if dev, ok := d.(remotessh.Device); ok {
		setup = &remoteSetup{dev, abi}
	} else {
		setup = &localSetup{abi}
	}
	tempdir, cleanup, err := setup.MakeTempDir(ctx)
	if err != nil {
		return func(ctx context.Context) {}, "", err
	}

	lib, json, err := findLibraryAndJSON(ctx, setup, tempdir, layout.LibGraphicsSpy)
	var f string
	if err != nil {
		return func(ctx context.Context) {}, f, err
	}
	err = setupJSON(ctx, lib, json, setup, tempdir, env)
	if err == nil {
		if fl, e := ioutil.TempFile("", "gapii_port"); e == nil {
			err = e
			f = fl.Name()
			fl.Close()
			o := cleanup
			cleanup = func(ctx context.Context) {
				o(ctx)
				os.Remove(f)
			}
		}
		if err == nil {
			env.Set("LD_PRELOAD", lib).
				AddPathStart("VK_INSTANCE_LAYERS", "VkGraphicsSpy").
				AddPathStart("VK_DEVICE_LAYERS", "VkGraphicsSpy").
				Set("GAPII_PORT_FILE", f)
			if runtime.GOOS == "windows" {
				// Adds the extra MSYS DLL dependencies onto the path.
				// TODO: remove this hacky work-around.
				// https://github.com/google/gapid/issues/17
				gapit, err := layout.Gapit(ctx)
				if err == nil {
					env.AddPathStart("PATH", gapit.Parent().System())
				}
			}
		}
	}
	return cleanup, f, err
}

// SetupReplay sets up the environment for a desktop. Returns a clean-up
// function to be called after replay completes, or an error.
func SetupReplay(ctx context.Context, d bind.Device, abi *device.ABI, env *shell.Env) (func(ctx context.Context), error) {
	var setup deviceReplaySetup
	if dev, ok := d.(remotessh.Device); ok {
		setup = &remoteSetup{dev, abi}
	} else {
		setup = &localSetup{abi}
	}
	tempdir, cleanup, err := setup.MakeTempDir(ctx)
	if err != nil {
		return nil, err
	}

	lib, json, err := findLibraryAndJSON(ctx, setup, tempdir, layout.LibVirtualSwapChain)
	if err != nil {
		return func(ctx context.Context) {}, err
	}

	err = setupJSON(ctx, lib, json, setup, tempdir, env)
	return cleanup, err
}

func findLibraryAndJSON(ctx context.Context, rs deviceReplaySetup, tempdir string, libType layout.LibraryType) (string, file.Path, error) {
	lib, err := rs.InitializeLibrary(ctx, tempdir, libType)
	if err != nil {
		return "", file.Path{}, err
	}

	json, err := layout.Json(ctx, libType)
	if err != nil {
		return "", file.Path{}, err
	}
	return lib, json, nil
}

func setupJSON(ctx context.Context, library string, json file.Path, rs deviceReplaySetup, tempdir string, env *shell.Env) error {
	sourceContent, err := ioutil.ReadFile(json.System())
	if err != nil {
		return err
	}

	libName := strings.Replace(library, "\\", "\\\\", -1)
	fixedContent := strings.Replace(string(sourceContent[:]), "<library>", libName, 1)

	rs.FinalizeJSON(ctx, tempdir+"/"+json.Basename(), fixedContent)
	env.AddPathStart("VK_LAYER_PATH", tempdir)

	return nil
}
