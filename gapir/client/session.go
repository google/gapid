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
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
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
)

const (
	sessionTimeout             = time.Second * 30
	maxCheckSocketFileAttempts = 10
	checkSocketFileRetryDelay  = time.Second
	connectTimeout             = time.Second * 10
	heartbeatInterval          = time.Millisecond * 500
)

type session struct {
	device   bind.Device
	port     int
	auth     auth.Token
	closeCBs []func()
	inited   chan struct{}
	// The connection for heartbeat
	conn *Connection
}

func newSession(d bind.Device) *session {
	return &session{device: d, inited: make(chan struct{})}
}

func (s *session) init(ctx context.Context, d bind.Device, abi *device.ABI, launchArgs []string) error {
	defer close(s.inited)
	var err error

	if host.Instance(ctx).SameAs(d.Instance()) {
		err = s.newHost(ctx, d, abi, launchArgs)
	} else if adbd, ok := d.(adb.Device); ok {
		err = s.newADB(ctx, adbd, abi)
	} else if remoted, ok := d.(remotessh.Device); ok {
		err = s.newRemote(ctx, remoted, abi, launchArgs)
	} else {
		err = log.Errf(ctx, nil, "Cannot connect to device type %+v", d)
	}
	if err != nil {
		s.close(ctx)
		return err
	}

	crash.Go(func() { s.heartbeat(ctx, heartbeatInterval) })
	return nil
}

func (s *session) newRemote(ctx context.Context, d remotessh.Device, abi *device.ABI, launchArgs []string) error {
	authTokenFile, authToken := auth.GenTokenFile()
	defer os.Remove(authTokenFile)

	otherdir, cleanup, err := d.MakeTempDir(ctx)
	if err != nil {
		return err
	}
	defer cleanup(ctx)

	pf := otherdir + "/auth"
	if err = d.PushFile(ctx, authTokenFile, pf); err != nil {
		return err
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
		"--idle-timeout-sec", strconv.Itoa(int(sessionTimeout / time.Second)),
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
		return err
	}

	remoteGapir := otherdir + "/gapir"

	env := shell.NewEnv()
	sessionCleanup, err := loader.SetupReplay(ctx, d, abi, env)
	if err != nil {
		return err
	}
	s.onClose(func() { sessionCleanup(ctx) })

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
		return nil
	}

	s.conn, err = newConnection(fmt.Sprintf("localhost:%d", port), authToken, connectTimeout)
	if err != nil {
		return log.Err(ctx, err, "Timeout waiting for connection")
	}

	s.onClose(func() {
		if s.conn != nil {
			s.conn.Close()
		}
	})
	log.I(ctx, "Heartbeat connection setup done")

	s.port = port
	s.auth = authToken
	return nil
}

// newHost spawns and returns a new GAPIR instance on the host machine.
func (s *session) newHost(ctx context.Context, d bind.Device, abi *device.ABI, launchArgs []string) error {
	authTokenFile, authToken := auth.GenTokenFile()
	defer os.Remove(authTokenFile)

	args := []string{
		"--idle-timeout-sec", strconv.Itoa(int(sessionTimeout / time.Second)),
		"--auth-token-file", authTokenFile,
	}
	args = append(args, launchArgs...)

	gapir, err := layout.Gapir(ctx, abi)
	if err != nil {
		log.F(ctx, true, "Couldn't locate gapir executable: %v", err)
		return nil
	}

	env := shell.CloneEnv()
	cleanup, err := loader.SetupReplay(ctx, d, abi, env)
	if err != nil {
		return err
	}
	s.onClose(func() { cleanup(ctx) })

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
		return nil
	}

	s.conn, err = newConnection(fmt.Sprintf("localhost:%d", port), authToken, connectTimeout)
	if err != nil {
		return log.Err(ctx, err, "Timeout waiting for connection")
	}
	s.onClose(func() {
		if s.conn != nil {
			s.conn.Close()
		}
	})
	log.I(ctx, "Heartbeat connection setup done")

	s.port = port
	s.auth = authToken
	return nil
}

