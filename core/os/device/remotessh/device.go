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
	"sync"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/shell"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Device extends the bind.Device interface with capabilities specific to
// remote SSH clients
type Device interface {
	bind.Desktop
	// PullFile will transfer the remote file at sourcePath to the local
	// machine at destPath
	PullFile(ctx context.Context, sourcePath, destPath string) error
	// GetFilePermissions gets the unix permissions for the remote file
	GetFilePermissions(ctx context.Context, path string) (os.FileMode, error)
	// DefaultReplayCacheDir returns the default path for replay resource caches
	DefaultReplayCacheDir() string
}

const (
	// Frequency at which to print scan errors
	printScanErrorsEveryNSeconds = 120
)

// MaxNumberOfSSHConnections defines the max number of ssh connections to each
// ssh remote device that can be used to run commands concurrently.
const MaxNumberOfSSHConnections = 15

// binding represents an attached SSH client.
type binding struct {
	bind.Simple

	connection    *ssh.Client
	configuration *Configuration
	env           *shell.Env
	// We duplicate OS here because we need to use it
	// before we get the rest of the information
	os device.OSKind

	// pool to limit the maximum number of connections
	ch chan int
}

type pooledSession struct {
	ch      chan int
	session *ssh.Session
}

func (p *pooledSession) kill() error {
	select {
	case <-p.ch:
	default:
	}
	<-p.ch
	return p.session.Signal(ssh.SIGSEGV)
}

func (p *pooledSession) wait() error {
	ret := p.session.Wait()
	select {
	case <-p.ch:
	default:
	}
	return ret
}

func newBinding(conn *ssh.Client, conf *Configuration, env *shell.Env) *binding {
	b := &binding{
		connection:    conn,
		configuration: conf,
		env:           env,
		ch:            make(chan int, MaxNumberOfSSHConnections),
		Simple: bind.Simple{
			To: &device.Instance{
				Serial:        "",
				Configuration: &device.Configuration{},
			},
			LastStatus: bind.Status_Online,
		},
	}
	return b
}

func (b *binding) newPooledSession() (*pooledSession, error) {
	b.ch <- int(0)
	session, err := b.connection.NewSession()
	if err != nil {
		<-b.ch
		err = fmt.Errorf("New SSH Session Error: %v, Current maximum number of ssh connections GAPID can issue to each remote device is: %v", err, MaxNumberOfSSHConnections)
		return nil, err
	}
	return &pooledSession{
		ch:      b.ch,
		session: session,
	}, nil
}

// Interface check
var _ Device = &binding{}

var (
	// Registry of all the discovered devices.
	registry = bind.NewRegistry()

	// cache is a map of device names to fully resolved bindings.
	cache      = map[string]*binding{}
	cacheMutex sync.Mutex // Guards cache.
)

func readConfigs(rcs []io.ReadCloser) ([]Configuration, error) {
	defer func() {
		for _, rc := range rcs {
			rc.Close()
		}
	}()
	configs := []Configuration{}
	for _, rc := range rcs {
		configurations, err := ReadConfigurations(rc)
		if err != nil {
			return nil, err
		}
		configs = append(configs, configurations...)
	}
	return configs, nil
}

// Monitor updates the registry with devices that are added and removed at the
// specified interval. Monitor returns once the context is cancelled.
func Monitor(ctx context.Context, r *bind.Registry, interval time.Duration, conf func() ([]io.ReadCloser, error)) error {
	unlisten := registry.Listen(bind.NewDeviceListener(r.AddDevice, r.RemoveDevice))
	defer unlisten()

	for _, d := range registry.Devices() {
		r.AddDevice(ctx, d)
	}

	var lastErrorPrinted time.Time
	for {
		if err := func() error {
			rcs, err := conf()
			if err != nil {
				return err
			}
			configs, err := readConfigs(rcs)
			if err != nil {
				return err
			}
			return scanDevices(ctx, configs)
		}(); err != nil {
			if time.Since(lastErrorPrinted).Seconds() > printScanErrorsEveryNSeconds {
				log.E(ctx, "Error scanning remote ssh devices: %v", err)
				lastErrorPrinted = time.Now()
			}
		} else {
			lastErrorPrinted = time.Time{}
		}

		select {
		case <-task.ShouldStop(ctx):
			return nil
		case <-time.After(interval):
		}
	}
}

