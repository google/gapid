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

package gapidapk

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/device"
)

const installAttempts = 5

var ensureInstalledMutex sync.Mutex

// APK represents the installed GAPIR APK.
type APK struct {
	*android.InstalledPackage
	path string
}

// EnsureInstalled ensures that gapid.apk with the specified ABI is installed on
// d with the same version as the host APK and returns the installed APK.
// If abi is nil or UnknownABI then the preferred ABI of the device is used.
func EnsureInstalled(ctx log.Context, d adb.Device, abi *device.ABI) (*APK, error) {
	ensureInstalledMutex.Lock()
	defer ensureInstalledMutex.Unlock()

	ctx = ctx.Enter("gapidapk.EnsureInstalled")

	if abi.SameAs(device.UnknownABI) {
		abi = d.Instance().GetConfiguration().PreferredABI(nil)
	}

	ctx = ctx.S("abi", abi.Name)

	// Check the device actually supports the requested ABI.
	if !d.Instance().Configuration.SupportsABI(abi) {
		return nil, cause.Explain(ctx, nil, "Device does not support requested ABI.").With("abi", abi.Name)
	}

	name := pkgName(abi)

	ctx.Info().Log("Examining gapid.apk on host...")
	apkPath, err := layout.GapidApk(ctx, abi)
	if err != nil {
		return nil, cause.Explain(ctx, err, "Finding gapid.apk on host")
	}

	ctx = ctx.S("gapid.apk", apkPath.System())
	apkData, err := ioutil.ReadFile(apkPath.System())
	if err != nil {
		return nil, cause.Explain(ctx, err, "Opening gapid.apk")
	}

	apkFiles, err := apk.Read(ctx, apkData)
	if err != nil {
		return nil, cause.Explain(ctx, err, "Reading gapid.apk")
	}

	apkManifest, err := apk.GetManifest(ctx, apkFiles)
	if err != nil {
		return nil, cause.Explain(ctx, err, "Reading gapid.apk manifest")
	}

	ctx = ctx.
		S("target-version-name", apkManifest.VersionName).
		I("target-version-code", apkManifest.VersionCode)

	for attempts := installAttempts; attempts > 0; attempts-- {
		ctx.Info().Log("Looking at installed packages...")
		packages, err := d.InstalledPackages(ctx)
		if err != nil {
			return nil, cause.Explain(ctx, err, "Listing installed packages")
		}

		if gapid := packages.FindByName(name); gapid == nil {
			ctx.Info().Log("Installing gapid.apk...")
			if err := d.InstallAPK(ctx, apkPath.System(), false, true); err != nil {
				return nil, cause.Explain(ctx, err, "Installing gapid.apk")
			}
		} else {
			ctx = ctx.
				S("installed-version-name", gapid.VersionName).
				I("installed-version-code", gapid.VersionCode)

			if gapid.VersionCode != apkManifest.VersionCode ||
				gapid.VersionName != apkManifest.VersionName {
				ctx.Info().Log("Uninstalling existing gapid.apk as version has changed.")
				gapid.Uninstall(ctx)
				continue
			}

			path, err := gapid.Path(ctx)
			if err != nil {
				return nil, cause.Explain(ctx, err, "Obtaining GAPID package path")
			}
			ctx.Info().Log("Found gapid package...")
			return &APK{gapid, filepath.Dir(path)}, nil
		}
	}

	return nil, cause.Explain(ctx, nil, "Unable to install GAPID")
}

// LibsPath returns the path on the Android device to the GAPID libs directory.
// gapid.apk must be installed for this path to be valid.
func (a APK) LibsPath(abi *device.ABI) string {
	switch {
	case abi.SameAs(device.AndroidARM),
		abi.SameAs(device.AndroidARMv7a),
		abi.SameAs(device.AndroidARMv7aHard):
		return a.path + "/lib/arm"
	case abi.SameAs(device.AndroidARM64v8a):
		return a.path + "/lib/arm64"
	}
	return a.path + "/lib/" + abi.Name
}

// LibGAPIIPath returns the path on the Android device to libgapii.so.
// gapid.apk must be installed for this path to be valid.
func (a APK) LibGAPIIPath(abi *device.ABI) string {
	return a.LibsPath(abi) + "/libgapii.so"
}

// LibInterceptorPath returns the path on the Android device to
// libinterceptor.so.
// gapid.apk must be installed for this path to be valid.
func (a APK) LibInterceptorPath(abi *device.ABI) string {
	return a.LibsPath(abi) + "/libinterceptor.so"
}

func pkgName(abi *device.ABI) string {
	switch {
	case abi.SameAs(device.AndroidARM),
		abi.SameAs(device.AndroidARMv7a),
		abi.SameAs(device.AndroidARMv7aHard):
		return "com.google.android.gapid.armeabi"
	case abi.SameAs(device.AndroidARM64v8a):
		return "com.google.android.gapid.aarch64"
	default:
		return fmt.Sprintf("com.google.android.gapid.%v", abi.Name)
	}
}
