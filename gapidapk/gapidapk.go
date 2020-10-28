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
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event/task"
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

	perfettoProducerLauncher = "agi_launch_producer"
	launcherPath             = "/data/local/tmp/agi_launch_producer"
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

func ensureInstalled(ctx context.Context, d adb.Device, abi *device.ABI) (*APK, error) {
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

	name := PackageName(abi)

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

// EnsureInstalled ensures that gapid.apk with the specified ABI is installed on
// d with the same version as the APK on the host, and returns the installed APK.
// If abi is nil or UnknownABI, all the ABI available on the host will be tried
// for d, and the preferred ABI of the device will be tried first. Once an ABI
// is found compatible with the device, the APK of that ABI will be ensured to
// be installed.
func EnsureInstalled(ctx context.Context, d adb.Device, abi *device.ABI) (*APK, error) {
	ensureInstalledMutex.Lock()
	defer ensureInstalledMutex.Unlock()

	ctx = log.Enter(ctx, "gapidapk.EnsureInstalled")
	if abi.SameAs(device.UnknownABI) {
		abisToTry := []*device.ABI{d.Instance().GetConfiguration().PreferredABI(nil)}
		abisToTry = append(abisToTry, d.Instance().GetConfiguration().GetABIs()...)
		for _, a := range abisToTry {
			tempCtx := log.Enter(ctx, fmt.Sprintf("Try ABI: %s", a.Name))
			apk, err := ensureInstalled(tempCtx, d, a)
			if err == nil {
				return apk, nil
			}
			log.I(tempCtx, err.Error())
		}
	} else {
		return ensureInstalled(ctx, d, abi)
	}
	return nil, log.Err(ctx, nil, "Unable to install GAPID")
}

// LibsPath returns the path on the Android device to the GAPID libs directory.
// gapid.apk must be installed for this path to be valid.
func (a APK) LibsPath(abi *device.ABI) string {
	switch {
	case abi.SameAs(device.AndroidARMv7a):
		return a.path + "/lib/arm"
	case abi.SameAs(device.AndroidARM64v8a):
		return a.path + "/lib/arm64"
	}
	return a.path + "/lib/" + abi.Name
}

// LibGAPIIPath returns the path on the Android device to the GAPII dynamic
// library file.
// gapid.apk must be installed for this path to be valid.
func (a APK) LibGAPIIPath(abi *device.ABI) string {
	return a.LibsPath(abi) + "/" + LibGAPIIName
}

const (
	// LibGAPIIName is the name of the GAPII dynamic library file.
	LibGAPIIName = "libgapii.so"

	// GraphicsSpyLayerName is the name of the graphics spy layer.
	GraphicsSpyLayerName = "GraphicsSpy"
)

// PackageName returns the full package name of the GAPID apk for the given ABI.
func PackageName(abi *device.ABI) string {
	switch {
	case abi.SameAs(device.AndroidARMv7a):
		return "com.google.android.gapid.armeabiv7a"
	case abi.SameAs(device.AndroidARM64v8a):
		return "com.google.android.gapid.arm64v8a"
	default:
		return fmt.Sprintf("com.google.android.gapid.%v", abi.Name)
	}
}

// LayerName returns the name of the layer to use for Vulkan vs GLES.
func LayerName(vulkan bool) string {
	if vulkan {
		return GraphicsSpyLayerName
	} else {
		return LibGAPIIName
	}
}

// EnsurePerfettoProducerLaunched kills the existing launch_producer and starts
// a new one to launch the perfetto data producer.
func EnsurePerfettoProducerLaunched(ctx context.Context, d adb.Device) error {
	// Always kill the exisiting perfetto producer launcher, otherwise perfetto client
	// may end up quering back wrong data source information.
	d.Shell("killall", perfettoProducerLauncher).Run(ctx)
	driver, err := d.GraphicsDriver(ctx)
	if err != nil {
		log.W(ctx, "Failed to query developer driver: %v, assuming no developer driver found.", err)
	}

	signal, launched := task.NewSignal()
	launched = task.Once(launched)
	crash.Go(func() {
		err := launchPerfettoProducer(ctx, d, driver, launched)
		if err != nil {
			log.E(ctx, "[EnsurePerfettoProducerLaunched] error: %v", err)
			launched(ctx)
		}
	})
	if !signal.Wait(ctx) {
		return task.StopReason(ctx)
	}

	// TODO(b/148420473): Data sources are queried before gpu.counters is registered.
	time.Sleep(1 * time.Second)
	return nil
}

func preparePerfettoProducerLauncherFromApk(ctx context.Context, d adb.Device) error {
	packageName := PackageName(d.Instance().GetConfiguration().PreferredABI(nil))
	res, err := d.Shell("pm", "path", packageName).Call(ctx)
	if err != nil {
		return log.Errf(ctx, err, "Failed to query path to apk %v", packageName)
	}
	packagePath := strings.Split(res, ":")[1]
	d.Shell("rm", "-f", launcherPath).Call(ctx)
	if _, err := d.Shell("unzip", "-o", packagePath, "assets/"+perfettoProducerLauncher, "-p", ">", launcherPath).Call(ctx); err != nil {
		return log.Errf(ctx, err, "Failed to unzip %v from %v", perfettoProducerLauncher, packageName)
	}

	// Finally, make sure the binary is executable
	d.Shell("chmod", "a+x", launcherPath).Call(ctx)
	return nil
}

func launchPerfettoProducer(ctx context.Context, d adb.Device, driver adb.Driver, launched task.Task) error {
	// Extract the producer launcher from the APK.
	if err := preparePerfettoProducerLauncherFromApk(ctx, d); err != nil {
		return err
	}

	// Construct the shell command to launch producer. If developer driver is
	// found, then construct against it. Otherwise, directly run the command to
	// launch the data producers in the system image.
	script := launcherPath
	if driver.Package != "" {
		abi := d.Instance().GetConfiguration().PreferredABI(nil)
		script = "LD_LIBRARY_PATH=\"" + driver.Path + "!/lib/" + abi.Name + "/\" " + script
	}

	// Construct IO pipe, shell command outputs to stdout, GAPID reads from
	// reader for logging purpose.
	reader, stdout := io.Pipe()
	fail := make(chan error, 1)
	crash.Go(func() {
		buf := bufio.NewReader(reader)
		for {
			line, e := buf.ReadString('\n')
			switch e {
			default:
				log.E(ctx, "[launch producer] Read error %v", e)
				fail <- e
				return
			case io.EOF:
				fail <- nil
				return
			case nil:
				// As soon as there's output, consider the binary running.
				launched(ctx)
				log.I(ctx, "[launch producer] %s", strings.TrimSuffix(adb.AnsiRegex.ReplaceAllString(line, ""), "\n"))
			}
		}
	})

	// Start the shell command to launch producer
	process, err := d.Shell(script).
		Capture(stdout, stdout).
		Start(ctx)
	if err != nil {
		stdout.Close()
		return err
	}

	wait := make(chan error, 1)
	crash.Go(func() {
		wait <- process.Wait(ctx)
	})

	// Wait until either an error or EOF is read, or shell command exits.
	select {
	case err = <-fail:
		return err
	case err = <-wait:
		log.I(ctx, "[launch producer] Exit.")
		// Do nothing.
	}
	stdout.Close()
	if err != nil {
		return err
	}
	return <-fail
}
