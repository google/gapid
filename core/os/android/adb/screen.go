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
	"regexp"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
)

const (
	ErrScreenOnState   = fault.Const("Couldn't get screen-on state")
	ErrLockScreenState = fault.Const("Couldn't get lockscreen state")
)

var displayOnRegex = regexp.MustCompile("Display Power: state=(ON|DOZE|OFF)")
var displayReadyRegex = regexp.MustCompile("mDisplayReady=(true|false)")

// IsScreenOn returns true if the device's screen is currently on.
func (b *binding) IsScreenOn(ctx context.Context) (bool, error) {
	res, err := b.Shell("dumpsys", "power").Call(ctx)
	if err != nil {
		return false, err
	}
	switch displayOnRegex.FindString(res) {
	case "Display Power: state=ON":
		switch displayReadyRegex.FindString(res) {
		case "mDisplayReady=true":
			return true, nil
		case "mDisplayReady=false":
			return false, nil
		}
	case "Display Power: state=DOZE":
		return false, nil
	case "Display Power: state=OFF":
		return false, nil
	}
	return false, log.Err(ctx, ErrScreenOnState, "")
}

// TurnScreenOn turns the device's screen on.
func (b *binding) TurnScreenOn(ctx context.Context) error {
	if isOn, err := b.IsScreenOn(ctx); err != nil || isOn {
		return err
	}
	return b.KeyEvent(ctx, android.KeyCode_Power)
}

// TurnScreenOff turns the device's screen off.
func (b *binding) TurnScreenOff(ctx context.Context) error {
	if isOn, err := b.IsScreenOn(ctx); err != nil || !isOn {
		return err
	}
	return b.KeyEvent(ctx, android.KeyCode_Power)
}

var keyguardRegex = regexp.MustCompile("mKeyguardShowing=(true|false)")

// IsShowingLockscreen returns true if the device's lockscreen is currently showing.
func (b *binding) IsShowingLockscreen(ctx context.Context) (bool, error) {
	res, err := b.Shell("dumpsys", "activity", "activities").Call(ctx)
	if err != nil {
		return false, err
	}
	switch keyguardRegex.FindString(res) {
	case "mKeyguardShowing=true":
		return true, nil
	case "mKeyguardShowing=false":
		return false, nil
	}
	return false, log.Err(ctx, ErrLockScreenState, "")
}
