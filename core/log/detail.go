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
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/context/memo"
)

// The context key used to store detail text in the log context
const DetailKey = memo.TextValue("detail")

// GetDetail gets the detail text stored in the given context.
func GetDetail(ctx Context) string {
	tag, _ := ctx.Value(DetailKey).(string)
	return tag
}

// Detail returns a context with the detail text set.
func (ctx logContext) Detail(tag string) Context {
	return Wrap(keys.WithValue(ctx.Unwrap(), DetailKey, tag))
}

// Detail returns a logger with the detail text set if the logger is active.
func (l Logger) Detail(tag string) Logger {
	if l.Active() {
		l.J = l.J.With(DetailKey, tag)
	}
	return l
}
