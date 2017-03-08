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

package log

import (
	"context"

	"github.com/google/gapid/core/context/keys"
)

// Stacktracer is the interface that controls when a stacktrace is taken.
type Stacktracer interface {
	ShouldStacktrace(m *Message) bool
}

type stacktracerKeyTy string

const stacktracerKey stacktracerKeyTy = "log.tagStacktracer"

// PutStacktracer returns a new context with the Stacktracer assigned to w.
func PutStacktracer(ctx context.Context, s Stacktracer) context.Context {
	return keys.WithValue(ctx, stacktracerKey, s)
}

// GetStacktracer returns the Stacktracer assigned to ctx.
func GetStacktracer(ctx context.Context) Stacktracer {
	out, _ := ctx.Value(stacktracerKey).(Stacktracer)
	return out
}

// SeverityStacktracer implements the Stacktracer interface which adds a
// stacktrace to messages equal to or more than the severity level.
type SeverityStacktracer Severity

// ShouldStacktrace returns true if the message of severity s should include a
// stacktrace.
func (s SeverityStacktracer) ShouldStacktrace(m *Message) bool { return Severity(s) >= m.Severity }
