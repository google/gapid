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
	"strconv"
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/core/vulkan/loader"
	"github.com/google/gapid/gapidapk"
	"github.com/google/gapid/gapis/replay/protocol"
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

func (s *session) init(ctx context.Context, d bind.Device, abi *device.ABI, launchArgs []string) error {
	defer close(s.inited)

	var err error
	if host.Instance(ctx).SameAs(d.Instance()) {
		err = s.newHost(ctx, d, launchArgs)
	} else if d, ok := d.(adb.Device); ok {
		err = s.newADB(ctx, d, abi)
	} else {
		err = log.Errf(ctx, nil, "Cannot connect to device type %v", d)
	}
	if err != nil {
		s.close()
		return err
	}

	crash.Go(func() { s.heartbeat(ctx, sessionTimeout/2) })
	return nil
}

// newHost spawns and returns a new GAPIR instance on the host machine.
func (s *session) newHost(ctx context.Context, d bind.Device, launchArgs []string) error {
	authToken := auth.GenToken()
	args := []string{
		"--idle-timeout-ms", strconv.Itoa(int(sessionTimeout / time.Millisecond)),
		"--auth-token", string(authToken),
	}
	args = append(args, launchArgs...)

	gapir, err := layout.Gapir(ctx)
	if err != nil {
		log.F(ctx, "Couldn't locate gapir executable: %v", err)
		return nil
	}

	env := shell.CloneEnv()
	if _, err := loader.SetupReplay(ctx, env); err != nil {
		return err
	}

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
			log.From(ctx).Log(severity, line)
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
	port, err := process.Start(ctx, gapir.System(), process.StartOptions{
		Env:    env,
		Args:   args,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		log.E(ctx, "Starting gapir. Error: %v", err)
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

func (s *session) newADB(ctx context.Context, d adb.Device, abi *device.ABI) error {
	ctx = log.V{"abi": abi}.Bind(ctx)

	log.I(ctx, "Checking gapid.apk is installed...")
	apk, err := gapidapk.EnsureInstalled(ctx, d, abi)
	if err != nil {
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
	if err := d.Forward(ctx, localPort, adb.NamedAbstractSocket(socket)); err != nil {
		return log.Err(ctx, err, "Forwarding port")
	}
	s.onClose(func() { d.RemoveForward(ctx, localPort) })

	log.I(ctx, "Launching GAPIR...")
	if err := d.StartActivity(ctx, *apk.ActivityActions[0]); err != nil {
		return err
	}

	log.I(ctx, "Waiting for connection to GAPIR...")
	for i := 0; i < 10; i++ {
		if _, err := s.ping(ctx); err == nil {
			log.I(ctx, "Connected to GAPIR")
			return nil
		}
		time.Sleep(time.Second)
	}

	return log.Err(ctx, nil, "Timeout waiting for connection")
}

func (s *session) connect(ctx context.Context) (io.ReadWriteCloser, error) {
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

func (s *session) ping(ctx context.Context) (time.Duration, error) {
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
