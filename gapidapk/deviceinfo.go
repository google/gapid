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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/flock"
)

const (
	sendDevInfoAction        = "com.google.android.gapid.action.SEND_DEV_INFO"
	sendDevInfoService       = "com.google.android.gapid.DeviceInfoService"
	sendDevInfoPort          = "gapid-devinfo"
	startServiceAttempts     = 3
	portListeningAttempts    = 5
	perfettoProducerLauncher = "launch_producer"
	launcherPath             = "/data/local/tmp/gapid_launch_producer"
	launcherScript           = "nohup %[1]s &"
)

func init() {
	adb.RegisterDeviceInfoProvider(fetchDeviceInfo)
}

// Returns true if the device is listening to sendDevInfoPort, false if not.
// Error if failed at getting the port info.
func devInfoPortListening(ctx context.Context, d adb.Device) (bool, error) {
	var stdout bytes.Buffer
	if err := d.Shell("cat", "/proc/net/unix").Capture(&stdout, nil).Run(ctx); err != nil {
		return false, log.Errf(ctx, err, "Getting unix abstract port info...")
	}
	if strings.Contains(stdout.String(), sendDevInfoPort) {
		return true, nil
	}
	return false, nil
}

// startDevInfoService tries to start the fresh run of the package and start
// the service to send device info.
func startDevInfoService(ctx context.Context, d adb.Device, apk *APK) error {
	ctx = log.Enter(ctx, "startDevInfoService")
	var listening bool

	action := apk.ServiceActions.FindByName(sendDevInfoAction, sendDevInfoService)
	if action == nil {
		return log.Err(ctx, nil, "Service intent was not found")
	}

	// Try to start service.
	err := task.Retry(ctx, startServiceAttempts, 100*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			log.I(ctx, "Attempt to start service: %s", sendDevInfoService)
			if err := d.StartService(ctx, *action); err != nil {
				return false, err
			}
			err := task.Retry(ctx, portListeningAttempts, time.Second, func(
				ctx context.Context) (bool, error) {
				var err error
				listening, err = devInfoPortListening(ctx, d)
				return listening, err
			})
			return listening, err
		})
	if listening {
		return nil
	}
	return log.Errf(ctx, err, "Start DevInfo service: Run out of attempts: %v",
		startServiceAttempts)
}

func fetchDeviceInfo(ctx context.Context, d adb.Device) error {
	apk, err := EnsureInstalled(ctx, d, device.UnknownABI)
	if err != nil {
		// The gapid.apk was not found. This can happen with partial builds used
		// for testing.
		// Don't return an error as this will prevent the device from being
		// registered and the device already comes with basic usable
		// information.
		log.W(ctx, "Couldn't find gapid.apk for device. Error: %v", err)
		return nil
	}

	// Close any previous runs of the apk
	apk.Stop(ctx)

	driver, err := d.GraphicsDriver(ctx)
	if err != nil {
		return err
	}

	var cleanup app.Cleanup

	// Set up device info service to use prerelease driver.
	nextCleanup, err := adb.SetupPrereleaseDriver(ctx, d, apk.InstalledPackage)
	cleanup = cleanup.Then(nextCleanup)
	if err != nil {
		cleanup.Invoke(ctx)
		return err
	}

	// Set driver package
	nextCleanup, err = android.SetupLayers(ctx, d, apk.Name, []string{driver.Package}, []string{}, true)
	cleanup = cleanup.Then(nextCleanup)
	if err != nil {
		cleanup.Invoke(ctx)
		return err
	}
	defer cleanup.Invoke(ctx)

	if d.Instance().GetConfiguration().GetOS().GetAPIVersion() >= 29 {
		startSignal, startFunc := task.NewSignal()
		startFunc = task.Once(startFunc)
		crash.Go(func() {
			err := launchPerfettoProducerFromApk(ctx, d, startFunc)
			if err != nil {
				log.E(ctx, "[launchPerfettoProducerFromApk] error: %v", err)
			}

			// Ensure the start signal is fired on failure/immediate return.
			startFunc(ctx)
		})
		startSignal.Wait(ctx)
	}

	// Make sure the device is available to query device info, this is to prevent
	// Vulkan trace from happening at the same time than device info query.
	m := flock.Lock(d.Instance().GetSerial())
	defer m.Unlock()

	// Tries to start the device info service.
	if err := startDevInfoService(ctx, d, apk); err != nil {
		return log.Err(ctx, err, "Starting service")
	}

	sock, err := adb.ForwardAndConnect(ctx, d, sendDevInfoPort)
	if err != nil {
		return log.Err(ctx, err, "Connecting to service port")
	}

	defer sock.Close()

	data, err := ioutil.ReadAll(sock)
	if err != nil {
		return log.Err(ctx, err, "Reading data")
	}

	if err := proto.UnmarshalMerge(data, d.Instance()); err != nil {
		return log.Err(ctx, err, "Unmarshalling device Instance")
	}

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

func launchPerfettoProducerFromApk(ctx context.Context, d adb.Device, startFunc task.Task) error {
	driver, err := d.GraphicsDriver(ctx)
	if err != nil {
		return err
	}

	// Extract the producer launcher from the APK.
	if err := preparePerfettoProducerLauncherFromApk(ctx, d); err != nil {
		return err
	}

	// Construct IO pipe, shell command outputs to stdout, GAPID reads from
	// reader for logging purpose.
	reader, stdout := io.Pipe()
	fail := make(chan error, 1)
	crash.Go(func() {
		buf := bufio.NewReader(reader)
		for {
			line, e := buf.ReadString('\n')
			// As long as there's output, consider the binary starting running.
			startFunc(ctx)
			switch e {
			default:
				log.E(ctx, "[launch producer] Read error %v", e)
				fail <- e
				return
			case io.EOF:
				fail <- nil
				return
			case nil:
				log.E(ctx, "[launch producer] %s", strings.TrimSuffix(adb.AnsiRegex.ReplaceAllString(line, ""), "\n"))
			}
		}
	})

	// Start the shell command to launch producer
	script := fmt.Sprintf(launcherScript, launcherPath)
	if driver.Package != "" {
		abi := d.Instance().GetConfiguration().PreferredABI(nil)
		script = "export LD_LIBRARY_PATH=\"" + driver.Path + "!/lib/" + abi.Name + "/\";" + script
	}
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
		// Do nothing.
	}
	stdout.Close()
	if err != nil {
		return err
	}
	return <-fail
}
