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

package jot

import (
	"context"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/fault/severity"
)

// To returns a jotter that uses the settings and values from a supplied context.
func To(ctx context.Context) Jotter {
	page, ok := memo.From(ctx)
	if !ok {
		return Jotter{}
	}
	return Jotter{Context: ctx, Page: page}
}

// At returns a jotter at the specified level for the given context.
// If the level is disabled, the jotter will also be disabled.
// It returns a jotter at the specified level for the given context.
func At(ctx context.Context, level severity.Level) Jotter {
	return To(severity.NewContext(ctx, level))
}

// With is To(ctx).With(name, value)
// It is used to get a jotter with an extra value attached.
func With(ctx context.Context, name interface{}, value interface{}) Jotter {
	return To(ctx).With(name, value)
}

// Jot is To(ctx).Jot(msg)
// It is used to build a simple note.
func Jot(ctx context.Context, msg string) Jotter {
	return To(ctx).Jot(msg)
}

// Jotf is To(ctx).Jotf(msg, args...)
// It is used to build a simple formatted note.
func Jotf(ctx context.Context, msg string, args ...interface{}) Jotter {
	return To(ctx).Jotf(msg, args...)
}

// Print is To(ctx).Print(msg)
// It is used to immediatly log a message.
func Print(ctx context.Context, msg string) {
	To(ctx).Print(msg)
}

// Printf is To(ctx).Printf(msg, args...)
// It is used to immediatly log a formatted message.
func Printf(ctx context.Context, msg string, args ...interface{}) {
	To(ctx).Printf(msg, args...)
}

// Fail forms a new error using cause.Explain and then writes it out to the
// default handler at error severity.
func Fail(ctx context.Context, err error, msg string) {
	page := cause.Explain(severity.NewContext(ctx, severity.Error), err, msg).Page
	output.Send(ctx, page)
}

// Failf forms a new error using cause.Explainf and then writes it out to the
// default handler at error severity.
func Failf(ctx context.Context, err error, msg string, args ...interface{}) {
	page := cause.Explainf(severity.NewContext(ctx, severity.Error), err, msg, args...).Page
	output.Send(ctx, page)
}

// Fatal forms a new error using cause.Explain and then writes it out to the
// default handler at critical severity.
func Fatal(ctx context.Context, err error, msg string) {
	page := cause.Explain(severity.NewContext(ctx, severity.Critical), err, msg).Page
	output.Send(ctx, page)
}

// Fatalf forms a new error using cause.Explainf and then writes it out to the
// default handler at critical severity.
func Fatalf(ctx context.Context, err error, msg string, args ...interface{}) {
	page := cause.Explainf(severity.NewContext(ctx, severity.Critical), err, msg, args...).Page
	output.Send(ctx, page)
}

// Error is At(ctx, severity.Error)
// It is used to get a jotter for the given context at the error level.
func Error(ctx context.Context) Jotter {
	return At(ctx, severity.Error)
}

// Errorf is Error(ctx).Printf(message, args...)
// It simple helper for the common case of printing error messages.
func Errorf(ctx context.Context, message string, args ...interface{}) {
	Error(ctx).Printf(message, args...)
}

// Warning is At(ctx, severity.Warning)
// It is used to get a jotter for the given context at the warning level.
func Warning(ctx context.Context) Jotter {
	return At(ctx, severity.Warning)
}

// Warningf is Warning(ctx).Printf(message, args...)
// It simple helper for the common case of printing warning messages.
func Warningf(ctx context.Context, message string, args ...interface{}) {
	Warning(ctx).Printf(message, args...)
}

// Notice is At(ctx, severity.Notice)
// It is used to get a jotter for the given context at the notice level.
func Notice(ctx context.Context) Jotter {
	return At(ctx, severity.Notice)
}

// Noticef is Notice(ctx).Printf(message, args...)
// It simple helper for the common case of printing noticies.
func Noticef(ctx context.Context, message string, args ...interface{}) {
	Notice(ctx).Printf(message, args...)
}

// Info is At(ctx, errors.Info)
// It is used to get a jotter for the given context at the info level.
func Info(ctx context.Context) Jotter {
	return At(ctx, severity.Info)
}

// Infof is Info(ctx).Printf(message, args...)
// It simple helper for the common case of printing informational messages.
func Infof(ctx context.Context, message string, args ...interface{}) {
	Info(ctx).Printf(message, args...)
}

// Debug is At(ctx, errors.Debug)
// It is used to get a jotter for the given context at the debug level.
func Debug(ctx context.Context) Jotter {
	return At(ctx, severity.Debug)
}

// Debugf is Debug(ctx).Printf(message, args...)
// It simple helper for the common case of printing debug messages.
func Debugf(ctx context.Context, message string, args ...interface{}) {
	Debug(ctx).Printf(message, args...)
}
