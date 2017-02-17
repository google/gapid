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

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
)

// Device extends the bind.Device interface with capabilities specific to android devices.
type Device interface {
	bind.Device
	// InstallAPK installs the specified APK to the device. If reinstall is true
	// and the package is already installed on the device then it will be replaced.
	InstallAPK(ctx log.Context, path string, reinstall bool, grantPermissions bool) error
	// SELinuxEnforcing returns true if the device is currently in a
	// SELinux enforcing mode, or false if the device is currently in a SELinux
	// permissive mode.
	SELinuxEnforcing(ctx log.Context) (bool, error)
	// SetSELinuxEnforcing changes the SELinux-enforcing mode.
	SetSELinuxEnforcing(ctx log.Context, enforce bool) error
	// StartActivity launches the specified action.
	StartActivity(ctx log.Context, a ActivityAction, extras ...ActionExtra) error
	// StartActivityForDebug launches an activity in debug mode.
	StartActivityForDebug(ctx log.Context, a ActivityAction, extras ...ActionExtra) error
	// StartService launches the specified service.
	StartService(ctx log.Context, a ServiceAction, extras ...ActionExtra) error
	// Pushes the local file to the remote one.
	Push(ctx log.Context, local, remote string) error
	// Pulls the remote file to the local one.
	Pull(ctx log.Context, remote, local string) error
	// KeyEvent simulates a key-event on the device.
	KeyEvent(ctx log.Context, key KeyCode) error
	// SendEvent simulates low-level user-input to the device.
	SendEvent(ctx log.Context, deviceID, eventType, eventCode, value int) error
	// SendTouch simulates touch-screen press or release.
	SendTouch(ctx log.Context, deviceID, x, y int, pressed bool)
	// GetTouchDimensions returns the resolution of the touch sensor.
	// This may be different to the dimensions of the LCD screen.
	GetTouchDimensions(ctx log.Context) (deviceID, minX, maxX, minY, maxY int, ok bool)
	// GetScreenDimensions returns the resolution of the display.
	GetScreenDimensions(ctx log.Context) (orientation, width, height int, ok bool)
	// InstalledPackages returns the sorted list of installed packages on the device.
	InstalledPackages(ctx log.Context) (InstalledPackages, error)
	// IsScreenOn returns true if the device's screen is currently on.
	IsScreenOn(ctx log.Context) (bool, error)
	// TurnScreenOn turns the device's screen on.
	TurnScreenOn(ctx log.Context) error
	// TurnScreenOff turns the device's screen off.
	TurnScreenOff(ctx log.Context) error
	// IsShowingLockscreen returns true if the device's lockscreen is currently showing.
	IsShowingLockscreen(ctx log.Context) (bool, error)
	// Logcat writes all logcat messages reported by the device to the chan msgs,
	// blocking until the context is stopped.
	Logcat(ctx log.Context, msgs chan<- LogcatMessage) error
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
	ctx = memo.Timestamp(ctx, func() time.Time { return m.Timestamp })
	jot.At(ctx, m.Priority.Severity()).
		With("tag", m.Tag).
		With("pid", m.ProcessID).
		With("tid", m.ThreadID).
		Print(m.Message)
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
func (p LogcatPriority) Severity() severity.Level {
	switch p {
	case Verbose:
		return severity.Debug
	case Debug:
		return severity.Info
	case Info:
		return severity.Notice
	case Warning:
		return severity.Warning
	case Error:
		return severity.Error
	case Fatal:
		return severity.Critical
	default:
		panic(fmt.Errorf("Unknown LogcatPriority %v", p))
	}
}
