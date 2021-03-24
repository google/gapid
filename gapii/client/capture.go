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
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/pkg/errors"
)

// Flags is a bit-field of flags to use when creating a capture.
type Flags uint32

const (

	// NOTE: flags must be kept in sync with gapii/cc/connection_header.h

	// DeferStart does not start tracing right away but waits for a signal
	// from gapit
	DeferStart Flags = 0x00000010
	// NoBuffer causes the trace to not buffer any data. This will allow
	// more data to be preserved if an application may crash.
	NoBuffer Flags = 0x00000020
	// HideUnknownExtensions will prevent any unknown extensions from being
	// seen by the application
	HideUnknownExtensions Flags = 0x00000040
	// StoreTimestamps requests that the capture contain timestamps
	StoreTimestamps Flags = 0x00000080
	// DisableCoherentMemoryTracker disables the coherent memory tracker from running.
	DisableCoherentMemoryTracker Flags = 0x000000100
	// WaitForDebugger makes gapii wait for a debugger to connect
	WaitForDebugger Flags = 0x000000200

	// VulkanAPI is hard-coded bit mask for Vulkan API, it needs to be kept in sync
	// with the api_index in the vulkan.api file.
	VulkanAPI = uint32(1 << 2)
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
	// Additional flags to pass to am start
	AdditionalFlags string
	// Enable ANGLE for application prior to start
	EnableAngle bool
	// The name of the pipe to connect/listen to.
	PipeName string
}

const sizeGap = 1024 * 1024 * 5
const timeGap = time.Second

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

func (p *Process) connect(ctx context.Context) error {
	log.I(ctx, "Waiting for connection to localhost:%d...", p.Port)

	// ADB has an annoying tendancy to insta-close forwarded sockets when
	// there's no application waiting for the connection. Treat errors as
	// another waiting-for-connection case.
	return task.Retry(ctx, 0, 500*time.Millisecond, func(ctx context.Context) (retry bool, err error) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", p.Port), 3*time.Second)
		if err != nil {
			return false, log.Err(ctx, err, "Dial failed")
		}
		r := bufio.NewReader(conn)
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		var magic [5]byte
		if _, err := io.ReadFull(r, magic[:]); err != nil {
			conn.Close()
			return false, log.Err(ctx, err, "Failed to read magic")
		}
		if magic != [...]byte{'g', 'a', 'p', 'i', 'i'} {
			conn.Close()
			return true, log.Errf(ctx, nil, "Got unexpected magic: %v", magic)
		}
		if err := sendHeader(conn, p.Options); err != nil {
			conn.Close()
			return true, log.Err(ctx, err, "Failed to send header")
		}
		p.conn = conn
		return true, nil
	})
}

func handleCommError(ctx context.Context, commErr error, anyDataReceived bool) (abort bool, err error) {
	switch {
	case errors.Cause(commErr) == io.EOF:
		log.E(ctx, "unexpected end of stream")
		abort = true
		// Most of the time, this error happens when the app crashed: rather
		// than reporting just "EOF", hint that this was probably a crash.
		err = log.Err(ctx, commErr, "The application exited during the capture")
	case commErr != nil && anyDataReceived:
		netErr, isnet := commErr.(net.Error)
		if !isnet || (!netErr.Temporary() && !netErr.Timeout()) {
			log.E(ctx, "Connection error: %v", commErr)
			// Got an error mid-stream terminate.
			abort = true
			err = commErr
		}
	case commErr != nil && !anyDataReceived:
		log.E(ctx, "Target not ready: %v", commErr)
		// Got an error without receiving a byte of data.
		// Treat failure-to-connect as target-not-ready instead of an error.
		abort = true
	}
	return
}

// Capture opens up the specified port and then waits for a capture to be
// delivered using the specified capture options o.
// It copies the capture into the supplied writer.
// If the process was started with the DeferStart flag, then tracing will wait
// until start is fired.
// Capturing will stop when the stop signal is fired (clean stop) or the
// context is cancelled (abort).
func (p *Process) Capture(ctx context.Context, start task.Signal, stop task.Signal, ready task.Task, w io.Writer, written *int64) (size int64, err error) {
	stopTiming := analytics.SendTiming("trace", "duration")
	defer func() {
		stopTiming(analytics.Size(size))
		var label string
		if (p.Options.Flags & DeferStart) != 0 {
			label = "mec"
		}
		if err != nil {
			analytics.SendEvent("trace", "failed", label, analytics.Size(size),
				analytics.TargetDevice(p.Device.Instance().GetConfiguration()))

		} else {
			analytics.SendEvent("trace", "succeeded", label, analytics.Size(size),
				analytics.TargetDevice(p.Device.Instance().GetConfiguration()))
		}
	}()

	if p.conn == nil {
		if err := p.connect(ctx); err != nil {
			return 0, err
		}
	}
	status.Event(ctx, status.ProcessScope, "Trace Connected")

	conn := p.conn
	defer conn.Close()

	writeErr := make(chan error)
	if (p.Options.Flags & DeferStart) != 0 {
		go func() {
			if start.Wait(ctx) {
				if err := writeStartTrace(conn); err != nil {
					writeErr <- err
				}
			}
		}()
	}
	go func() {
		if stop.Wait(ctx) {
			if err := writeEndTrace(conn); err == nil {
				time.Sleep(2 * time.Second)
				writeErr <- errors.New("Traced application is unresponsive.")
			} else {
				writeErr <- err
			}
		}
	}()

	var count siSize
	var lastErrorMsg string
mainLoop:
	for {
		select {
		case err := <-writeErr:
			log.E(ctx, "Write error: %v", err)
			return int64(count), err
		default:
		}
		if task.Stopped(ctx) {
			log.I(ctx, "Stop: %v", count)
			break mainLoop
		}
		msgType, dataSize, headerErr := readHeader(conn)
		if abort, err := handleCommError(ctx, headerErr, count > 0); abort {
			return int64(count), err
		}
		switch msgType {
		case messageData:
			read, dataErr := readData(ctx, conn, dataSize, w, written)
			count += read
			if dataErr != nil {
				return int64(count), dataErr
			}
		case messageEndTrace:
			log.D(ctx, "Received end trace message: %v", count)
			// if received error messages, return most recent
			if lastErrorMsg != "" {
				return int64(count), errors.New(lastErrorMsg)
			}
			break mainLoop
		case messageError:
			errorMsg, errorErr := readError(conn, dataSize)
			if errorMsg != "" {
				log.E(ctx, "Received error: %s", errorMsg)
				lastErrorMsg = errorMsg
			}
			if abort, err := handleCommError(ctx, errorErr, true); abort {
				return int64(count), err
			}
		}
	}
	return int64(count), nil
}
