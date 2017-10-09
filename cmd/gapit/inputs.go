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

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
)

// The input file is essentially a sparse table of integer values.
type inputProperty string
type inputEvent map[inputProperty]int
type inputEvents []inputEvent

var inputProperties []inputProperty

func registerInputPropery(prop string) inputProperty {
	inputProperties = append(inputProperties, inputProperty(prop))
	return inputProperty(prop)
}

var (
	// Display dimensions
	kOrientation = registerInputPropery("orientation")
	kWidth       = registerInputPropery("width")
	kHeight      = registerInputPropery("height")
	// Touch-screen dimensions
	kMinX = registerInputPropery("minX")
	kMaxX = registerInputPropery("maxX")
	kMinY = registerInputPropery("minY")
	kMaxY = registerInputPropery("maxY")
	// Frame statistics
	kTime          = registerInputPropery("time")
	kFrame         = registerInputPropery("frame")
	kDrawsPerFrame = registerInputPropery("drawsPerFrame")
	// Screen tap/swipe
	kX       = registerInputPropery("x")
	kY       = registerInputPropery("y")
	kPressed = registerInputPropery("pressed")
	// End of recording
	kEnd = registerInputPropery("end")
)

func writeEvent(out io.Writer, event inputEvent) {
	var line []string
	for _, name := range inputProperties {
		if value, ok := event[name]; ok {
			line = append(line, fmt.Sprintf("%s:%v", name, value))
		}
	}
	fmt.Fprintf(out, "%s\n", strings.Join(line, " "))
}

func parseEvent(line string) (inputEvent, error) {
	input := inputEvent{}
	for _, kvp := range strings.Split(line, " ") {
		if kvp != "" {
			parts := strings.Split(kvp, ":")
			if len(parts) == 2 {
				name := inputProperty(parts[0])
				value, err := strconv.Atoi(parts[1])
				if err != nil {
					return nil, err
				}
				input[name] = value
			} else {
				return nil, fmt.Errorf("Failed to parse key-value pair: '%s'", kvp)
			}
		}
	}
	return input, nil
}

// Implementation of the Writer interface which forwards the text to a lambda.
type lambdaWriter struct {
	f func(s string)
}

func (w *lambdaWriter) Write(p []byte) (n int, err error) {
	w.f(string(p))
	return len(p), nil
}

func atoi(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return v
}

type frameInfo struct{ frame, drawsPerFrame int }

// Monitor logcat and parse frame statistics.
func monitorFrameStatistics(ctx context.Context, d adb.Device, out chan frameInfo) {
	re := regexp.MustCompile("NumFrames:([0-9]+).*NumDrawsPerFrame:([0-9]+)")
	stdout := &lambdaWriter{f: func(s string) {
		for _, match := range re.FindAllStringSubmatch(s, -1) {
			out <- frameInfo{frame: atoi(match[1]), drawsPerFrame: atoi(match[2])}
		}
	}}
	d.Command("logcat", "-T", "1", "-s", "GAPID:I").Capture(stdout, nil).Run(ctx)
	close(out)
}

type currentFrameInfo struct {
	value frameInfo
	mutex sync.Mutex
}

// Observe channel and make copy of the most recent value.
// We need to make copy of the channel to observe it.
func (info *currentFrameInfo) update(in <-chan frameInfo, out chan frameInfo) {
	for v := range in {
		info.mutex.Lock()
		info.value = v
		info.mutex.Unlock()
		if out != nil {
			out <- v
		}
	}
	if out != nil {
		close(out)
	}
}

func (info *currentFrameInfo) get() frameInfo {
	info.mutex.Lock()
	v := info.value
	info.mutex.Unlock()
	return v
}

// Write frame statistics into a file (rate limited).
func recordFrameStatistics(out io.Writer, in <-chan frameInfo) {
	const rate_limit = 10
	const min_change = 0.2
	startTime := time.Now()
	nextFrame, lastDrawsPerFrame := 0, 0
	for info := range in {
		// Emit statistics only if the number of draws changed significantly
		// and if it has been at least couple of frame since last time.
		change := float64(info.drawsPerFrame+1)/float64(lastDrawsPerFrame+1) - 1.0
		if info.frame >= nextFrame && math.Abs(change) >= min_change {
			t := int(time.Now().Sub(startTime).Seconds() * 1000)
			writeEvent(out, inputEvent{kTime: t, kFrame: info.frame, kDrawsPerFrame: info.drawsPerFrame})
			nextFrame = info.frame + rate_limit
			lastDrawsPerFrame = info.drawsPerFrame
		}
	}
}

type touchInfo struct{ x, y, pressed int }

// Monitor and parse touch screen events.
func monitorTouchScreen(ctx context.Context, d adb.Device, out chan touchInfo) {
	x, y, pressed := 0, 0, 0
	stdout := &lambdaWriter{f: func(s string) {
		for _, line := range strings.Split(s, "\n") {
			var device_id, value int
			var event_type, event_code string
			if _, err := fmt.Sscanf(line, "/dev/input/event%d: %s %s %x",
				&device_id, &event_type, &event_code, &value); err == nil {
				switch {
				case event_type == "EV_ABS" && event_code == "ABS_MT_POSITION_X":
					x, pressed = value, 1
				case event_type == "EV_ABS" && event_code == "ABS_MT_POSITION_Y":
					y, pressed = value, 1
				case event_type == "EV_SYN" && event_code == "SYN_REPORT":
					out <- touchInfo{x: x, y: y, pressed: pressed}
					pressed = 0
				}
			}
		}
	}}
	d.Shell("getevent", "-l").Capture(stdout, nil).Run(ctx)
	close(out)
}

