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

package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
)

func TestContextCancel(t *testing.T) {
	assert := assert.To(t)
	before := context.Background()
	after, cancel := task.WithCancel(before)
	assert.For("Cancel builds new context").That(before).NotEquals(after)
	assert.For("Stopped before cancel").That(task.Stopped(after)).Equals(false)
	assert.For("StopReason before cancel").That(task.StopReason(after)).IsNil()
	cancel()
	assert.For("Stopped after cancel").That(task.Stopped(after)).Equals(true)
	assert.For("StopReason after cancel").That(task.StopReason(after)).Equals(context.Canceled)
}

func TestContextTimeout(t *testing.T) {
	assert := assert.To(t)
	before := context.Background()
	duration := ExpectBlocking
	after, _ := task.WithTimeout(before, duration)
	select {
	case <-task.ShouldStop(after):
		// expected
	case <-time.After(ExpectBlocking + ExpectNonBlocking):
		// should have cancelled by now
		assert.For("timeout").Error("context was not cancelled in time")
	}
}
