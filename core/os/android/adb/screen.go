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
	// Some slow devices can take up to a second to properly get the
	// screen in an unlocked state, so use 1.5 seconds to be on the
	// safe side.
	keyEventEffectDelay time.Duration = 1500 * time.Millisecond
)

const (
	ErrScreenState = fault.Const("Couldn't get screen state")
)

var screenStateRegex = regexp.MustCompile("mAwake=(true|false)")
var lockStateRegex = regexp.MustCompile("(?:mDreamingLockscreen|mShowingLockscreen)=(true|false)")

// getScreenState returns the screen state
func (b *binding) getScreenState(ctx context.Context) (int, error) {
	res, err := b.Shell("dumpsys", "window").Call(ctx)
	if err != nil {
		return -1, err
	}

	screenStateMatch := screenStateRegex.FindStringSubmatch(res)
	screenState := false
	switch {
	case screenStateMatch == nil:
		return -1, log.Err(ctx, ErrScreenState, "")
	case screenStateMatch[1] == "true":
		screenState = true
	}

	lockStateMatch := lockStateRegex.FindStringSubmatch(res)
	lockState := false
	switch {
	case lockStateMatch == nil:
		return -1, log.Err(ctx, ErrScreenState, "Unable to determine lock state")
	case lockStateMatch[1] == "true":
		lockState = true
	}

	switch {
	case screenState && lockState:
		return screenOnLocked, nil
	case screenState:
		return screenOnUnlocked, nil
	case lockState:
		return screenOffLocked, nil
	default:
		return screenOffUnlocked, nil
	}
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
	screenState, err := b.getScreenState(ctx)
	if err != nil {
		return false, err
	}
	switch screenState {
	case screenOnUnlocked:
		return true, nil

	default:
		// Devices may do unexpected transitions between screen
		// states, so this code does not try to be smart about
		// expected state changes: unless screenOnUnlocked, apply the
		// wakeup (put screen on) and dismiss-keyguard (unlock screen
		// if no credentials required) sequence.
		if err := b.KeyEvent(ctx, android.KeyCode_Wakeup); err != nil {
			return false, err
		}
		if err := b.Shell("wm", "dismiss-keyguard").Run(ctx); err != nil {
			return false, err
		}
		// A sleep here is necessary to make sure the above commands had enough
		// time to affect the screen state.
		time.Sleep(keyEventEffectDelay)
		return b.isScreenUnlocked(ctx)
	}
}
