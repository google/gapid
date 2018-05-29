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

package adb_test

import (
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
)

func TestLogcat(t_ *testing.T) {
	ctx, _ := task.WithDeadline(log.Testing(t_), time.Now().Add(3*time.Second))
	d := mustConnect(ctx, "logcat_device")
	msgs := make(chan android.LogcatMessage, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := d.Logcat(ctx, msgs)
		assert.For(ctx, "err").ThatError(err).Succeeded()
	}()
	year, loc := time.Now().Year(), time.Local
	expected := []android.LogcatMessage{
		{
			Timestamp: time.Date(year, time.March, 29, 15, 16, 29, 514*1e6, loc),
			ProcessID: 24153,
			ThreadID:  24153,
			Priority:  android.Verbose,
			Tag:       "AndroidRuntime",
			Message:   ">>>>>> START com.android.internal.os.RuntimeInit uid 0 <<<<<<\n",
		},
		{
			Timestamp: time.Date(year, time.March, 29, 15, 16, 29, 518*1e6, loc),
			ProcessID: 24153,
			ThreadID:  24153,
			Priority:  android.Debug,
			Tag:       "AndroidRuntime",
			Message:   "CheckJNI is OFF\n",
		},
		{
			Timestamp: time.Date(year, time.March, 29, 15, 16, 29, 761*1e6, loc),
			ProcessID: 31608,
			ThreadID:  31608,
			Priority:  android.Info,
			Tag:       "Finsky",
			Message:   "[1] PackageVerificationReceiver.onReceive: Verification requested, id = 331",
		},
		{
			Timestamp: time.Date(year, time.March, 29, 15, 16, 32, 205*1e6, loc),
			ProcessID: 31608,
			ThreadID:  31655,
			Priority:  android.Warning,
			Tag:       "qtaguid",
			Message:   "Failed write_ctrl(u 48) res=-1 errno=22",
		},
		{
			Timestamp: time.Date(year, time.March, 29, 15, 16, 32, 205*1e6, loc),
			ProcessID: 31608,
			ThreadID:  31655,
			Priority:  android.Error,
			Tag:       "NetworkManagementSocketTagger",
			Message:   "untagSocket(48) failed with errno -22",
		},
		{
			Timestamp: time.Date(year, time.March, 29, 15, 16, 32, 219*1e6, loc),
			ProcessID: 31608,
			ThreadID:  31608,
			Priority:  android.Fatal,
			Tag:       "Finsky",
			Message:   "[1] PackageVerificationReceiver.onReceive: Verification requested, id = 331",
		},
	}
	for _, msg := range expected {
		assert.For(ctx, "msg").That(<-msgs).Equals(msg)
	}
	assert.For(ctx, "msg").That(<-msgs).Equals(android.LogcatMessage{})
	<-done
}
