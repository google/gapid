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

package flock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestDeviceFLock(t *testing.T) {
	defer RemoveAllDeviceFLockUnderlyingFiles()
	mustNewDeviceFLock := func(sn string) *DeviceFLock {
		dfl, err := NewDeviceFLock(sn)
		assert.To(t).For("NewDeviceFLock(sn: %s), result: *DeviceFLock: %p, err: %v", sn, dfl, err).That(err).IsNil()
		assert.To(t).For("NewDeviceFLock(sn: %s), result: *DeviceFLock: %p, err: %v", sn, dfl, err).That(dfl).IsNotNil()
		return dfl
	}
	expectTryLock := func(fl *DeviceFLock, expectResult bool) {
		locked := fl.TryLock()
		assert.To(t).For("DeviceFLock.tryLock(), path: %s, expect result: %v, actual result: %v", fl.fl.path, expectResult, locked).That(locked).Equals(expectResult)
	}
	expectUnlock := func(fl *DeviceFLock, expectResult bool) {
		unlocked := fl.Unlock()
		assert.To(t).For("DeviceFLock.unlock(), path: %s, expect result: %v, actual result: %v", fl.fl.path, expectResult, unlocked).That(unlocked).Equals(expectResult)
	}
	expectTryLockDevice := func(sn string, expectLocked bool) *DeviceFLock {
		dfl, locked, err := TryLockDevice(sn)
		assert.To(t).For("TryLockDevice(sn: %s), expectLocked: %v, result: *DeviceFLock: %p, locked: %v, err: %v", expectLocked, dfl, locked, err).That(err).IsNil()
		assert.To(t).For("TryLockDevice(sn: %s), expectLocked: %v, result: *DeviceFLock: %p, locked: %v, err: %v", expectLocked, dfl, locked, err).That(locked).Equals(expectLocked)
		assert.To(t).For("TryLockDevice(sn: %s), expectLocked: %v, result: *DeviceFLock: %p, locked: %v, err: %v", expectLocked, dfl, locked, err).That(dfl).IsNotNil()
		return dfl
	}
	mustLockDevice := func(sn string) *DeviceFLock {
		dfl, err := LockDevice(sn)
		assert.To(t).For("LockDevice(sn: %s), result: *DeviceFLock: %p, err: %v", dfl, err).That(err).IsNil()
		assert.To(t).For("LockDevice(sn: %s), result: *DeviceFLock: %p, err: %v", dfl, err).That(dfl).IsNotNil()
		return dfl
	}
	done := make(chan bool)

	// Single lock and unlock
	dflA1 := mustNewDeviceFLock("A")
	expectTryLock(dflA1, true)
	expectUnlock(dflA1, true)

	// Double unlock
	expectTryLock(dflA1, true)
	expectUnlock(dflA1, true)
	expectUnlock(dflA1, true)

	// Double lock
	expectTryLock(dflA1, true)
	expectTryLock(dflA1, true)
	expectUnlock(dflA1, true)

	// Mutex
	dflA2 := mustNewDeviceFLock("A")
	expectTryLock(dflA1, true)
	expectTryLock(dflA2, false)
	expectUnlock(dflA2, false)
	expectUnlock(dflA1, true)
	expectTryLock(dflA2, true)
	expectUnlock(dflA2, true)

	// Blocking
	expectTryLock(dflA1, true)
	go func() {
		defer expectUnlock(dflA2, true)
		dflA2.Lock()
		done <- true
	}()
	expectUnlock(dflA1, true)
	<-done

	// Helper function: TryLockDevice
	dflA3 := expectTryLockDevice("A", true)
	dflA4 := expectTryLockDevice("A", false)
	expectUnlock(dflA3, true)
	dflA5 := expectTryLockDevice("A", true)
	expectUnlock(dflA4, false)
	expectUnlock(dflA5, true)

	// Helper function: LockDevice
	expectTryLock(dflA1, true)
	go func() {
		dflA6 := mustLockDevice("A")
		defer expectUnlock(dflA6, true)
		done <- true
	}()
	expectUnlock(dflA1, true)
	<-done

	// Helper function: ReleaseAllDeviceFLocks
	dflB := expectTryLockDevice("B", true)
	dflC := expectTryLockDevice("C", true)
	dflD := expectTryLockDevice("D", true)
	err := RemoveAllDeviceFLockUnderlyingFiles()
	assert.To(t).For("ReleaseAllDeviceFLocks should succeed").That(err).IsNil()
	cleared := true
	filepath.Walk(os.TempDir(), func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, deviceLockFileSuffix) && !info.IsDir() {
			cleared = false
		}
		return nil
	})
	assert.To(t).For("ReleaseAllDeviceFLocks should succeed").That(cleared).Equals(true)
	expectUnlock(dflB, true)
	expectUnlock(dflC, true)
	expectUnlock(dflD, true)
}
