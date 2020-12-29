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

package main

import (
	"os"
	"syscall"
)

func createConsole() error {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}

	p, err := dll.FindProc("AllocConsole")
	if err != nil {
		return err
	}

	r, _, err := p.Call()
	if r == 0 {
		return err
	}

	// Re-open the standard IO streams. This (and the getStdHandle function below) is copied from
	// golang's https://golang.org/src/syscall/syscall_windows.go.
	os.Stdin = os.NewFile(uintptr(getStdHandle(syscall.STD_INPUT_HANDLE)), "/dev/stdin")
	os.Stdout = os.NewFile(uintptr(getStdHandle(syscall.STD_OUTPUT_HANDLE)), "/dev/stdout")
	os.Stderr = os.NewFile(uintptr(getStdHandle(syscall.STD_ERROR_HANDLE)), "/dev/stderr")

	return nil
}

func getStdHandle(h int) (fd syscall.Handle) {
	r, _ := syscall.GetStdHandle(h)
	syscall.CloseOnExec(r)
	return r
}
