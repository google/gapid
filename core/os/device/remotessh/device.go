// Copyright (C) 2018 Google Inc.
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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/gapid/core/app/layout"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
)

// Device extends the bind.Device interface with capabilities specific to
// remote SSH clients
type Device interface {
	bind.Device
	// PushFile will transfer the local file at sourcePath to the remote
	// machine at destPath
	PushFile(ctx context.Context, sourcePath, destPath string) error
	// Forward will forward the local port to the remote port
	Forward(ctx context.Context, local, device uint16) error
	// RemoveForward removes a port forward from local
	RemoveForward(ctx context.Context, local uint16) error
	// MakeTempDir makes a temporary directory, and returns the
	// path, as well as a function to call to clean it up.
	MakeTempDir(ctx context.Context) (string, func(ctx context.Context), error)
	// WriteFile writes the given file into the given location on the remote device
	WriteFile(ctx context.Context, contents io.Reader, mode string, destPath string) error
	// ForwardPort forwards the remote port. It automatically selects an open
	// local port.
	ForwardPort(ctx context.Context, remoteport int) (int, error)
}

// binding represents an attached SSH client.
type binding struct {
	bind.Simple

	connection    *ssh.Client
	configuration *Configuration
	// We duplicate OS here because we need to use it
	// before we get the rest of the information
	os device.OSKind
}

var _ Device = &binding{}

func (binding) Forward(ctx context.Context, local, device uint16) error {
	return nil
}

func (binding) RemoveForward(ctx context.Context, local uint16) error {
	return nil
}

// Devices returns the list of reachable SSH devices.
func Devices(ctx context.Context, configuration io.Reader) ([]bind.Device, error) {
	configurations, err := ReadConfiguration(configuration)
	if err != nil {
		return nil, err
	}
	devices := make([]bind.Device, 0, len(configurations))

	for _, cfg := range configurations {
		if device, err := GetConnectedDevice(ctx, cfg); err == nil {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

func GetSSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func GetPrivateKeyAuth(path string) (ssh.AuthMethod, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(bytes)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// GetConnectedDevice returns a device that has an option connection.
func GetConnectedDevice(ctx context.Context, c Configuration) (Device, error) {
	auths := []ssh.AuthMethod{}
	if agent := GetSSHAgent(); agent != nil {
		auths = append(auths, agent)
	}

	if c.Keyfile != "" {
		if auth, err := GetPrivateKeyAuth(c.Keyfile); err == nil {
			auths = append(auths, auth)
		}
	}

	if len(auths) == 0 {
		return nil, fmt.Errorf("No valid authentication method for SSH connection %s", c.Name)
	}

	hosts, err := knownhosts.New(c.KnownHosts)
	if err != nil {
		return nil, fmt.Errorf("Could not read known hosts %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.User,
		Auth:            auths,
		HostKeyCallback: hosts,
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), sshConfig)
	if err != nil {
		return nil, err
	}

	b := &binding{
		connection:    connection,
		configuration: &c,
		Simple: bind.Simple{
			To: &device.Instance{
				Serial:        "",
				Configuration: &device.Configuration{},
			},
			LastStatus: bind.Status_Online,
		},
	}

	kind := device.UnknownOS

	// Try to get the OS string for Mac/Linux
	if osName, err := b.Shell("uname", "-a").Call(ctx); err == nil {
		if strings.Contains(osName, "Darwin") {
			kind = device.OSX
		} else if strings.Contains(osName, "Linux") {
			kind = device.Linux
		}
	}

	if kind == device.UnknownOS {
		// Try to get the OS string for Windows
		if osName, err := b.Shell("ver").Call(ctx); err == nil {
			if strings.Contains(osName, "Windows") {
				kind = device.Windows
			}
		}
	}

	if kind == device.UnknownOS {
		return nil, fmt.Errorf("Could not determine unix type")
	}
	b.os = kind

	dir, cleanup, err := b.MakeTempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup(ctx)

	localDeviceInfo, err := layout.DeviceInfo(ctx, b.os)
	if err != nil {
		return nil, err
	}

	if err = b.PushFile(ctx, localDeviceInfo.System(), dir+"/device_info"); err != nil {
		return nil, err
	}

	devInfo, err := b.Shell("./device_info").In(dir).Call(ctx)

	if err != nil {
		return nil, err
	}

	var device device.Instance

	if err := jsonpb.Unmarshal(bytes.NewReader([]byte(devInfo)), &device); err != nil {
		panic(err)
	}

	b.To = &device
	b.To.Name = c.Name
	return b, nil
}
