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

package client

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/pkg/errors"
)

// Flags is a bit-field of flags to use when creating a capture.
type Flags uint32

const (
	// DisablePrecompiledShaders fakes no support for PCS, forcing the app to
	// share shader source.
	DisablePrecompiledShaders Flags = 0x00000001
	// RecordErrorState queries the driver error state after each all and stores
	// errors as extras.
	RecordErrorState Flags = 0x10000000
	// DeferStart does not start tracing right away but waits for a signal
	// from gapit
	DeferStart Flags = 0x00000010
)

// Options to use when creating a capture.
type Options struct {
	// If non-zero, then a framebuffer-observation will be made after every n end-of-frames.
	ObserveFrameFrequency uint32
	// If non-zero, then a framebuffer-observation will be made after every n draw calls.
	ObserveDrawFrequency uint32
	// If non-zero, then the capture will only start at frame n.
	StartFrame uint32
	// If non-zero, then only n frames will be captured.
	FramesToCapture uint32
	// A bitmask of the APIs to capture in a trace.
	APIs uint32
	// Combination of FlagXX bits.
	Flags Flags
	// APK is an apk to install before tracing
	APK file.Path
}

const sizeGap = 1024 * 1024 * 5
const timeGap = time.Second
const startMidExecutionCapture = 0xdeadbeef

type siSize int64

var formats = []string{
	"%.0fB",
	"%.2fKB",
	"%.2fMB",
	"%.2fGB",
	"%.2fTB",
	"%.2fPB",
	"%.2fEB",
}

func (s siSize) String() string {
	if s == 0 {
		return "0.0B"
	}
	size := float64(s)
	e := math.Floor(math.Log(size) / math.Log(1000))
	f := formats[int(e)]
	v := math.Floor(size/math.Pow(1000, e)*10+0.5) / 10
	return fmt.Sprintf(f, v)
}

func capture(ctx context.Context, port int, s task.Signal, w io.Writer, o Options) (int64, error) {
	if task.Stopped(ctx) {
		return 0, nil
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return 0, nil // Treat failure-to-connect as target-not-ready instead of an error.
	}
	defer conn.Close()
	if err := sendHeader(conn, o); err != nil {
		return 0, log.Err(ctx, err, "Header send failed")
	}

	var count, nextSize siSize
	startTime := time.Now()
	nextTime := startTime
	started := false
	for {
		if task.Stopped(ctx) {
			log.I(ctx, "Stop: %v", count)
			break
		}
		if (o.Flags & DeferStart) != 0 {
			if !started && s.Fired() {
				started = true
				w := endian.Writer(conn, device.LittleEndian)
				w.Uint32(startMidExecutionCapture)
			}
		}
		now := time.Now()
		conn.SetReadDeadline(now.Add(time.Millisecond * 100)) // Allow for stop event and UI refreshes.
		n, err := io.CopyN(w, conn, 1024*64)
		count += siSize(n)
		switch {
		case errors.Cause(err) == io.EOF:
			// End of stream. End.
			log.I(ctx, "EOF: %v", count)
			return int64(count), nil
		case err != nil && count > 0:
			err, isnet := err.(net.Error)
			if !isnet || (!err.Temporary() && !err.Timeout()) {
				log.I(ctx, "Connection error: %v", err)
				// Got an error mid-stream terminate.
				return int64(count), err
			}
		case err != nil && count == 0:
			// Got an error without receiving a byte of data.
			// Treat failure-to-connect as target-not-ready instead of an error.
			return 0, nil
		}
		if count > nextSize || now.After(nextTime) {
			nextSize = count + sizeGap
			nextTime = now.Add(timeGap)
			delta := time.Duration(int64(now.Sub(startTime)/time.Millisecond)) * time.Millisecond
			log.I(ctx, "Capturing: %v in %v", count, delta)
		}
	}
	return int64(count), nil
}

// Capture opens up the specified port and then waits for a capture to be
// delivered using the specified capture options.
// It copies the capture into the supplied writer.
func Capture(ctx context.Context, port int, s task.Signal, w io.Writer, options Options) (int64, error) {
	log.I(ctx, "Waiting for connection to localhost:%d...", port)
	for {
		count, err := capture(ctx, port, s, w, options)
		if err != nil {
			return count, err
		}
		if count != 0 {
			return count, nil
		}
		// ADB has an annoying tendancy to insta-close forwarded sockets when
		// there's no application waiting for the connection. Treat this as
		// another waiting-for-connection case.
		select {
		case <-task.ShouldStop(ctx):
			log.I(ctx, "Aborted.")
			return 0, nil
		case <-time.After(500 * time.Millisecond):
			log.I(ctx, "Retry...")
		}
	}
}
