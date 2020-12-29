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

//go:build !analytics

package analytics

// This file contains stub implementations of the analytics package (used when
// the analytics build tag is omitted).

import (
	"context"
	"time"
)

// Payload is the interface implemented by Google Analytic payload types.
type Payload interface{}

// Send queues the Payload p to be sent to the Google Analytics server if
// analytics is enabled with a prior call to Enable().
func Send(p Payload) {}

// Flush flushes any pending Payloads to be sent to the Google Analytics server.
func Flush() {}

// Enable turns on Google Analytics tracking functionality using the given
// clientID and version.
func Enable(ctx context.Context, clientID string, version AppVersion) {}

// Disable turns off Google Analytics tracking.
func Disable() {}

// Stop is the function returned by SendTiming to end the timed region and send
// the result to the Google Analytics server (if enabled).
type Stop func(extras ...Payload)

// SendTiming is a no-op when analytics are not enabled.
func SendTiming(category, name string) Stop { return func(...Payload) {} }

// SendEvent sends the specific event to the Google Analytics server (if
// enabled).
func SendEvent(category, action, label string, extras ...Payload) {}

// SendException sends the exception to the Google Analytics server (if
// enabled).
func SendException(description string, fatal bool, extras ...Payload) {}

// SendBug sends an event indicating that a known bug has been hit.
func SendBug(id int, extras ...Payload) {}

// TargetDevice returns a payload value that describes a target device.
func TargetDevice(interface{}) Payload { return nil }

// AppVersion holds information about the currently running application and its
// version.
type AppVersion struct {
	Name, Build         string
	Major, Minor, Point int
}

// Size is a payload value for size values, usually representing bytes.
// Examples: File size.
type Size int

// Count is a payload value for count values, usually number of items.
// Examples: Number of commands in a capture.
type Count int

// SessionStart is a payload value to indicate the start of a session.
type SessionStart struct{}

// SessionEnd is a payload value to indicate the end of a session.
type SessionEnd struct{}

// Exception is a payload value that represents an application exception or
// crash.
type Exception struct {
	Description string
	Fatal       bool
}

// Event is a payload value that represents a single event.
type Event struct {
	Category string
	Action   string
	Label    string
	Value    int
}

// Timing is a payload value that represents a timing.
type Timing struct {
	Category string
	Name     string
	Label    string
	Duration time.Duration
}