var socketNames = map[device.Architecture]string{
	device.ARMv7a: "gapir-arm",
	device.ARMv8a: "gapir-arm64",
	device.X86:    "gapir-x86",
	device.X86_64: "gapir-x86-64",
}

func (s *session) newADB(ctx context.Context, d adb.Device, abi *device.ABI) error {
	ctx = log.V{"abi": abi}.Bind(ctx)

	log.I(ctx, "Checking gapid.apk is installed...")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return err
	}

	log.I(ctx, "Launching GAPIR...")
	if err := d.StartActivity(ctx, *apk.ActivityActions[0],
		android.IntExtra{"idle_timeout", int(sessionTimeout / time.Second)},
		android.StringExtra{"auth_token", string(s.auth)},
	); err != nil {
		return err
	}

	log.I(ctx, "Setting up port forwarding...")
	localPort, err := adb.LocalFreeTCPPort()
	if err != nil {
		return log.Err(ctx, err, "Finding free port")
	}
	s.port = int(localPort)
	socket, ok := socketNames[abi.Architecture]
	ctx = log.V{"socket": socket}.Bind(ctx)
	if !ok {
		return log.Errf(ctx, nil, "Unsupported architecture: %v", abi.Architecture)
	}
	apkDir, err := apk.FileDir(ctx)
	if err != nil {
		return log.Errf(ctx, err, "Getting gapid.apk files directory")
	}
	appDir, err := apk.AppDir(ctx)
	if err != nil {
		return log.Errf(ctx, err, "Getting gapid.apk directory")
	}

	// Ignore the error returned from this. This is best-effort.
	// See: https://android.googlesource.com/platform/ndk.git/+/ndk-release-r18/ndk-gdb.py#386
	// for more information.
	_, _ = d.Shell("run-as", apk.Name, "chmod", "+x", appDir).Call(ctx)

	// Wait for the socket file to be created
	socketPath := strings.Join([]string{apkDir, socket}, "/")
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
		return log.Errf(ctx, err, "Checking socket: %v", socketPath)
	}
	log.I(ctx, "Gapir socket: '%v' is opened now", socketPath)

	if err := d.Forward(ctx, localPort, adb.NamedFileSystemSocket(socketPath)); err != nil {
		return log.Err(ctx, err, "Forwarding port")
	}
	s.onClose(func() { d.RemoveForward(ctx, localPort) })

	log.I(ctx, "Waiting for connection to GAPIR...")
	s.conn, err = newConnection(fmt.Sprintf("localhost:%d", localPort), s.auth, connectTimeout)
	if err != nil {
		return log.Err(ctx, err, "Timeout waiting for connection")
	}
	s.onClose(func() {
		if s.conn != nil {
			s.conn.Close()
		}
	})
	log.I(ctx, "Heartbeat connection setup done")
	return nil
}

func (s *session) connect(ctx context.Context) (*Connection, error) {
	<-s.inited
	return newConnection(fmt.Sprintf("localhost:%d", s.port), s.auth, connectTimeout)
}

func (s *session) onClose(f func()) {
	s.closeCBs = append(s.closeCBs, f)
}

func (s *session) close(ctx context.Context) {
	s.conn.Shutdown(ctx)
	for _, f := range s.closeCBs {
		f()
	}
	s.closeCBs = nil
}

func (s *session) ping(ctx context.Context) (time.Duration, error) {
	if s.conn == nil {
		return time.Duration(0), log.Errf(ctx, nil, "cannot ping without gapir connection")
	}
	start := time.Now()
	err := s.conn.Ping(ctx)
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func (s *session) heartbeat(ctx context.Context, pingInterval time.Duration) {
	defer s.close()
	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case <-time.After(pingInterval):
			_, err := s.ping(ctx)
			if err != nil {
				log.E(ctx, "Error sending keep-alive ping. Error: %v", err)
				return
			}
		}
	}
}
