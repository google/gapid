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

// Filter is used by the log system to drop log records that don't match the filter.
// It is invoked for each property attached to the log context.
// It should return true if the context passes the filter.
type Filter string

// PreFilter is an optimised version of Filter that is invoked only for the initial setting of the Severity property.
// This is an optimization for the most common binning case.
// It should return true if the severity is active.
type PreFilter severity.Level

// GetFilter gets the active Filter for this context.
func GetFilter(ctx Context) Filter {
	return ""
}

// GetPreFilter gets the active PreFilter for this context.
func GetPreFilter(ctx Context) PreFilter {
	return PreFilter(severity.GetFilter(ctx.Unwrap()))
}

// Filter returns a context with the given Filter set on it.
func (ctx logContext) Filter(f Filter) Context {
	return ctx
}

// PreFilter returns a context with the given PreFilter set on it.
func (ctx logContext) PreFilter(f PreFilter) Context {
	return Wrap(severity.Filter(ctx.Unwrap(), severity.Level(f)))
}

// Pass is an implementation of Filter that does no filtering.
const Pass = Filter("Pass")

// Null is an implementation of PreFilter that drops all log messages.
const Null = Filter("Null")

// Limit returns an implementation of PreFilter that allows only log messages of equal or higher priority to the
// specified limit.
func Limit(limit severity.Level) PreFilter {
	return PreFilter(limit)
}

// NoLimit is an implementation of PreFilter that allows all messages through.
const NoLimit = PreFilter(severity.Debug + 10)
