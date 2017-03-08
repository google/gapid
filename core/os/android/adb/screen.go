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

var screenOnRegex = regexp.MustCompile("mScreenOnFully=(true|false)")

// IsScreenOn returns true if the device's screen is currently on.
func (b *binding) IsScreenOn(ctx context.Context) (bool, error) {
	res, err := b.Shell("dumpsys", "window", "policy").Call(ctx)
	if err != nil {
		return false, err
	}
	switch screenOnRegex.FindString(res) {
	case "mScreenOnFully=true":
		return true, nil
	case "mScreenOnFully=false":
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

var lockscreenRegex = regexp.MustCompile("mShowingLockscreen=(true|false)")

// IsShowingLockscreen returns true if the device's lockscreen is currently showing.
func (b *binding) IsShowingLockscreen(ctx context.Context) (bool, error) {
	res, err := b.Shell("dumpsys", "window", "policy").Call(ctx)
	if err != nil {
		return false, err
	}
	switch lockscreenRegex.FindString(res) {
	case "mShowingLockscreen=true":
		return true, nil
	case "mShowingLockscreen=false":
		return false, nil
	}
	return false, log.Err(ctx, ErrLockScreenState, "")
}
