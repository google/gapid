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

//go:build analytics

package analytics

import (
	"time"

	"github.com/google/gapid/core/app/analytics/param"
	"github.com/google/gapid/core/os/device"
)

// Payload is the interface implemented by Google Analytic payload types.
type Payload interface {
	values(add func(param.Parameter, interface{}))
}

// payloads is a list of payloads.
type payloads []Payload

func (c payloads) values(add func(param.Parameter, interface{})) {
	for _, p := range c {
		p.values(add)
	}
}

// Size is a payload value for size values, usually representing bytes.
// Examples: File size.
type Size int

func (s Size) values(add func(param.Parameter, interface{})) {
	add(param.Size, int(s))
}

// Count is a payload value for count values, usually number of items.
// Examples: Number of commands in a capture.
type Count int

func (s Count) values(add func(param.Parameter, interface{})) {
	add(param.Count, int(s))
}

// SessionStart is a payload value to indicate the start of a session.
type SessionStart struct{}

func (SessionStart) values(add func(param.Parameter, interface{})) {
	add(param.SessionControl, "start")
}

// SessionEnd is a payload value to indicate the end of a session.
type SessionEnd struct{}

func (SessionEnd) values(add func(param.Parameter, interface{})) {
	add(param.SessionControl, "end")
}

// Exception is a payload value that represents an application exception or
// crash.
type Exception struct {
	Description string
	Fatal       bool
}

func (m Exception) values(add func(param.Parameter, interface{})) {
	add(param.HitType, "exception")
	add(param.ExceptionDescription, m.Description)
	add(param.ExceptionFatal, m.Fatal)
}

// Event is a payload value that represents a single event.
type Event struct {
	Category string
	Action   string
	Label    string
	Value    int
}

func (m Event) values(add func(param.Parameter, interface{})) {
	add(param.HitType, "event")
	add(param.EventCategory, m.Category)
	add(param.EventAction, m.Action)
	if m.Label != "" {
		add(param.EventLabel, m.Label)
	}
	if m.Value != 0 {
		add(param.EventValue, m.Value)
	}
}

// Timing is a payload value that represents a timing.
type Timing struct {
	Category string
	Name     string
	Label    string
	Duration time.Duration
}

func (m Timing) values(add func(param.Parameter, interface{})) {
	add(param.HitType, "timing")
	add(param.TimingCategory, m.Category)
	add(param.TimingName, m.Name)
	add(param.TimingDuration, m.Duration.Nanoseconds()/1e6)
	if m.Label != "" {
		add(param.TimingLabel, m.Label)
	}
}

// TargetDevice returns a payload value that describes a target device.
func TargetDevice(d *device.Configuration) Payload {
	return targetDevice{d}
}

type targetDevice struct{ *device.Configuration }

func (m targetDevice) values(add func(param.Parameter, interface{})) {
	if m.Configuration != nil {
		os, gpu := getOSAndGPU(m.Configuration)
		add(param.TargetOS, os)
		add(param.TargetGPU, gpu)
	}
}
