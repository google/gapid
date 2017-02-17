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
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/fault/severity"
)

// Log emits a log record with the current context and supplied message to the active log handler.
func (l Logger) Log(msg string) {
	l.J.Print(msg)
}

// Logf emits a log record with the current context and supplied message to the active log handler.
func (l Logger) Logf(format string, args ...interface{}) {
	l.J.Printf(format, args...)
}

// Print is shorthand for ctx.At(InfoLevel).Log(msg)
// Useful for hidden by default simple progress messages
func (ctx logContext) Print(msg string) {
	jot.Info(ctx.Unwrap()).Print(msg)
}

// Print is shorthand for ctx.At(InfoLevel).Logf(format, args...)
// This is useful for Printf debugging, should generally not be left in the code.
func (ctx logContext) Printf(format string, args ...interface{}) {
	jot.Info(ctx.Unwrap()).Printf(format, args...)
}

// Fatal is shorthand for ctx.At(EmergencyLevel).Log(msg)
// Useful in applications where emergency level logging also causes a panic.
func (ctx logContext) Fatal(msg string) {
	jot.At(ctx.Unwrap(), severity.Critical).Print(msg)
}

// Fatalf is shorthand for ctx.At(EmergencyLevel).Logf(format, args...)
// Useful in applications where emergency level logging also causes a panic.
func (ctx logContext) Fatalf(format string, args ...interface{}) {
	jot.At(ctx.Unwrap(), severity.Critical).Printf(format, args...)
}

// Fatal is shorthand for ctx.At(EmergencyLevel).Cause(err).Log(msg)
// Useful in applications where emergency level logging also causes a panic.
func (ctx logContext) FatalError(err error, msg string) {
	jot.Fatal(ctx.Unwrap(), err, msg)
}
