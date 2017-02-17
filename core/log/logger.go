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
	"io"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text"
)

// Logger contains the tracking information while generating a filtered logging message.
// They are immutable, passed by value and allocated on the stack.
// You construct one with the At function, chain WithValue calls to build the context, and finish with a Log call.
// You can build a partially complete builder and use it to log many times with similar messages, this is very useful
// if you want logging statements in a loop where even the minimal cost of initializing the Builder is too much.
type Logger struct {
	J jot.Jotter
}

// At constructs a new Logger from the supplied context at the specified severity level.
// It applies the prefilter, and returns an inactive logger if the severity level is not active.
func (ctx logContext) At(level severity.Level) Logger {
	ctx.internal = severity.NewContext(ctx.Unwrap(), level)
	return Logger{J: jot.At(ctx.internal, level)}
}

// Active returns true if the logger is not suppressing log messages.
func (l Logger) Active() bool {
	return l.J.Context != nil
}

// updates the logger in place, only use when you have already copied the logger and checked it is active
func (l *Logger) update(key string, value interface{}) {
	l.J = l.J.With(key, value)
}

func (l Logger) Writer() io.WriteCloser {
	return text.Writer(func(s string) error {
		l.Log(s)
		return nil
	})
}
