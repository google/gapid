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

package adb

import (
	"context"
	"fmt"
	"net"

	"github.com/google/gapid/core/context/keys"
)

// Port is the interface for sockets ports that can be forwarded from an Android
// Device to the local machine.
type Port interface {
	// adbForwardString returns the "port specification" for the adb forward
	// command. See: http://developer.android.com/tools/help/adb.html#commandsummary
	// This method is hidden as its only use is for adb command-line parameters,
	// which this package abstracts.
	adbForwardString() string
}

// TCPPort represents a TCP/IP port on either the local machine or Android
// device. TCPPort implements the Port interface.
type TCPPort int

func (p TCPPort) adbForwardString() string {
	return fmt.Sprintf("tcp:%d", p)
}

// LocalFreeTCPPort returns a currently free TCP port on the localhost.
// There are two potential issues with using this for ADB port forwarding:
// * There is the potential for the port to be taken between the function
//   returning and the port being used by ADB.
// * The system _may_ hold on to the socket after it has been told to close.
// Because of these issues, there is a potential for flakiness.
func LocalFreeTCPPort() (TCPPort, error) {
	socket, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer socket.Close()
	return TCPPort(socket.Addr().(*net.TCPAddr).Port), nil
}

// NamedAbstractSocket represents an abstract UNIX domain socket name on either
// the local machine or Android device. NamedAbstractSocket implements the Port
// interface.
type NamedAbstractSocket string

func (p NamedAbstractSocket) adbForwardString() string {
	return fmt.Sprintf("localabstract:%s", p)
}

// NamedFileSystemSocket represents an file system UNIX domain socket name on
// either the local machine or Android device. NamedFileSystemSocket implements
// the Port interface.
type NamedFileSystemSocket string

func (p NamedFileSystemSocket) adbForwardString() string {
	return fmt.Sprintf("localfilesystem:%s", p)
}

// Forward will forward the specified device Port to the specified local Port.
func (b *binding) Forward(ctx context.Context, local, device Port) error {
	return b.Command("forward", local.adbForwardString(), device.adbForwardString()).Run(ctx)
}

// RemoveForward removes a port forward made by Forward.
func (b *binding) RemoveForward(ctx context.Context, local Port) error {
	// Clone context to ignore cancellation.
	ctx = keys.Clone(context.Background(), ctx)
	return b.Command("forward", "--remove", local.adbForwardString()).Run(ctx)
}

// SetupLocalPort makes sure that the given port can be accessed on localhost
// It returns a new port number to connect to on localhost
func (b *binding) SetupLocalPort(ctx context.Context, port int) (int, error) {
	localPort, err := LocalFreeTCPPort()
	if err != nil {
		return 0, err
	}
	return int(localPort), b.Forward(ctx, localPort, TCPPort(port))
}
