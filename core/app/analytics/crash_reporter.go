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

//go:build analytics

package analytics

import (
	"bytes"
	"encoding/base64"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/fault/stacktrace/crunch"
)

const (
	maxExceptionLength = 150
)

func init() {
	crash.Register(onCrash)
}

func onCrash(e interface{}, s stacktrace.Callstack) {
	filter := stacktrace.MatchPackage("github.com/google/gapid/.*")
	stack := s.Filter(stacktrace.Trim(filter))
	encoded := encodeCrashCode(stack, maxExceptionLength)
	SendException(encoded, true)
}

func encodeCrashCode(c stacktrace.Callstack, maxLen int) string {
	for len(c) > 0 {
		buf := bytes.Buffer{}
		w := base64.NewEncoder(base64.RawStdEncoding, &buf)
		w.Write(crunch.Crunch(c))
		w.Close()
		s := buf.String()
		if len(s) > maxLen {
			c = c[:len(c)-1]
			continue
		}
		return s
	}
	return "<none>"
}