// Devices returns the list of attached ssh devices.
func Devices(ctx context.Context, configuration []io.ReadCloser) ([]bind.Device, error) {
	configs, err := readConfigs(configuration)
	if err != nil {
		return nil, err
	}

	if err := scanDevices(ctx, configs); err != nil {
		return nil, err
	}
	devs := registry.Devices()
	out := make([]bind.Device, len(devs))
	for i, d := range devs {
		out[i] = d
	}
	return out, nil
}

func scanDevices(ctx context.Context, configurations []Configuration) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	allConfigs := make(map[string]bool)

	for _, cfg := range configurations {
		allConfigs[cfg.Name] = true

		// If this device already exists, see if we
		// can/have to remove it
		if cached, ok := cache[cfg.Name]; ok {
			if !deviceStillConnected(ctx, cached) {
				delete(cache, cfg.Name)
				registry.RemoveDevice(ctx, cached)
			}
		} else {
			if device, err := GetConnectedDevice(ctx, cfg); err == nil {
				registry.AddDevice(ctx, device)
				cache[cfg.Name] = device.(*binding)
			} else {
				log.E(ctx, "Failed to connect to remote device %s: %v", cfg.Name, err)
			}
		}
	}

	for name, dev := range cache {
		if _, ok := allConfigs[name]; !ok {
			delete(cache, name)
			registry.RemoveDevice(ctx, dev)
		}
	}
	return nil
}

func deviceStillConnected(ctx context.Context, d *binding) bool {
	return d.Status(ctx) == bind.Status_Online
}

// getSSHAgent returns a connection to a local SSH agent, if one exists.
func getSSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

// This returns an SSH auth for the given private key.
// It will fail if the private key was encrypted.
func getPrivateKeyAuth(path string) (ssh.AuthMethod, error) {
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

// GetConnectedDevice returns a device that matches the given configuration.
func GetConnectedDevice(ctx context.Context, c Configuration) (Device, error) {
	auths := []ssh.AuthMethod{}

	if c.Keyfile != "" {
		// This returns an SSH auth for the given private key.
		// It will fail if the private key was encrypted.
		if auth, err := getPrivateKeyAuth(c.Keyfile); err == nil {
			auths = append(auths, auth)
		}
	}

	if agent := getSSHAgent(); agent != nil {
		auths = append(auths, agent)
	}

	if len(auths) == 0 {
		return nil, log.Errf(ctx, nil, "No valid authentication method for SSH connection %s", c.Name)
	}

	hosts, err := knownhosts.New(c.KnownHosts)
	if err != nil {
		return nil, log.Errf(ctx, err, "Could not read known hosts")
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.User,
		Auth:            auths,
		HostKeyCallback: hosts,
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), sshConfig)
	if err != nil {
		return nil, log.Errf(ctx, err, "Dial tcp: %s:%d with sshConfig: %v failed", c.Host, c.Port, sshConfig)
	}

	env := shell.NewEnv()

	for _, e := range c.Env {
		env.Add(e)
	}

	b := newBinding(connection, &c, env)

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
		return nil, log.Errf(ctx, nil, "Could not determine unix type")
	}
	b.os = kind
	dir, cleanup, err := b.TempDir(ctx)
	if err != nil {
		return nil, log.Errf(ctx, err, "Could not make temporary directory")
	}
	defer cleanup(ctx)

	localDeviceInfo, err := layout.DeviceInfo(ctx, b.os)
	if err != nil {
		return nil, log.Errf(ctx, err, "Could not get device info")
	}

	if err = b.PushFile(ctx, localDeviceInfo.System(), dir+"/device-info"); err != nil {
		return nil, log.Errf(ctx, err, "Error running: './device-info'")
	}

	stderr := bytes.Buffer{}
	stdout := bytes.Buffer{}

	err = b.Shell("./device-info").In(dir).Capture(&stdout, &stderr).Run(ctx)

	if err != nil {
		return nil, err
	}

	if stderr.String() != "" {
		log.W(ctx, "Deviceinfo succeeded, but returned error string %s", stderr.String())
	}
	devInfo := stdout.String()

	var device device.Instance

	if err := jsonpb.Unmarshal(bytes.NewReader([]byte(devInfo)), &device); err != nil {
		panic(err)
	}

	device.Name = c.Name
	device.GenID()
	for i := range device.ID.Data {
		// Flip some bits, since if you have a local & ssh device
		// they would otherwise be the same
		device.ID.Data[i] = 0x10 ^ device.ID.Data[i]
	}

	b.To = &device

	return b, nil
}

// DefaultReplayCacheDir implements Device interface
func (b *binding) DefaultReplayCacheDir() string {
	return ""
}
