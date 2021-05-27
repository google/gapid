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

package client

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/device/remotessh"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/core/vulkan/loader"
	"github.com/google/gapid/gapidapk"
	perfetto_android "github.com/google/gapid/gapis/perfetto/android"
)

const (
	deviceConnectionTimeout    = time.Second * 30
	maxCheckSocketFileAttempts = 10
	checkSocketFileRetryDelay  = time.Second
	// socketName must match kSocketName in gapir/cc/main.cpp
	socketName = "gapir-socket"
)

type deviceConnectionInfo struct {
	cleanupFunc func()
	port        int
	authToken   auth.Token
}

func initDeviceConnection(ctx context.Context, d bind.Device, abi *device.ABI, launchArgs []string) (*deviceConnectionInfo, error) {
	if host.Instance(ctx).SameAs(d.Instance()) {
		return newHost(ctx, bind.Host(ctx), abi, launchArgs)
	} else if adbd, ok := d.(adb.Device); ok {
		return newADB(ctx, adbd, abi, launchArgs)
	} else if remoted, ok := d.(remotessh.Device); ok {
		return newRemote(ctx, remoted, abi, launchArgs)
	} else {
		return nil, log.Errf(ctx, nil, "Cannot connect to device type %+v", d)
	}
}

func newRemote(ctx context.Context, d remotessh.Device, abi *device.ABI, launchArgs []string) (*deviceConnectionInfo, error) {
	authTokenFile, authToken := auth.GenTokenFile()
	defer os.Remove(authTokenFile)

	otherdir, cleanupTemp, err := d.TempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanupTemp(ctx)

	pf := otherdir + "/auth"
	if err = d.PushFile(ctx, authTokenFile, pf); err != nil {
		return nil, err
	}

	forceEnableDiskCache := true

	// If the user has not specified anything disk-cache related
	// we should force a disk cache for remote devices.
	for _, a := range launchArgs {
		if a == "--enable-disk-cache" ||
			a == "--disk-cache-path" ||
			a == "--cleanup-on-disk-cache" {
			forceEnableDiskCache = false
		}
	}

	args := []string{
		"--idle-timeout-sec", strconv.Itoa(int(deviceConnectionTimeout / time.Second)),
		"--auth-token-file", pf,
	}
	args = append(args, launchArgs...)
	if forceEnableDiskCache {
		args = append(args, "--enable-disk-cache")
		if len(d.DefaultReplayCacheDir()) > 0 {
			args = append(args, "--disk-cache-path", d.DefaultReplayCacheDir())
		}
		args = append(args, "--cleanup-on-disk-cache")
	}

	gapir, err := layout.Gapir(ctx, abi)
	if err = d.PushFile(ctx, gapir.System(), otherdir+"/gapir"); err != nil {
		return nil, err
	}

	remoteGapir := otherdir + "/gapir"

	env := shell.NewEnv()

	cleanup, err := loader.SetupReplay(ctx, d, abi, env)
	if err != nil {
		return nil, err
	}

	cleanupFunc := func() { cleanup.Invoke(ctx) }

	parser := func(severity log.Severity) io.WriteCloser {
		h := log.GetHandler(ctx)
		if h == nil {
			return nil
		}
		ctx := log.PutProcess(ctx, "gapir")
		ctx = log.PutFilter(ctx, nil)
		return text.Writer(func(line string) error {
			if m := parseHostLogMsg(line); m != nil {
				h.Handle(m)
				return nil
			}
			log.From(ctx).Log(severity, false, line)
			return nil
		})
	}

	stdout := parser(log.Info)
	if stdout != nil {
		defer stdout.Close()
	}

	stderr := parser(log.Error)
	if stderr != nil {
		defer stderr.Close()
	}

	log.I(ctx, "Starting gapir on remote: %v %v", remoteGapir, args)

	port, err := process.StartOnDevice(ctx, remoteGapir, process.StartOptions{
		Env:    env,
		Args:   args,
		Stdout: stdout,
		Stderr: stderr,
		Device: d,
	})

	if err != nil {
		log.E(ctx, "Starting gapir. Error: %v", err)
		return nil, err
	}

	return &deviceConnectionInfo{port: port, authToken: authToken, cleanupFunc: cleanupFunc}, nil
}

