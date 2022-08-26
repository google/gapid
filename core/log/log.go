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
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/google/gapid/core/text"
)

// Logger provides a logging interface.
type Logger struct {
	handler     Handler
	filter      Filter
	stacktracer Stacktracer
	clock       Clock
	tag         string
	process     string
	trace       []string
	values      *values
}

// SetFilter sets the filter for the logger
func (l *Logger) SetFilter(f Filter) *Logger {
	l.filter = f
	return l
}

// From returns a new Logger from the context ctx.
func From(ctx context.Context) *Logger {
	return &Logger{
		GetHandler(ctx),
		GetFilter(ctx),
		GetStacktracer(ctx),
		GetClock(ctx),
		GetTag(ctx),
		GetProcess(ctx),
		GetTrace(ctx),
		getValues(ctx),
	}
}

// Bind returns a new Logger from the context ctx with the additional values in
// v.
func Bind(ctx context.Context, v V) *Logger {
	return From(v.Bind(ctx))
}

// D logs a debug message to the logging target.
func D(ctx context.Context, fmt string, args ...interface{}) { From(ctx).D(fmt, args...) }

// I logs a info message to the logging target.
func I(ctx context.Context, fmt string, args ...interface{}) { From(ctx).I(fmt, args...) }

// W logs a warning message to the logging target.
func W(ctx context.Context, fmt string, args ...interface{}) { From(ctx).W(fmt, args...) }

// E logs a error message to the logging target.
func E(ctx context.Context, fmt string, args ...interface{}) { From(ctx).E(fmt, args...) }

// F logs a fatal message to the logging target.
// If stopProcess is true then the message indicates the process should stop.
func F(ctx context.Context, stopProcess bool, fmt string, args ...interface{}) {
	From(ctx).F(fmt, stopProcess, args...)
}

// D logs a debug message to the logging target.
func (l *Logger) D(fmt string, args ...interface{}) { l.Logf(Debug, false, fmt, args...) }

// I logs a info message to the logging target.
func (l *Logger) I(fmt string, args ...interface{}) { l.Logf(Info, false, fmt, args...) }

// W logs a warning message to the logging target.
func (l *Logger) W(fmt string, args ...interface{}) { l.Logf(Warning, false, fmt, args...) }

// E logs a error message to the logging target.
func (l *Logger) E(fmt string, args ...interface{}) { l.Logf(Error, false, fmt, args...) }

// F logs a fatal message to the logging target.
// If stopProcess is true then the message indicates the process should stop.
func (l *Logger) F(fmt string, stopProcess bool, args ...interface{}) {
	l.Logf(Fatal, stopProcess, fmt, args...)
}

// Logf logs a printf-style message at severity s to the logging target.
func (l *Logger) Logf(s Severity, stopProcess bool, fmt string, args ...interface{}) {
	h := l.handler
	if h == nil {
		return
	}

	if l.filter != nil {
		if !l.filter.ShowSeverity(s) {
			return
		}
	}

	h.Handle(l.Messagef(s, stopProcess, fmt, args...))
}

// Log logs a message at severity s to the logging target.
func (l *Logger) Log(s Severity, stopProcess bool, f string) {
	h := l.handler
	if h == nil {
		return
	}

	if l.filter != nil {
		if !l.filter.ShowSeverity(s) {
			return
		}
	}

	h.Handle(l.Message(s, stopProcess, f))
}

// Messagef returns a new Message with the given severity and text.
func (l *Logger) Messagef(s Severity, stopProcess bool, text string, args ...interface{}) *Message {
	return l.Message(s, stopProcess, fmt.Sprintf(text, args...))
}

// Message returns a new Message with the given severity and text.
func (l *Logger) Message(s Severity, stopProcess bool, text string) *Message {
	var t time.Time
	if c := l.clock; c != nil {
		t = c.Time()
	} else {
		t = time.Now()
	}

	if s := l.stacktracer; s != nil {
		// TODO
	}

	m := &Message{
		Text:        text,
		Time:        t.In(time.Local),
		Severity:    s,
		StopProcess: stopProcess,
		Tag:         l.tag,
		Process:     l.process,
		// Callstack: callstack(), // TODO: Callstack
		Trace: l.trace,
	}

	for n := l.values; n != nil; n = n.parent {
		for name, value := range n.v {
			m.Values = append(m.Values, &Value{Name: name, Value: value})
		}
	}

	sort.Sort(m.Values)

	return m
}

// Writer returns an io.WriteCloser that writes lines to to the logger at the
// specified severity.
func (l *Logger) Writer(s Severity) io.WriteCloser {
	return text.Writer(func(line string) error {
		l.Log(s, false, line)
		return nil
	})
}

// Fatal implements the standard go logger interface.
func (l *Logger) Fatal(args ...interface{}) { l.F("%s", true, fmt.Sprint(args...)) }

// Fatalf implements the standard go logger interface.
func (l *Logger) Fatalf(format string, args ...interface{}) { l.F(format, true, args...) }

// Fatalln implements the standard go logger interface.
func (l *Logger) Fatalln(args ...interface{}) { l.Fatal(args...) }

// Print implements the standard go logger interface.
func (l *Logger) Print(args ...interface{}) { l.I("%s", fmt.Sprint(args...)) }

// Printf implements the standard go logger interface.
func (l *Logger) Printf(format string, args ...interface{}) { l.I(format, args...) }

// Println implements the standard go logger interface.
func (l *Logger) Println(args ...interface{}) { l.Print(args...) }
