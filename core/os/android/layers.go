// Copyright (C) 2019 Google Inc.
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

package android

import (
	"context"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

const eglLayersExt = "EGL_ANDROID_GLES_layers"

// SupportsLayersViaSystemSettings returns whether the given device supports
// loading GLES and Vulkan layers via the system settings.
func SupportsLayersViaSystemSettings(d Device) bool {
	// TODO: this currently looks for the EGL extension that advertises the
	// GLES layer loading capability. Technically, the extension could be
	// absent, but the device could still support loading Vulkan layers via
	// the settings. There doesn't appear to be a way to detect Vulkan support
	// and we currently assume a device either supports both, or none.
	exts := d.Instance().GetConfiguration().GetDrivers().GetOpengl().GetExtensions()
	for _, ext := range exts {
		if ext == eglLayersExt {
			return true
		}
	}
	return false
}

// SetupLayer initializes d to use either a Vulkan or GLES layer from layerPkgs
// limited to the app with package appPkg using the system settings and returns
// a cleanup to remove the layer settings.
func SetupLayers(ctx context.Context, d Device, appPkg string, layerPkgs []string, layers []string, vulkan bool) (app.Cleanup, error) {
	var cleanup app.Cleanup
	// pushSetting changes a device property for the duration of the trace.
	pushSetting := func(ns, key, val string) error {
		cleanup = cleanup.Then(func(ctx context.Context) {
			log.D(ctx, "Removing setting %v", key)
			d.DeleteSystemSetting(ctx, ns, key)
		})
		return d.SetSystemSetting(ctx, ns, key, val)
	}

	if err := pushSetting("global", "enable_gpu_debug_layers", "1"); err != nil {
		return cleanup.Invoke(ctx), err
	}
	if err := pushSetting("global", "gpu_debug_app", appPkg); err != nil {
		return cleanup.Invoke(ctx), err
	}
	if err := pushSetting("global", "gpu_debug_layer_app", "\""+strings.Join(layerPkgs, ":")+"\""); err != nil {
		return cleanup.Invoke(ctx), err
	}
	if len(layers) > 0 {
		if vulkan {
			if err := pushSetting("global", "gpu_debug_layers", "\""+strings.Join(layers, ":")+"\""); err != nil {
				return cleanup.Invoke(ctx), err
			}
		} else {
			if err := pushSetting("global", "gpu_debug_layers_gles", "\""+strings.Join(layers, ":")+"\""); err != nil {
				return cleanup.Invoke(ctx), err
			}
		}
	}

	return cleanup, nil
}
