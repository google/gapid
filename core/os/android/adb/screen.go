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
	"time"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
)

const (
	screenOnLocked = iota
	screenOnUnlocked
	screenOffLocked
	screenOffUnlocked
)

const (
	keyEventEffectDelay time.Duration = 500 * time.Millisecond
)

const (
	ErrScreenState = fault.Const("Couldn't get screen state")
)

var screenStateRegex = regexp.MustCompile("mScreenState=(ON|OFF)_(LOCKED|UNLOCKED)")

// getScreenState returns the screen state
func (b *binding) getScreenState(ctx context.Context) (int, error) {
	res, err := b.Shell("dumpsys", "nfc").Call(ctx)
	if err != nil {
		return -1, err
	}
	switch screenStateRegex.FindString(res) {
	case "mScreenState=ON_LOCKED":
		return screenOnLocked, nil
	case "mScreenState=ON_UNLOCKED":
		return screenOnUnlocked, nil
	case "mScreenState=OFF_LOCKED":
		return screenOffLocked, nil
	case "mScreenState=OFF_UNLOCKED":
		return screenOffUnlocked, nil
	}
	return -1, log.Err(ctx, ErrScreenState, "")
}

// isScreenUnlocked returns true is the device's screen is on and unlocked.
func (b *binding) isScreenUnlocked(ctx context.Context) (bool, error) {
	screenState, err := b.getScreenState(ctx)
	if err != nil {
		return false, err
	}
	return screenState == screenOnUnlocked, nil
}

// UnlockScreen returns true if it managed to turn on and unlock the screen.
func (b *binding) UnlockScreen(ctx context.Context) (bool, error) {
	// Use wakeup key event to put screen on, and menu key event to
	// try to unlock. Exit early as soon as the screen is on and
	// unlocked.

	// KeyEvent() eventually calls adb shell commands, which are
	// asynchronous. There is no nice way to know when they actually
	// terminated, hence the sleep here to make sure the key events
	// have time to affect the screen state.

	screenState, err := b.getScreenState(ctx)
	if err != nil {
		return false, err
	}
	switch screenState {
	case screenOnUnlocked:
		return true, nil

	case screenOffUnlocked:
		if err := b.KeyEvent(ctx, android.KeyCode_Wakeup); err != nil {
			return false, err
		}
		time.Sleep(keyEventEffectDelay)
		return b.isScreenUnlocked(ctx)

	case screenOffLocked:
		if err := b.KeyEvent(ctx, android.KeyCode_Wakeup); err != nil {
			return false, err
		}
		fallthrough
	case screenOnLocked:
		if err := b.KeyEvent(ctx, android.KeyCode_Menu); err != nil {
			return false, err
		}
		time.Sleep(keyEventEffectDelay)
		return b.isScreenUnlocked(ctx)
	}
	return false, log.Err(ctx, ErrScreenState, "")
}
