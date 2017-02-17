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

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/context/memo"
)

// Raw creates a raw mode logger for the specified channel.
func (ctx logContext) Raw(channel string) Logger {
	// return a context with only the handler embedded
	// TODO:
	n := context.Background()
	n = output.NewContext(n, output.FromContext(ctx.Unwrap()))
	if channel != "" {
		n = memo.Tag(n, channel)
	}
	return Logger{J: jot.To(n)}
}
