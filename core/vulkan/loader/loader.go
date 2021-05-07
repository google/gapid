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
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

const (
	// Since Android NDK r21, the VK_LAYER_KHRONOS_validation meta layer
	// is available on both desktop and Android.
	VulkanValidationLayer = "VK_LAYER_KHRONOS_validation"
)

// setupHelper describes setting up the files on the device
type setupHelper struct {
	device bind.Device
	abi    *device.ABI
}

func (r *setupHelper) makeTempDir(ctx context.Context) (string, app.Cleanup, error) {
	return r.device.TempDir(ctx)
}

func (r *setupHelper) initializeLibrary(ctx context.Context, tempdir string, library layout.LibraryType) (string, error) {
	lib, err := layout.Library(ctx, library, r.abi)
	if err != nil {
		return "", err
	}
	libName := layout.LibraryName(library, r.abi)
	if err := r.device.PushFile(ctx, lib.System(), tempdir+"/"+libName); err != nil {
		return "", err
	}
	return tempdir + "/" + libName, nil
}

func (r *setupHelper) finalizeJSON(ctx context.Context, jsonName string, content string) (string, error) {
	if err := r.device.WriteFile(ctx, bytes.NewReader([]byte(content)), os.FileMode(0644), jsonName); err != nil {
		return "", err
	}
	return jsonName, nil
}

// SetupLayers sets up the environment so that the correct layers are enabled
// for the application.
func SetupLayers(ctx context.Context, layers []string, skipMissingLayers bool, d bind.Device, abi *device.ABI, env *shell.Env) (app.Cleanup, error) {
	if len(layers) == 0 {
		return nil, nil
	}
	setup := &setupHelper{d, abi}

	tempdir, cleanup, err := setup.makeTempDir(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range layers {
		libType, err := layout.LibraryFromLayerName(l)
		if err != nil {
			return nil, err
		}
		lib, json, err := findLibraryAndJSON(ctx, setup, tempdir, libType)

		if err != nil {
			if skipMissingLayers {
				continue
			}
			return cleanup.Invoke(ctx), err
		}
		err = setupJSON(ctx, lib, json, setup, tempdir, env)
		if err != nil {
			return cleanup.Invoke(ctx), err
		}
		env.AddPathStart("VK_INSTANCE_LAYERS", l)
	}
	env.AddPathStart("VK_LAYER_PATH", tempdir)
	if abi.OS == device.Windows {
		// Adds the extra MSYS DLL dependencies onto the path.
		// TODO: remove this hacky work-around.
		// https://github.com/google/gapid/issues/17
		gapit, err := layout.Gapit(ctx)
		if err == nil {
			env.AddPathStart("PATH", gapit.Parent().System())
		}
	}
	return cleanup, nil
}

// SetupTrace sets up the environment for tracing a local app. Returns a
// clean-up function to be called after the trace completes, and a temporary
// filename that can be used to find the port if stdout fails, or an error.
func SetupTrace(ctx context.Context, d bind.Device, abi *device.ABI, env *shell.Env) (app.Cleanup, string, error) {
	setup := &setupHelper{d, abi}

	tempdir, cleanup, err := setup.makeTempDir(ctx)
	if err != nil {
		return nil, "", err
	}

	lib, json, err := findLibraryAndJSON(ctx, setup, tempdir, layout.LibGraphicsSpy)
	var f string
	if err != nil {
		return cleanup.Invoke(ctx), f, err
	}
	err = setupJSON(ctx, lib, json, setup, tempdir, env)
	if err != nil {
		return cleanup.Invoke(ctx), "", err
	}
	env.AddPathStart("VK_LAYER_PATH", tempdir)

	f, c, err := d.TempFile(ctx)
	if err != nil {
		return cleanup.Invoke(ctx), "", err
	}
	cleanup = cleanup.Then(c)
	env.Set("LD_PRELOAD", lib).
		AddPathStart("VK_INSTANCE_LAYERS", "GraphicsSpy").
		AddPathStart("VK_DEVICE_LAYERS", "GraphicsSpy").
		Set("GAPII_PORT_FILE", f)
	if abi.OS == device.Windows {
		// Adds the extra MSYS DLL dependencies onto the path.
		// TODO: remove this hacky work-around.
		// https://github.com/google/gapid/issues/17
		gapit, err := layout.Gapit(ctx)
		if err == nil {
			env.AddPathStart("PATH", gapit.Parent().System())
		}
	}
	return cleanup, f, err
}

// SetupReplay sets up the environment for a desktop. Returns a clean-up
// function to be called after replay completes, or an error.
func SetupReplay(ctx context.Context, d bind.Device, abi *device.ABI, env *shell.Env) (app.Cleanup, error) {
	setup := &setupHelper{d, abi}

	tempdir, cleanup, err := setup.makeTempDir(ctx)
	if err != nil {
		return nil, err
	}

	lib, json, err := findLibraryAndJSON(ctx, setup, tempdir, layout.LibVirtualSwapChain)
	if err != nil {
		return cleanup.Invoke(ctx), err
	}

	if err = setupJSON(ctx, lib, json, setup, tempdir, env); err != nil {
		return cleanup.Invoke(ctx), err
	}
	env.AddPathStart("VK_LAYER_PATH", tempdir)
	return cleanup, nil
}

// findLibraryAndJSON moves the library to the correct location (either locally or remotely) and returns
// a string representing the location, it also returns the file.Path of the associated JSON file for this
// library.
func findLibraryAndJSON(ctx context.Context, rs *setupHelper, tempdir string, libType layout.LibraryType) (string, file.Path, error) {
	lib, err := rs.initializeLibrary(ctx, tempdir, libType)
	if err != nil {
		return "", file.Path{}, err
	}

	json, err := layout.Json(ctx, libType, rs.abi)
	if err != nil {
		return "", file.Path{}, err
	}
	return lib, json, nil
}

func setupJSON(ctx context.Context, library string, json file.Path, rs *setupHelper, tempdir string, env *shell.Env) error {
	sourceContent, err := ioutil.ReadFile(json.System())
	if err != nil {
		return err
	}

	libName := strings.Replace(library, "\\", "\\\\", -1)
	fixedContent := strings.Replace(string(sourceContent[:]), "<library>", libName, 1)

	rs.finalizeJSON(ctx, tempdir+"/"+json.Basename(), fixedContent)

	return nil
}