// newHost spawns and returns a new GAPIR instance on the host machine.
func newHost(ctx context.Context, d bind.DeviceWithShell, abi *device.ABI, launchArgs []string) (*deviceConnectionInfo, error) {
	authTokenFile, authToken := auth.GenTokenFile()
	defer os.Remove(authTokenFile)

	args := []string{
		"--idle-timeout-sec", strconv.Itoa(int(deviceConnectionTimeout / time.Second)),
		"--auth-token-file", authTokenFile,
	}
	args = append(args, launchArgs...)

	gapir, err := layout.Gapir(ctx, abi)
	if err != nil {
		log.F(ctx, true, "Couldn't locate gapir executable: %v", err)
		return nil, err
	}

	env := shell.CloneEnv()
	cleanup, err := loader.SetupReplay(ctx, d, abi, env)
	if err != nil {
		return nil, err
	}

	cleanupFunc := func() { cleanup.Invoke(ctx) }

	parser := func(severity log.Severity) io.WriteCloser {
		h := log.GetHandler(ctx)
		if h == nil {
			return nil
		}
		ctx := log.PutProcess(ctx, "gapir")
		ctx = log.PutFilter(ctx, nil)
		return text.Writer(func(line string) error {
			if m := parseHostLogMsg(line); m != nil {
				h.Handle(m)
				return nil
			}
			log.From(ctx).Log(severity, false, line)
			return nil
		})
	}

	stdout := parser(log.Info)
	if stdout != nil {
		defer stdout.Close()
	}

	stderr := parser(log.Error)
	if stderr != nil {
		defer stderr.Close()
	}

	log.I(ctx, "Starting gapir on host: %v %v", gapir.System(), args)
	port, err := process.StartOnDevice(ctx, gapir.System(), process.StartOptions{
		Env:    env,
		Args:   args,
		Stdout: stdout,
		Stderr: stderr,
		Device: d,
	})
	if err != nil {
		log.E(ctx, "Starting gapir. Error: %v", err)
		return nil, err
	}

	return &deviceConnectionInfo{port: port, authToken: authToken, cleanupFunc: cleanupFunc}, nil
}

func newADB(ctx context.Context, d adb.Device, abi *device.ABI, launchArgs []string) (*deviceConnectionInfo, error) {
	ctx = log.V{"abi": abi}.Bind(ctx)

	log.I(ctx, "Unlocking device screen")
	unlocked, err := d.UnlockScreen(ctx)
	if err != nil {
		log.W(ctx, "Failed to determine lock state: %s", err)
	} else if !unlocked {
		return nil, log.Err(ctx, nil, "Please unlock your device screen: GAPID can automatically unlock the screen only when no PIN/password/pattern is needed")
	}

	log.I(ctx, "Checking gapid.apk is installed...")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return nil, err
	}

	completeLaunchArgs := []string{
		"--idle-timeout-sec", string(int(deviceConnectionTimeout / time.Second)),
	}

	for _, arg := range launchArgs {
		completeLaunchArgs = append(completeLaunchArgs, arg)
	}

	gapirActivityIndex := -1
	for i, activityAction := range apk.ActivityActions {
		if activityAction.Activity == "com.google.android.gapid.ReplayerActivity" {
			gapirActivityIndex = i
			break
		}
	}
	if gapirActivityIndex < 0 {
		return nil, log.Errf(ctx, nil, "Cannot find gapir activity in gapid APK")
	}

	log.I(ctx, "Launching GAPIR...")
	// Configure GAPIR to be traceable for replay profiling
	cleanup, err := perfetto_android.SetupProfileLayersSource(ctx, d, apk.InstalledPackage, abi)

	if err != nil {
		// TODO(apbodnar) Fail here if we know we need render stages
		log.W(ctx, "Failed to setup GPU activity producer environment for replayer")
		cleanup.Invoke(ctx)
	}

	if err := d.StartActivity(ctx, *apk.ActivityActions[gapirActivityIndex],
		android.StringExtra{"gapir-intent-flag", strings.Join(completeLaunchArgs, " ")},
	); err != nil {
		return nil, err
	}

	log.I(ctx, "Setting up port forwarding...")
	localPort, err := adb.LocalFreeTCPPort()
	if err != nil {
		return nil, log.Err(ctx, err, "Finding free port")
	}

	port := int(localPort)
	ctx = log.V{"socket": socketName}.Bind(ctx)
	apkDir, err := apk.FileDir(ctx)
	if err != nil {
		return nil, log.Errf(ctx, err, "Getting gapid.apk files directory")
	}
	appDir, err := apk.AppDir(ctx)
	if err != nil {
		return nil, log.Errf(ctx, err, "Getting gapid.apk directory")
	}

	// Ignore the error returned from this. This is best-effort.
	// See: https://android.googlesource.com/platform/ndk.git/+/ndk-release-r18/ndk-gdb.py#386
	// for more information.
	_, _ = d.Shell("run-as", apk.Name, "chmod", "+x", appDir).Call(ctx)

	// Wait for the socket file to be created
	socketPath := strings.Join([]string{apkDir, socketName}, "/")
	err = task.Retry(ctx, maxCheckSocketFileAttempts, checkSocketFileRetryDelay,
		func(ctx context.Context) (bool, error) {
			str, err := d.Shell("run-as", apk.Name, "ls", socketPath).Call(ctx)
			if err != nil {
				return false, err
			}
			if strings.HasSuffix(str, "No such file or directory") {
				return false, log.Errf(ctx, nil, "Gapir socket '%v' not created yet", socketPath)
			}
			return true, nil
		})
	if err != nil {
		return nil, log.Errf(ctx, err, "Checking socket: %v", socketPath)
	}
	log.I(ctx, "Gapir socket: '%v' is opened now", socketPath)

	if err := d.Forward(ctx, localPort, adb.NamedFileSystemSocket(socketPath)); err != nil {
		return nil, log.Err(ctx, err, "Forwarding port")
	}

	cleanupFunc := func() {
		cleanup.Invoke(ctx)
		d.RemoveForward(ctx, localPort)
	}

	return &deviceConnectionInfo{port: port, authToken: "", cleanupFunc: cleanupFunc}, nil
}
