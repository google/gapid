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
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"sync"
	"time"

	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
)

const (
	installAttempts = 5
	checkFrequency  = time.Second * 5
)

var ensureInstalledMutex sync.Mutex

// APK represents the installed GAPIR APK.
type APK struct {
	*android.InstalledPackage
	path string
}

type lastInstallCheckKey struct{ *device.ABI }
type lastInstallCheckRes struct {
	time time.Time
	apk  *APK
}

// EnsureInstalled ensures that gapid.apk with the specified ABI is installed on
// d with the same version as the host APK and returns the installed APK.
// If abi is nil or UnknownABI then the preferred ABI of the device is used.
func EnsureInstalled(ctx context.Context, d adb.Device, abi *device.ABI) (*APK, error) {
	ensureInstalledMutex.Lock()
	defer ensureInstalledMutex.Unlock()

	ctx = log.Enter(ctx, "gapidapk.EnsureInstalled")

	if abi.SameAs(device.UnknownABI) {
		abi = d.Instance().GetConfiguration().PreferredABI(nil)
	}

	ctx = log.V{"abi": abi.Name}.Bind(ctx)

	// Was this recently checked?
	reg, checkKey := bind.GetRegistry(ctx), lastInstallCheckKey{abi}
	if res, ok := reg.DeviceProperty(ctx, d, checkKey).(lastInstallCheckRes); ok {
		if time.Since(res.time) < checkFrequency {
			return res.apk, nil
		}
	}

	// Check the device actually supports the requested ABI.
	if !d.Instance().Configuration.SupportsABI(abi) {
		return nil, log.Errf(ctx, nil, "Device does not support requested abi: %v", abi.Name)
	}

	name := pkgName(abi)

	log.I(ctx, "Examining gapid.apk on host...")
	apkPath, err := layout.GapidApk(ctx, abi)
	if err != nil {
		return nil, log.Err(ctx, err, "Finding gapid.apk on host")
	}

	ctx = log.V{"gapid.apk": apkPath.System()}.Bind(ctx)
	apkData, err := ioutil.ReadFile(apkPath.System())
	if err != nil {
		return nil, log.Err(ctx, err, "Opening gapid.apk")
	}

	apkFiles, err := apk.Read(ctx, apkData)
	if err != nil {
		return nil, log.Err(ctx, err, "Reading gapid.apk")
	}

	apkManifest, err := apk.GetManifest(ctx, apkFiles)
	if err != nil {
		return nil, log.Err(ctx, err, "Reading gapid.apk manifest")
	}

	ctx = log.V{
		"target-version-name": apkManifest.VersionName,
		"target-version-code": apkManifest.VersionCode,
	}.Bind(ctx)

	for attempts := installAttempts; attempts > 0; attempts-- {
		log.I(ctx, "Looking for gapid.apk...")
		gapid, err := d.InstalledPackage(ctx, name)
		if err != nil {
			log.I(ctx, "Installing gapid.apk...")
			if err := d.InstallAPK(ctx, apkPath.System(), false, true); err != nil {
				return nil, log.Err(ctx, err, "Installing gapid.apk")
			}
			continue
		}

		ctx = log.V{
			"installed-version-name": gapid.VersionName,
			"installed-version-code": gapid.VersionCode,
		}.Bind(ctx)

		if gapid.VersionCode != apkManifest.VersionCode ||
			gapid.VersionName != apkManifest.VersionName {
			log.I(ctx, "Uninstalling existing gapid.apk as version has changed.")
			gapid.Uninstall(ctx)
			continue
		}

		apkPath, err := gapid.Path(ctx)
		if err != nil {
			return nil, log.Err(ctx, err, "Obtaining GAPID package path")
		}
		log.I(ctx, "Found gapid package...")

		out := &APK{gapid, path.Dir(apkPath)}
		reg.SetDeviceProperty(ctx, d, checkKey, lastInstallCheckRes{time.Now(), out})
		return out, nil
	}

	return nil, log.Err(ctx, nil, "Unable to install GAPID")
}

// LibsPath returns the path on the Android device to the GAPID libs directory.
// gapid.apk must be installed for this path to be valid.
func (a APK) LibsPath(abi *device.ABI) string {
	switch {
	case abi.SameAs(device.AndroidARM),
		abi.SameAs(device.AndroidARMv7a):
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
		abi.SameAs(device.AndroidARMv7a):
		return "com.google.android.gapid.armeabi"
	case abi.SameAs(device.AndroidARM64v8a):
		return "com.google.android.gapid.aarch64"
	default:
		return fmt.Sprintf("com.google.android.gapid.%v", abi.Name)
	}
}
