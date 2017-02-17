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
	"time"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

// Context is a wrapper to make context.Context fluent.
// you can get the underlying context if you need it, and build a wrapper using log.In.
// Because Context is a pure wrapper, it interacts cleanly with any library that uses context.Context directly.
type Context interface {
	// conform to the context.Context interface to make migration easier
	context.Context

	// Unwrap a context.Context from a Context.
	Unwrap() context.Context

	// At constructs a new Logger from the context at the specified severity level.
	// It applies the prefilter, and returns an inactive logger if the severity level is not active.
	At(level severity.Level) Logger
	// Raw creates a raw mode logger for the specified channel.
	Raw(channel string) Logger
	// Emergency is shorthand for ctx.At(EmergencyLevel)
	Emergency() Logger
	// Alert is is shorthand for ctx.At(AlertLevel)
	Alert() Logger
	// Critical is shorthand for ctx.At(CriticalLevel)
	Critical() Logger
	// Error is shorthand for ctx.At(ErrorLevel).Cause(err)
	Error() Logger
	// Warning is shorthand for ctx.At(WarningLevel)
	Warning() Logger
	// Notice is shorthand for ctx.At(NoticeLevel)
	Notice() Logger
	// Info is shorthand for ctx.At(InfoLevel)
	Info() Logger
	// Debug is shorthand for ctx.At(DebugLevel)
	Debug() Logger

	// WithValue returns a new context with the additional key value pair specified.
	// If the context already has the specified value, it will be overwritten.
	WithValue(key string, value interface{}) Context
	// Severity returns a context with the given Severity set on it.
	Severity(level severity.Level) Context
	// Tag returns a context with the tag set.
	Tag(tag string) Context
	// Enter returns a context with an entry added to the trace chain.
	Enter(name string) Context
	// Filter returns a context with the given Filter set on it.
	Filter(f Filter) Context
	// PreFilter returns a context with the given PreFilter set on it.
	PreFilter(f PreFilter) Context
	// Handler returns a new Builder with the given Handler set on it.
	Handler(h note.Handler) Context
	// StackFilter returns a context with the given StackFilter set on it.
	//StackFilter(f StackFilter) Context
	// V is shorthand for ctx.WithValue(key, value)
	V(key string, value interface{}) Context
	// S is shorthand for ctx.WithValue(key, value), but is only for string.
	S(key string, value string) Context
	// I is shorthand for ctx.WithValue(key, value) but is only for int.
	I(key string, value int) Context
	// F is shorthand for ctx.WithValue(key, value) but is only for float64.
	F(key string, value float64) Context
	// T is shorthand for ctx.WithValue(key, reflect.TypeOf(value)).
	T(key string, value interface{}) Context

	// Record returns a log record from the current builder context with the supplied message.
	Record(msg interface{}) note.Page

	// Print is shorthand for ctx.At(InfoLevel).Log(msg)
	// Useful for hidden by default simple progress messages
	Print(msg string)
	// Print is shorthand for ctx.At(InfoLevel).Logf(format, args...)
	// This is useful for Printf debugging, should generally not be left in the code.
	Printf(format string, args ...interface{})
	// Fatal is shorthand for ctx.At(EmergencyLevel).Log(msg)
	// Useful in applications where emergency level logging also causes a panic.
	Fatal(msg string)
	// Fatalf is shorthand for ctx.At(EmergencyLevel).Logf(format, args...)
	// Useful in applications where emergency level logging also causes a panic.
	Fatalf(format string, args ...interface{})
}

type logContext struct {
	internal context.Context
}

// Wrap a supplied context.Context as a fluent Context.
func Wrap(ctx context.Context) Context {
	return logContext{internal: ctx}
}

// Unwrap a context.Context from a Context.
func (ctx logContext) Unwrap() context.Context {
	return ctx.internal
}

// Background returns a context.Background as a log.Context
// This is the starting point for the root of a context tree.
func Background() Context {
	return Wrap(context.Background())
}

// TODO returns a context.TODO as a log.Context
// This is normally used to get a context in a place where one is not available as a short term measure until one is
// correctly wired through the intervening calls.
func TODO() Context {
	return Wrap(context.TODO())
}

func (ctx logContext) Deadline() (deadline time.Time, ok bool) { return ctx.internal.Deadline() }
func (ctx logContext) Done() <-chan struct{}                   { return ctx.internal.Done() }
func (ctx logContext) Err() error                              { return ctx.internal.Err() }
func (ctx logContext) Record(msg interface{}) note.Page        { return jot.To(ctx.Unwrap()).Page }
