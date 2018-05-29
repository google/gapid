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
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestMutex(t *testing.T) {
	assert := assert.To(t)
	ReleaseAllLocks()
	defer ReleaseAllLocks()
	done := make(chan bool)

	expectLocked := func(m *Mutex, expectResult bool) {
		locked := m.Locked()
		assert.For("Mutex.Locked(), expect result: %v, actual result: %v", expectResult, locked).That(locked).Equals(expectResult)
	}
	expectTryLockInMutex := func(m *Mutex, expectResult bool) {
		locked := m.TryLock()
		assert.For("Mutex.TryLock(), file: %s, expect result: %v, actual result: %v", m.fm.p, expectResult, locked).That(locked).Equals(expectResult)
	}
	expectUnlock := func(m *Mutex, expectResult bool) {
		unlocked := m.Unlock()
		assert.For("Mutex.Unlock(), file: %s, expect result: %v, actual result: %v", m.fm.p, expectResult, unlocked).That(unlocked).Equals(expectResult)
	}
	expectTryLock := func(n string, expectLocked bool) *Mutex {
		m := TryLock(n)
		locked := m.Locked()
		assert.For("TryLock(%s), file: %v, expectLocked: %v, actual locked: %v", n, m.fm.p, expectLocked, locked).That(locked).Equals(expectLocked)
		return m
	}
	mustLock := func(n string) *Mutex {
		m := Lock(n)
		assert.For("Lock(%s), expect locked, actual locked: %v", m.Locked()).That(m.Locked()).Equals(true)
		return m
	}

	// Basic lock and unlock
	dflA1 := New("A")
	expectLocked(dflA1, false)
	expectTryLockInMutex(dflA1, true)
	expectLocked(dflA1, true)
	expectUnlock(dflA1, true)
	expectLocked(dflA1, false)

	// Mutex
	dflA2 := New("A")
	expectTryLockInMutex(dflA1, true)
	expectTryLockInMutex(dflA2, false)
	expectLocked(dflA2, false)
	expectUnlock(dflA1, true)
	expectTryLockInMutex(dflA2, true)
	expectLocked(dflA1, false)
	expectLocked(dflA2, true)
	expectUnlock(dflA2, true)

	// Blocking
	expectTryLockInMutex(dflA1, true)
	go func() {
		expectLocked(dflA2, false)
		defer expectUnlock(dflA2, true)
		dflA2.Lock()
		expectLocked(dflA2, true)
		done <- true
	}()
	expectLocked(dflA1, true)
	expectUnlock(dflA1, true)
	expectLocked(dflA1, false)
	<-done

	// Helper function: TryLockDevice
	dflA3 := expectTryLock("A", true)
	dflA4 := expectTryLock("A", false)
	expectUnlock(dflA3, true)
	expectLocked(dflA4, false)

	// Helper function: LockDevice
	expectTryLockInMutex(dflA1, true)
	go func() {
		dflA6 := mustLock("A")
		defer expectUnlock(dflA6, true)
		done <- true
	}()
	expectUnlock(dflA1, true)
	<-done

	// Helper function: ReleaseAllLocks
	dflB := expectTryLock("B", true)
	dflC := expectTryLock("C", true)
	expectUnlock(dflB, true)
	expectUnlock(dflC, true)
	err := ReleaseAllLocks()
	assert.For("ReleaseAllLocks should succeed").That(err).IsNil()
}
