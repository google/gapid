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

//go:build crashreporting

package reporting

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
)

const testCrashReport = false

func TestReport(t *testing.T) {
	if testCrashReport {
		ctx := log.Testing(t)

		_, err := Reporter{
			AppName:    "crash-reporting-test",
			AppVersion: "0",
			OSName:     "unknown",
			OSVersion:  "unknown",
		}.reportStacktrace(stacktrace.Capture(), crashStagingURL)

		assert.For(ctx, "err").ThatError(err).Succeeded()
	}
}
