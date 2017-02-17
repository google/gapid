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

package severity

import (
	"context"
	"strconv"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/text/note"
)

type (
	// Level defines the severity level of a logging message.
	// The levels match the ones defined in rfc5424 for syslog.
	Level int32
)

const (
	// Emergency indicates the system is unusable, no further data should be trusted.
	Emergency Level = 0
	// Alert indicates action must be taken immediately.
	Alert Level = 1
	// Critical indicates errors severe enough to terminate processing.
	Critical Level = 2
	// Error indicates non terminal failure conditions that may have an effect on results.
	Error Level = 3
	// Warning indicates issues that might affect performance or compatibility, but could be ignored.
	Warning Level = 4
	// Notice indicates normal but significant conditions.
	Notice Level = 5
	// Info indicates minor informational messages that should generally be ignored.
	Info Level = 6
	// Debug indicates verbose debug-level messages.
	Debug Level = 7
)

const (
	// LevelKey is the context key for the current severity level.
	LevelKey = levelKeyType("Level")
	// FilterKey is the context key for the severity filter control.
	FilterKey = filterKeyType("LevelFilter")
)

var (
	// Section is the note section description for the severity.
	Section = note.SectionInfo{Key: "Severity", Order: 0, Relevance: note.Important}
	// DefaultLevel is the default severity to use when one has not been set.
	DefaultLevel = Notice
	// DefaultFilter is the default severity to filter at.
	DefaultFilter = Notice
)

type (
	// levelKeyType.
	levelKeyType string

	// filterKeyType is a key type for a level filter.
	filterKeyType string
)

var (
	levelToName = map[Level]string{
		Emergency: "Emergency",
		Alert:     "Alert",
		Critical:  "Critical",
		Error:     "Error",
		Warning:   "Warning",
		Notice:    "Notice",
		Info:      "Info",
		Debug:     "Debug",
	}
)

// NewContext returns a new context with the severity set to level.
func NewContext(ctx context.Context, level Level) context.Context {
	return keys.WithValue(ctx, LevelKey, level)
}

// FromContext returns the current severity level of the context.
func FromContext(ctx context.Context) Level {
	v := ctx.Value(LevelKey)
	if v == nil {
		return DefaultLevel
	}
	return v.(Level)
}

// FindLevel picks the severity level out of the page, if it has one.
func FindLevel(p note.Page) Level {
	for _, section := range p {
		if section.Key == Section.Key {
			for _, item := range section.Content {
				if item.Key == LevelKey {
					return item.Value.(Level)
				}
			}
		}
	}
	return DefaultLevel
}

// GetFilter returns the current severity level filter of the context.
func GetFilter(ctx context.Context) Level {
	v := ctx.Value(FilterKey)
	if v == nil {
		return DefaultFilter
	}
	return v.(Level)
}

// Filter returns a new context with the severity level filter set.
// It will filter out notes at >level
func Filter(ctx context.Context, level Level) context.Context {
	return keys.WithValue(ctx, FilterKey, level)
}

// Choose allows *Level to be used as a command line flag.
func (l *Level) Choose(c interface{}) { *l = c.(Level) }

// String returns the name of the severity level.
func (l Level) String() string {
	if name, ok := levelToName[l]; ok {
		return name
	}
	return strconv.Itoa(int(l))
}

func (k levelKeyType) OmitKey() bool { return true }
func (k levelKeyType) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(Section, k, value)
}

func (filterKeyType) Transcribe(ctx context.Context, p *note.Page, value interface{}) {}
func (filterKeyType) Filter(ctx context.Context) bool {
	return Enabled(ctx, FromContext(ctx))
}

// Enabled tests if the specified level is currently enabled for logging in the given context.
func Enabled(ctx context.Context, level Level) bool {
	return level > GetFilter(ctx)
}
