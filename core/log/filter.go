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

// Filter is the filter of log messages.
type Filter interface {
	// ShowSeverity returns true if the message of severity s should be shown.
	ShowSeverity(s Severity) bool
}

type filterKeyTy string

const filterKey filterKeyTy = "log.filterKey"

// PutFilter returns a new context with the Filter assigned to w.
func PutFilter(ctx context.Context, w Filter) context.Context {
	return keys.WithValue(ctx, filterKey, w)
}

// GetFilter returns the Filter assigned to ctx.
func GetFilter(ctx context.Context) Filter {
	out, _ := ctx.Value(filterKey).(Filter)
	return out
}

// SeverityFilter implements the Filter interface which filters out any messages
// below the severity value.
type SeverityFilter Severity

// ShowSeverity returns true if the message of severity s should be shown.
func (f SeverityFilter) ShowSeverity(s Severity) bool { return Severity(f) <= s }
