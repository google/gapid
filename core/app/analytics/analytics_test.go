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

package analytics

import (
	"testing"
	"time"

	"github.com/google/gapid/core/log"
)

func Test(t *testing.T) {
	ctx := log.Testing(t)
	version := AppVersion{"test", "test", 1, 0, 0}
	// Want to just validate? Replace Enable() with:
	// endpoint := newValidateEndpoint()
	// encoder := newEncoder("abc123", "testOS", "testGPU", version)
	// send = newBatcher(endpoint, encoder)
	Enable(ctx, "test-user", version)
	OnError = func(err error) {
		t.Fatalf("Error: %v", err)
	}
	Send(SessionStart{})
	SendEvent("test", "cat", "meow")
	stop := SendTiming("test", "one-second")
	time.Sleep(time.Second)
	stop()
	Flush()
	Send(SessionEnd{})
	Disable()
}
