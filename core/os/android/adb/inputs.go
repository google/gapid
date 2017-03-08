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
	"regexp"
	"strconv"

	"github.com/google/gapid/core/os/android"
)

// Event types
const (
	EV_ABS = 3
	EV_SYN = 0
)

// Event codes
const (
	ABS_MT_TRACKING_ID = 57
	ABS_MT_POSITION_X  = 53
	ABS_MT_POSITION_Y  = 54
	ABS_MT_PRESSURE    = 58
	ABS_MT_TOUCH_MAJOR = 48
	SYN_REPORT         = 0
)

// KeyEvent simulates a key-event on the device.
func (b *binding) KeyEvent(ctx context.Context, key android.KeyCode) error {
	return b.Shell("input", "keyevent", strconv.Itoa(int(key))).Run(ctx)
}

// SendEvent simulates low-level user-input to the device.
func (b *binding) SendEvent(ctx context.Context, deviceId, eventType, eventCode, value int) error {
	args := fmt.Sprintf("/dev/input/event%v %v %v %v", deviceId, eventType, eventCode, value)
	return b.Shell("sendevent", args).Run(ctx)
}

// SendTouch simulates touch-screen press or release.
func (b *binding) SendTouch(ctx context.Context, deviceId, x, y int, pressed bool) {
	b.SendEvent(ctx, deviceId, EV_ABS, ABS_MT_TRACKING_ID, 0)
	b.SendEvent(ctx, deviceId, EV_ABS, ABS_MT_POSITION_X, x)
	b.SendEvent(ctx, deviceId, EV_ABS, ABS_MT_POSITION_Y, y)
	b.SendEvent(ctx, deviceId, EV_ABS, ABS_MT_PRESSURE, 50)
	b.SendEvent(ctx, deviceId, EV_ABS, ABS_MT_TOUCH_MAJOR, 5)
	b.SendEvent(ctx, deviceId, EV_SYN, SYN_REPORT, 0)
	if !pressed {
		b.SendEvent(ctx, deviceId, EV_ABS, ABS_MT_TRACKING_ID, -1)
		b.SendEvent(ctx, deviceId, EV_SYN, SYN_REPORT, 0)
	}
}

func atoi(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return v
}

// GetTouchDimensions returns the resolution of the touch sensor.
// This may be different to the dimensions of the LCD screen.
func (b *binding) GetTouchDimensions(ctx context.Context) (deviceId, minX, maxX, minY, maxY int, ok bool) {
	for i := 0; i < 10; i++ {
		device := "/dev/input/event" + strconv.Itoa(i)
		if info, err := b.Shell("getevent", "-lp", device).Call(ctx); err == nil {
			reX := regexp.MustCompile("ABS_MT_POSITION_X.* min ([0-9]+), max ([0-9]+),")
			reY := regexp.MustCompile("ABS_MT_POSITION_Y.* min ([0-9]+), max ([0-9]+),")
			matchX := reX.FindStringSubmatch(info)
			matchY := reY.FindStringSubmatch(info)
			if matchX != nil && matchY != nil {
				return i, atoi(matchX[1]), atoi(matchX[2]), atoi(matchY[1]), atoi(matchY[2]), true
			}
		}
	}
	return 0, 0, 0, 0, 0, false
}

// GetScreenDimensions returns the resolution of the display.
func (b *binding) GetScreenDimensions(ctx context.Context) (orientation, width, height int, ok bool) {
	if info, err := b.Shell("dumpsys", "display").Call(ctx); err == nil {
		re := regexp.MustCompile("mDefaultViewport.*orientation=([0-9]+).*deviceWidth=([0-9]+).*deviceHeight=([0-9]+)")
		if match := re.FindStringSubmatch(info); match != nil {
			return atoi(match[1]), atoi(match[2]), atoi(match[3]), true
		}
	}
	return 0, 0, 0, false
}
