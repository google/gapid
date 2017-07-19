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
	"context"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

// SetupTrace sets up the environment for tracing a local app. Returns a
// clean-up function to be called after the trace completes, and a temporary
// filename that can be used to find the port if stdout fails, or an error.
func SetupTrace(ctx context.Context, env *shell.Env) (func(), string, error) {
	lib, json, err := findLibraryAndJSON(ctx, layout.LibGraphicsSpy)
	var f string
	if err != nil {
		return func() {}, f, err
	}
	cleanup, err := setupJSON(lib, json, env)
	if err == nil {
		if fl, e := ioutil.TempFile("", "gapii_port"); e == nil {
			err = e
			f = fl.Name()
			fl.Close()
			o := cleanup
			cleanup = func() {
				o()
				os.Remove(f)
			}
		}
		if err == nil {
			env.Set("LD_PRELOAD", lib.System()).
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

// SetupReplay sets up the environment for local replay. Returns a clean-up
// function to be called after replay completes, or an error.
func SetupReplay(ctx context.Context, env *shell.Env) (func(), error) {
	lib, json, err := findLibraryAndJSON(ctx, layout.LibVirtualSwapChain)
	if err != nil {
		return func() {}, err
	}
	return setupJSON(lib, json, env)
}

func findLibraryAndJSON(ctx context.Context, libType layout.LibraryType) (file.Path, file.Path, error) {
	lib, err := layout.Library(ctx, libType)
	if err != nil {
		return file.Path{}, file.Path{}, err
	}

	json, err := layout.Json(ctx, libType)
	if err != nil {
		return file.Path{}, file.Path{}, err
	}
	return lib, json, nil
}

func setupJSON(library, json file.Path, env *shell.Env) (func(), error) {
	cleanup := func() {}

	sourceContent, err := ioutil.ReadFile(json.System())
	if err != nil {
		return cleanup, err
	}
	tempdir, err := ioutil.TempDir("", "gapit_dir")
	if err != nil {
		return cleanup, err
	}
	cleanup = func() {
		os.RemoveAll(tempdir)
	}

	libName := strings.Replace(library.System(), "\\", "\\\\", -1)
	fixedContent := strings.Replace(string(sourceContent[:]), "<library>", libName, 1)
	ioutil.WriteFile(tempdir+"/"+json.Basename(), []byte(fixedContent), 0644)
	env.AddPathStart("VK_LAYER_PATH", tempdir)

	return cleanup, nil
}
