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

package remotessh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"golang.org/x/crypto/ssh"
)

// Port is the interface for socket ports that can be tunneled through SSH.
type Port interface {
	// Dial returns a net.Conn to this port via an SSH tunnel.
	dial(conn *ssh.Client) (net.Conn, error)
}

// TCPPort represents a TCP/IP port on the remote machine.
type TCPPort int

func (p TCPPort) dial(conn *ssh.Client) (net.Conn, error) {
	return conn.Dial("tcp", fmt.Sprintf("localhost:", p))
}

// UnixPort represents a Unix port on the remote machine.
type UnixPort string

func (p UnixPort) dial(conn *ssh.Client) (net.Conn, error) {
	return conn.Dial("unix", string(p))
}

// doTunnel tunnels a single connection through the SSH connection.
func (b binding) doTunnel(ctx context.Context, local net.Conn, remotePort Port) error {
	remote, err := b.connection.Dial("tcp", fmt.Sprintf("localhost:%d", remotePort))
	if err != nil {
		local.Close()
		return err
	}

	wg := sync.WaitGroup{}

	copy := func(writer net.Conn, reader net.Conn) {
		// Use the same buffer size used in io.Copy
		buf := make([]byte, 32*1024)
		var err error
		for {
			nr, er := reader.Read(buf)
			if nr > 0 {
				nw, ew := writer.Write(buf[0:nr])
				if ew != nil {
					err = ew
					break
				}
				if nr != nw {
					err = fmt.Errorf("short write")
					break
				}
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break
			}
		}
		writer.Close()
		if err != nil {
			log.E(ctx, "Copy Error %s", err)
		}
		wg.Done()
	}

	wg.Add(2)
	crash.Go(func() { copy(local, remote) })
	crash.Go(func() { copy(remote, local) })

	crash.Go(func() {
		defer local.Close()
		defer remote.Close()
		wg.Wait()
	})
	return nil
}

// SetupLocalPort forwards a local TCP port to the remote machine on the remote port.
// The local port that was opened is returned.
func (b binding) SetupLocalPort(ctx context.Context, remotePort int) (int, error) {
	listener, err := net.Listen("tcp", ":0")

	if err != nil {
		return 0, err
	}
	crash.Go(func() {
		<-task.ShouldStop(ctx)
		listener.Close()
	})
	crash.Go(func() {
		defer listener.Close()
		for {
			local, err := listener.Accept()
			if err != nil {
				return
			}
			if err = b.doTunnel(ctx, local, TCPPort(remotePort)); err != nil {
				return
			}
		}
	})

	return listener.Addr().(*net.TCPAddr).Port, nil
}
