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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	fileSuffix = ".gapid-flock"
	// panic messages
	acquireAlreadyOwnedLockMsg   = "Try to acquire the lock which is already held by current mutex."
	unlockAlreadyReleasedLockMsg = "Try to unlock an already released mutex."
	cannotMakeDirMsg             = "Cannot make temporary directory for flock files."
)

var dir = filepath.Join(os.TempDir(), "gapid-flocks")

// TryLock creates a Mutex by using a file named with the given string, and
// tries to acquire the inter-process lock, then returns the created Mutex. This
// is a non-blocking call. The returned Mutex may not hold the lock, and should
// be checked with Locked() to tell if the locked is held by the Mutex.
func TryLock(n string) *Mutex {
	m := New(n)
	m.TryLock()
	return m
}

// Lock creates a Mutex by using a file named with the given strings, and
// acquires the inter-process lock, then returns the created Mutex. This is a
// blocking call and the returned Mutex is guaranteed to hold the lock.
func Lock(n string) *Mutex {
	m := New(n)
	m.Lock()
	return m
}

// Mutex is a file based inter-process mutex. Mutex should only be created by
// New, Lock or TryLock.
type Mutex struct {
	fm *mutex
	tm sync.Mutex
}

// New creates a new file based inter-process mutex which uses the given string
// as the name of the underlying file. The Mutex created by New does not hold
// any lock.
func New(n string) *Mutex {
	return &Mutex{
		fm: &mutex{p: filepath.Join(dir, n+fileSuffix)},
	}
}

// Locked returns true if the Mutex holds the lock and valid to call Unlock,
// false if the Mutex does not hold the lock and valid to call TryLock or Lock.
// The state of locked/unlocked can only be changed by calling Lock, TryLock and
// Unlock functions, removing/modifing the underlying files does not change this
// state.
func (m *Mutex) Locked() bool {
	return m.fm.locked
}

// TryLock acquires the inter-process lock in a non-blocking way. Returns true
// if the lock is now held by the Mutex, or false if not. Panic if the Mutex
// already holds the lock.
func (m *Mutex) TryLock() bool {
	if m.fm.locked {
		panic(acquireAlreadyOwnedLockMsg)
	}
	m.tm.Lock()
	defer m.tm.Unlock()

	if err := m.fm.tryLock(); err != nil {
		return false
	}
	return true
}

// Lock acquires the inter-process lock in a blocking way, it only returns
// when it has acquired the lock. Panic if the Mutex already holds the lock
// when calling this function.
func (m *Mutex) Lock() {
	for {
		if m.TryLock() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Unlock releases the inter-process lock, returns true if the lock is released
// successfully, false otherwise. Panic if the Mutex has already been unlocked.
func (m *Mutex) Unlock() bool {
	if !m.fm.locked {
		panic(unlockAlreadyReleasedLockMsg)
	}
	m.tm.Lock()
	defer m.tm.Unlock()
	if err := m.fm.unlock(); err != nil {
		return false
	}
	return true
}

func openFile(path string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		panic(cannotMakeDirMsg)
	}
	f, ofe := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0755)
	if ofe == nil {
		return f, nil
	}
	if os.IsExist(ofe) {
		return os.Open(path)
	}
	return nil, ofe
}

type mutex struct {
	locked bool
	m      sync.Mutex
	f      *os.File
	p      string
}

func (m *mutex) tryLock() error {
	m.m.Lock()
	defer m.m.Unlock()
	if m.locked {
		panic(acquireAlreadyOwnedLockMsg)
	}
	f, err := openFile(m.p)
	if err != nil {
		return err
	}
	if err = sysTryLock(f); err != nil {
		f.Close()
		return err
	}
	m.f = f
	m.locked = true
	return nil
}

func (m *mutex) unlock() error {
	m.m.Lock()
	defer m.m.Unlock()
	if !m.locked {
		panic(unlockAlreadyReleasedLockMsg)
	}
	if m.f != nil {
		defer m.f.Close()
	}
	if err := sysUnlock(m.f); err != nil {
		return err
	}
	m.locked = false
	return nil
}

// ReleaseAllLocks releases all the FLocks by removing all the underlying files.
func ReleaseAllLocks() error {
	dirInfo, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("%v is not a directory", dir)
	}
	if err = os.RemoveAll(dir); err == nil {
		return nil
	}
	cantRemove := []string{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.Mode().IsRegular() {
			return nil
		}
		if err = os.Remove(path); err != nil {
			cantRemove = append(cantRemove, path)
		}
		return nil
	})
	if len(cantRemove) > 0 {
		return fmt.Errorf("Lock files cannot be removed: %v", cantRemove)
	}
	return nil
}
