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
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/gapidapk"
	"github.com/google/gapid/gapis/replay/protocol"
)

var (
	// LogPath is the full filepath of the logfile new instances of gapir should write to.
	LogPath string

	// GapirPath is the full filepath to the gapir executable.
	GapirPath file.Path

	// VirtualSwapchainLayerPath is the path to the virtual swapchain layer for
	// Vulkan application replaying.
	VirtualSwapchainLayerPath file.Path
)

const sessionTimeout = time.Second * 10

type session struct {
	device   bind.Device
	port     int
	auth     auth.Token
	closeCBs []func()
	inited   chan struct{}
}

func newSession(d bind.Device) *session {
	return &session{device: d, inited: make(chan struct{})}
}

func (s *session) init(ctx log.Context, d bind.Device, abi *device.ABI) error {
	defer close(s.inited)

	var err error
	if device.Host(ctx).SameAs(d.Instance()) {
		err = s.newHost(ctx, d)
	} else if d, ok := d.(adb.Device); ok {
		err = s.newADB(ctx, d, abi)
	} else {
		err = cause.Explainf(ctx, nil, "Cannot connect to device type %v", d)
	}
	if err != nil {
		s.close()
		return err
	}

	go s.heartbeat(ctx, sessionTimeout/2)
	return nil
}

// newHost spawns and returns a new GAPIR instance on the host machine.
func (s *session) newHost(ctx log.Context, d bind.Device, gapirArgs ...string) error {
	authToken := auth.GenToken()
	args := []string{
		"--idle-timeout-ms", strconv.Itoa(int(sessionTimeout / time.Millisecond)),
		"--auth-token", string(authToken),
	}
	args = append(args, gapirArgs...)
	if LogPath != "" {
		args = append(args, "--log", LogPath)
	}

	if !GapirPath.Exists() {
		jot.Fail(ctx, nil, "Couldn't locate gapir executable")
		return nil
	}

	// Set the VK_LAYER_PATH for replaying Vulkan applications.
	var extraEnv = map[string][]string{
		"VK_LAYER_PATH": {VirtualSwapchainLayerPath.String()},
	}

	port, err := process.Start(ctx, GapirPath.System(), extraEnv, args...)
	if err != nil {
		jot.Fail(ctx, err, "Starting gapir")
		return nil
	}

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

func (s *session) newADB(ctx log.Context, d adb.Device, abi *device.ABI) error {
	ctx = ctx.V("abi", abi)

	ctx.Info().Log("Checking gapid.apk is installed...")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
		return err
	}

	ctx.Info().Log("Setting up port forwarding...")
	localPort, err := adb.LocalFreeTCPPort()
	if err != nil {
		return cause.Explain(ctx, err, "Finding free port")
	}
	s.port = int(localPort)
	socket, ok := socketNames[abi.Architecture]
	ctx = ctx.S("socket", socket)
	if !ok {
		return cause.Explain(ctx, nil, "Unsupported architecture").With("architecture", abi.Architecture)
	}
	if err := d.Forward(ctx, localPort, adb.NamedAbstractSocket(socket)); err != nil {
		return cause.Explain(ctx, err, "Forwarding port")
	}
	s.onClose(func() { d.RemoveForward(ctx, localPort) })

	ctx.Info().Log("Launching GAPIR...")
	if err := d.StartActivity(ctx, *apk.ActivityActions[0]); err != nil {
		return err
	}

	ctx.Info().Log("Waiting for connection to GAPIR...")
	for i := 0; i < 10; i++ {
		if _, err := s.ping(ctx); err == nil {
			ctx.Info().Log("Connected to GAPIR")
			return nil
		}
		time.Sleep(time.Second)
	}

	return cause.Explain(ctx, nil, "Timeout waiting for connection")
}

func (s *session) connect(ctx log.Context) (io.ReadWriteCloser, error) {
	<-s.inited
	return process.Connect(s.port, s.auth)
}

func (s *session) onClose(f func()) {
	s.closeCBs = append(s.closeCBs, f)
}

func (s *session) close() {
	for _, f := range s.closeCBs {
		f()
	}
	s.closeCBs = nil
}

func (s *session) ping(ctx log.Context) (time.Duration, error) {
	connection, err := process.Connect(s.port, s.auth)
	if err != nil {
		return 0, err
	}
	defer connection.Close()
	w := endian.Writer(connection, device.LittleEndian) // TODO: Endianness
	r := endian.Reader(connection, device.LittleEndian) // TODO: Endianness
	start := time.Now()
	if w.Uint8(uint8(protocol.ConnectionType_Ping)); w.Error() != nil {
		return 0, w.Error()
	}
	if response := r.String(); w.Error() != nil || response != "PONG" {
		return 0, fmt.Errorf("Expected 'PONG', got: '%v' (err: %v)", response, r.Error())
	}
	return time.Since(start), nil
}

func (s *session) heartbeat(ctx log.Context, pingInterval time.Duration) {
	defer s.close()
	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case <-time.Tick(pingInterval):
			_, err := s.ping(ctx)
			if err != nil {
				jot.Fail(ctx, err, "Error sending keep-alive ping")
				return
			}
		}
	}
}