// Write touch-screen events to file (rate limited).
// The touch event will also include the most recent frame information.
func recordTouchInfo(out io.Writer, currentInfo *currentFrameInfo, in <-chan touchInfo) {
	const rate_limit = 10
	startTime := time.Now()
	wasPressed, nextPressedFrame := 0, 0
	for info := range in {
		frameInfo := currentInfo.get()
		if wasPressed != info.pressed || (info.pressed == 1 && frameInfo.frame >= nextPressedFrame) {
			wasPressed = info.pressed
			t := int(time.Now().Sub(startTime).Seconds() * 1000)
			writeEvent(out, inputEvent{kTime: t,
				kFrame: frameInfo.frame, kDrawsPerFrame: frameInfo.drawsPerFrame,
				kX: info.x, kY: info.y, kPressed: info.pressed})
			nextPressedFrame = frameInfo.frame + rate_limit
		}
	}
}

func startRecordingInputs(ctx context.Context, d adb.Device, filename string) (cleanup func(), err error) {
	out, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(out, "# %s\n", strings.Join(os.Args, " "))

	// Record screen dimensions
	if orientation, width, height, ok := d.GetScreenDimensions(ctx); ok {
		writeEvent(out, inputEvent{kOrientation: orientation, kWidth: width, kHeight: height})
	}

	// Record touch dimensions
	if _, minX, maxX, minY, maxY, ok := d.GetTouchDimensions(ctx); ok {
		writeEvent(out, inputEvent{kMinX: minX, kMaxX: maxX, kMinY: minY, kMaxY: maxY})
	}

	touchInfos := make(chan touchInfo, 256)
	frameInfos := make(chan frameInfo, 256)
	frameInfosCopy := make(chan frameInfo, 256)
	stats := &currentFrameInfo{}

	crash.Go(func() { monitorTouchScreen(ctx, d, touchInfos) })
	crash.Go(func() { monitorFrameStatistics(ctx, d, frameInfos) })
	crash.Go(func() { stats.update(frameInfos, frameInfosCopy) })
	crash.Go(func() { recordTouchInfo(out, stats, touchInfos) })
	crash.Go(func() { recordFrameStatistics(out, frameInfosCopy) })

	startTime := time.Now()
	return func() {
		t := int(time.Now().Sub(startTime).Seconds() * 1000)
		writeEvent(out, inputEvent{kTime: t, kEnd: 1})
	}, nil
}

func loadReplayInputs(filename string) (inputs inputEvents, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "#") {
			input, err := parseEvent(line)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, input)
		}
	}
	return
}

// Load given file and start replaying the user inputs.
// 'eof' will be signalled when the end of file is reached.
func startReplayingInputs(ctx context.Context, d adb.Device, replayInputsIn string, stop task.CancelFunc) error {
	inputs, err := loadReplayInputs(replayInputsIn)
	if err != nil {
		return err
	}
	deviceId, _, maxX, _, maxY, ok := d.GetTouchDimensions(ctx)
	if !ok {
		return fmt.Errorf("Faild to get touchscreen dimensions")
	}
	stats := currentFrameInfo{}
	frameInfos := make(chan frameInfo, 256)
	crash.Go(func() { monitorFrameStatistics(ctx, d, frameInfos) })
	crash.Go(func() { stats.update(frameInfos, nil) })
	crash.Go(func() {
		ctx := log.Enter(ctx, "Inputs")
		startTime := time.Now()
		time_drift := time.Duration(0)
		value_of := inputEvent{} // Keep track of most recent state.
		for _, input := range inputs {
			for k, v := range input {
				value_of[k] = v
			}
			if target, ok := input[kTime]; ok {
				// Wait for the minimum time
				current := time.Now().Sub(startTime)
				target := time.Duration(target)*time.Millisecond + time_drift
				if current < target {
					log.I(ctx, "Wait until %.1fs", target.Seconds())
					time.Sleep(target - current)
				} else {
					time_drift = time_drift + current - target
					log.I(ctx, "Time drift: %.1fs", time_drift.Seconds())
				}
			}
			if target, ok := input[kFrame]; ok {
				// Wait for the minimum frame
				for stats.get().frame < target {
					time_drift = time_drift + time.Second
					log.I(ctx, "Wait for more frames (%v seen, %v needed, %.1fs drift)",
						stats.get().frame, target, time_drift.Seconds())
					time.Sleep(time.Second)
				}
			}
			if target, ok := input[kDrawsPerFrame]; ok {
				// Wait for the minimum draw count
				target = target - target/10 - 1
				for stats.get().drawsPerFrame < target {
					time_drift = time_drift + time.Second
					log.I(ctx, "Wait for more draws (%v seen, %v needed, %.1fs drift)",
						stats.get().drawsPerFrame, target, time_drift.Seconds())
					time.Sleep(time.Second)
					target = target - target/10 - 1 // relax over time to ensure progress
				}
			}
			if pressed, ok := input[kPressed]; ok {
				// Press/release screen
				x, y := input[kX]*maxX/value_of[kMaxX], input[kY]*maxY/value_of[kMaxY]
				log.I(ctx, "Touch: x:%v y:%v pressed:%v", x, y, pressed)
				d.SendTouch(ctx, deviceId, x, y, pressed != 0)
			}
			if end, ok := input[kEnd]; ok && end == 1 {
				stop()
				return
			}
		}
	})
	return nil
}
