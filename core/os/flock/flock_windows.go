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

//go:build windows

package flock

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel, _       = syscall.LoadLibrary("kernel32.dll")
	lockFileEx, _   = syscall.GetProcAddress(kernel, "LockFileEx")
	unlockFileEx, _ = syscall.GetProcAddress(kernel, "UnlockFileEx")
)

const (
	failImmediately = 0x00000001
	exclusiveLock   = 0x00000002
)

func sysTryLock(f *os.File) error {
	r1, _, _ := syscall.Syscall6(
		uintptr(lockFileEx),
		uintptr(6),
		uintptr(f.Fd()),
		uintptr(exclusiveLock|failImmediately),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&syscall.Overlapped{})))
	if r1 == 1 {
		return nil
	}
	return errors.New("Failed on locking file")
}

func sysUnlock(f *os.File) error {
	r1, _, _ := syscall.Syscall6(
		uintptr(unlockFileEx),
		uintptr(5),
		uintptr(f.Fd()),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&syscall.Overlapped{})),
		uintptr(0))
	if r1 == 1 {
		return nil
	}
	return errors.New("Failed on unlocking file")
}
