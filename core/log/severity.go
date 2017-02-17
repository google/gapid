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

import "github.com/google/gapid/core/fault/severity"

// GetSeverity gets the severity stored in the given context.
func GetSeverity(ctx Context) severity.Level {
	return severity.FromContext(ctx.Unwrap())
}

// Severity returns a context with the given Severity set on it.
func (ctx logContext) Severity(level severity.Level) Context {
	return Wrap(severity.NewContext(ctx.Unwrap(), level))
}

// Emergency is shorthand for ctx.At(EmergencyLevel)
func (ctx logContext) Emergency() Logger {
	return ctx.At(severity.Emergency)
}

// Alert is is shorthand for ctx.At(AlertLevel)
func (ctx logContext) Alert() Logger {
	return ctx.At(severity.Alert)
}

// Critical is shorthand for ctx.At(CriticalLevel)
func (ctx logContext) Critical() Logger {
	return ctx.At(severity.Critical)
}

// Error is shorthand for ctx.At(ErrorLevel).Cause(err)
func (ctx logContext) Error() Logger {
	return ctx.At(severity.Error)
}

// Warning is shorthand for ctx.At(WarningLevel)
func (ctx logContext) Warning() Logger {
	return ctx.At(severity.Warning)
}

// Notice is shorthand for ctx.At(NoticeLevel)
func (ctx logContext) Notice() Logger {
	return ctx.At(severity.Notice)
}

// Info is shorthand for ctx.At(InfoLevel)
func (ctx logContext) Info() Logger {
	return ctx.At(severity.Info)
}

// Debug is shorthand for ctx.At(DebugLevel)
func (ctx logContext) Debug() Logger {
	return ctx.At(severity.Debug)
}

// DebugN is shorthand for ctx.At(DebugLevel + n)
func (ctx logContext) DebugN(n int) Logger {
	return ctx.At(severity.Level(int(severity.Debug) + n))
}
