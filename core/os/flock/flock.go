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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var errAlreadyLocked = errors.New("FLock already locked by others")

const (
	deviceLockFileSuffix = ".gapid-device-flock"
)

// RemoveAllDeviceFLockUnderlyingFiles releases all the FLocks by removing all
// the underlying files, files whose name ends with `deviceLockFileSuffix`.
func RemoveAllDeviceFLockUnderlyingFiles() error {
	return filepath.Walk(os.TempDir(), func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, deviceLockFileSuffix) {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		return os.Remove(path)
	})
}

// LockDevice acquires the lock for the device specified by the given serial or
// name of the device in a Blocking way, returns the DeviceFLock which holds the
// lock and error.
func LockDevice(sn string) (*DeviceFLock, error) {
	l, err := NewDeviceFLock(sn)
	if err != nil {
		return nil, err
	}
	l.Lock()
	return l, nil
}

// TryLockDevice creates a DeviceFLock and use it to acquires the lock for the
// device specified by the given serial or name of the device in a Non-Blocking
// way, returns the created DeviceFLock, boolean to show whether the lock has
// been acquired or not, and error.
func TryLockDevice(sn string) (*DeviceFLock, bool, error) {
	l, err := NewDeviceFLock(sn)
	if err != nil {
		return nil, false, err
	}
	locked := l.TryLock()
	return l, locked, nil
}

// DeviceFLock is a file-based lock for devices valid for a fixed peroid of
// time, specificlly for locking the use of devices. I.e, one device can have
// only one FLock, and after a fixed period of time, the lock will be invalid,
// so others can acquire the FLock to use the corresponding device.
type DeviceFLock struct {
	sn string
	fl *fLock
	m  sync.Mutex
}

// NewDeviceFLock creates a new File-lock based lock for the device
// identified by the given serial/name.
func NewDeviceFLock(sn string) (*DeviceFLock, error) {
	l := &DeviceFLock{
		sn: sn,
		fl: &fLock{},
	}
	var err error
	l.fl.path, err = deviceFLockPath(sn)
	return l, err
}

// TryLock acquires the lock in a non-blocking way, returns true if the lock is
// acquired by the DeviceFLock, false otherwise.
func (l *DeviceFLock) TryLock() bool {
	l.m.Lock()
	defer l.m.Unlock()
	locked, err := l.fl.tryLock()
	if err != nil {
		return false
	}
	return locked
}

// Lock acquires the lock in a blocking way.
func (l *DeviceFLock) Lock() {
	l.m.Lock()
	defer l.m.Unlock()
	l.fl.lock()
	return
}

// Unlock releases the lock, returns true if succeeded, false otherwise.
func (l *DeviceFLock) Unlock() bool {
	l.m.Lock()
	defer l.m.Unlock()
	l.fl.unlock()
	_, err := os.Stat(l.fl.path)
	if os.IsNotExist(err) {
		return true
	}
	return false
}

func deviceFLockPath(sn string) (string, error) {
	n := sn + deviceLockFileSuffix
	return filepath.Abs(filepath.Join(os.TempDir(), n))
}

// fLock is a file-based lock. It locks by atomically creating and opening a
// file and releases the lock by close and remove the file.
type fLock struct {
	path    string
	modTime time.Time
	m       sync.Mutex
	locked  bool
}

// syncLockedState updates the locked state if the underlying file has been
// changed or removed.
func (l *fLock) syncLockedState() {
	if !l.locked {
		return
	}
	info, err := os.Stat(l.path)
	// The underlying file has already been removed.
	if os.IsNotExist(err) {
		l.locked = false
		return
	}
	if info == nil {
		l.locked = false
		return
	}
	// The underlying file has been changed by others (recreated).
	if info.ModTime() != l.modTime {
		l.locked = false
		return
	}
}

// tryLock acquires the lock in a non-blocking way. Call tryLock on an already
// locked fLock will return immediately will a success lock. However, this is
// not a recursive lock.
func (l *fLock) tryLock() (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	l.syncLockedState()
	if l.locked {
		return true, nil
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL, 0400)
	if os.IsExist(err) {
		return false, errAlreadyLocked
	}
	if err != nil {
		return false, err
	}
	if err := f.Close(); err != nil {
		os.Remove(l.path)
		return false, err
	}
	info, err := os.Stat(l.path)
	if err != nil {
		return false, err
	}
	l.modTime = info.ModTime()
	l.locked = true
	return true, nil
}

// lock acquires the lock in a blocking way.
func (l *fLock) lock() {
	for {
		locked, _ := l.tryLock()
		if locked {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (l *fLock) unlock() (bool, error) {
	l.m.Lock()
	defer l.m.Unlock()
	l.syncLockedState()
	if !l.locked {
		return true, nil
	}
	if err := os.Remove(l.path); err != nil {
		return false, err
	}
	l.locked = false
	l.modTime = time.Time{}
	return true, nil
}
