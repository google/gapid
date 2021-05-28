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

package android

import (
	"context"
	"fmt"
	"time"

	perfetto_pb "protos/perfetto/config"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
)

// Device extends the bind.Device interface with capabilities specific to android devices.
type Device interface {
	bind.DeviceWithShell
	// InstallAPK installs the specified APK to the device. If reinstall is true
	// and the package is already installed on the device then it will be replaced.
	InstallAPK(ctx context.Context, path string, reinstall bool, grantPermissions bool) error
	// SELinuxEnforcing returns true if the device is currently in a
	// SELinux enforcing mode, or false if the device is currently in a SELinux
	// permissive mode.
	SELinuxEnforcing(ctx context.Context) (bool, error)
	// SetSELinuxEnforcing changes the SELinux-enforcing mode.
	SetSELinuxEnforcing(ctx context.Context, enforce bool) error
	// StartActivity launches the specified action.
	StartActivity(ctx context.Context, a ActivityAction, extras ...ActionExtra) error
	// StartActivityForDebug launches an activity in debug mode.
	StartActivityForDebug(ctx context.Context, a ActivityAction, extras ...ActionExtra) error
	// StartService launches the specified service.
	StartService(ctx context.Context, a ServiceAction, extras ...ActionExtra) error
	// Pushes the local file to the remote one.
	Push(ctx context.Context, local, remote string) error
	// Pulls the remote file to the local one.
	Pull(ctx context.Context, remote, local string) error
	// KeyEvent simulates a key-event on the device.
	KeyEvent(ctx context.Context, key KeyCode) error
	// SendEvent simulates low-level user-input to the device.
	SendEvent(ctx context.Context, deviceID, eventType, eventCode, value int) error
	// SendTouch simulates touch-screen press or release.
	SendTouch(ctx context.Context, deviceID, x, y int, pressed bool)
	// GetTouchDimensions returns the resolution of the touch sensor.
	// This may be different to the dimensions of the LCD screen.
	GetTouchDimensions(ctx context.Context) (deviceID, minX, maxX, minY, maxY int, ok bool)
	// GetScreenDimensions returns the resolution of the display.
	GetScreenDimensions(ctx context.Context) (orientation, width, height int, ok bool)
	// InstalledPackages returns the sorted list of installed packages on the device.
	InstalledPackages(ctx context.Context) (InstalledPackages, error)
	// InstalledPackage returns information about a single installed package on the device.
	InstalledPackage(ctx context.Context, name string) (*InstalledPackage, error)
	// UnlockScreen returns true if it managed to turn on and unlock the screen.
	UnlockScreen(ctx context.Context) (bool, error)
	// Logcat writes all logcat messages reported by the device to the chan msgs,
	// blocking until the context is stopped.
	Logcat(ctx context.Context, msgs chan<- LogcatMessage) error
	// NativeBridgeABI returns the native ABI for the given emulated ABI for the
	// device by consulting the ro.dalvik.vm.isa.<emulated_isa>=<native_isa>
	// system properties. If there is no native ABI for the given ABI, then abi
	// is simply returned.
	NativeBridgeABI(ctx context.Context, abi *device.ABI) *device.ABI
	// ForceStop stops the everything associated with the given package.
	ForceStop(ctx context.Context, pkg string) error
	// SystemProperty returns the system property in string.
	SystemProperty(ctx context.Context, name string) (string, error)
	// SetSystemProperty sets the system property with the given string value.
	SetSystemProperty(ctx context.Context, name, value string) error
	// SystemSetting returns the system setting with the given namespaced key.
	SystemSetting(ctx context.Context, namespace, key string) (string, error)
	// SetSystemSetting sets the system setting with with the given namespaced
	// key to value.
	SetSystemSetting(ctx context.Context, namespace, key, value string) error
	// DeleteSystemSetting removes the system setting with with the given
	// namespaced key.
	DeleteSystemSetting(ctx context.Context, namespace, key string) error
	// StartPerfettoTrace starts a perfetto trace.
	StartPerfettoTrace(ctx context.Context, config *perfetto_pb.TraceConfig, out string, stop task.Signal, ready task.Task) error
	// SupportsAngle returns true if this device will work with ANGLE
	SupportsAngle(ctx context.Context) bool
}

// LogcatMessage represents a single logcat message.
type LogcatMessage struct {
	Timestamp time.Time
	Priority  LogcatPriority
	Tag       string
	ProcessID int
	ThreadID  int
	Message   string
}

// Log writes the LogcatMessage to ctx with the corresponding message severity.
func (m LogcatMessage) Log(ctx context.Context) {
	// Override the timestamping function to replicate the logcat timestamp
	ctx = log.PutClock(ctx, log.FixedClock(m.Timestamp))
	ctx = log.PutTag(ctx, m.Tag)
	ctx = log.PutProcess(ctx, "logcat")
	ctx = log.V{
		"pid": m.ProcessID,
		"tid": m.ThreadID,
	}.Bind(ctx)
	log.From(ctx).Log(m.Priority.Severity(), false, m.Message)
}

// LogcatPriority represents the priority of a logcat message.
type LogcatPriority int

const (
	// Verbose represents the 'V' logcat message priority.
	Verbose = LogcatPriority(iota)
	// Debug represents the 'D' logcat message priority.
	Debug
	// Info represents the 'I' logcat message priority.
	Info
	// Warning represents the 'W' logcat message priority.
	Warning
	// Error represents the 'E' logcat message priority.
	Error
	// Fatal represents the 'F' logcat message priority.
	Fatal
)

// Severity returns a Severity that closely corresponds to the priority.
func (p LogcatPriority) Severity() log.Severity {
	switch p {
	case Verbose:
		return log.Verbose
	case Debug:
		return log.Debug
	case Info:
		return log.Info
	case Warning:
		return log.Warning
	case Error:
		return log.Error
	case Fatal:
		return log.Fatal
	default:
		panic(fmt.Errorf("Unknown LogcatPriority %v", p))
	}
}
