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

package bind

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// ListExecutables returns the executables in a particular directory as given by path
func (b *binding) ListExecutables(ctx context.Context, path string) ([]string, error) {
	rets := []string{}
	if path == "" {
		return rets, nil
	}
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return rets, nil
	}
	for _, inf := range infos {
		if strings.HasSuffix(inf.Name(), ".exe") {
			rets = append(rets, inf.Name())
		}
	}
	return rets, nil
}

func (b *binding) drives(ctx context.Context) ([]string, error) {
	drives := []string{}

	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return drives, err
	}

	defer kernel32.Release()

	getDrives, err := kernel32.FindProc("GetLogicalDriveStringsW")
	if err != nil {
		return drives, err
	}

	l, _, _ := getDrives.Call(uintptr(0), uintptr(unsafe.Pointer(nil)))

	buff := make([]uint16, l)
	bs := uint32(l)

	hr, _, _ := getDrives.Call(uintptr(bs), uintptr(unsafe.Pointer(&buff[0])))

	if hr == 0 {
		return nil, fmt.Errorf("Could not get drive listing")
	}

	accumulation := []uint16{}
	for i := 0; i < int(hr); i++ {
		v := buff[i]
		if v == 0 {
			if len(accumulation) == 0 {
				return nil, fmt.Errorf("Unhandled drive listing output")
			}
			drives = append(drives, syscall.UTF16ToString(accumulation))
			accumulation = []uint16{}
		} else {
			accumulation = append(accumulation, v)
		}
	}
	if len(accumulation) > 0 {
		drives = append(drives, syscall.UTF16ToString(accumulation))
	}

	return drives, nil
}

// GetURIRoot returns the root URI for the entire system
func (b *binding) GetURIRoot() string {
	return ""
}

// ListDirectories returns a list of directories rooted at a particular path
func (b *binding) ListDirectories(ctx context.Context, path string) ([]string, error) {
	if path == "" {
		return b.drives(ctx)
	}
	rets := []string{}
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return rets, nil
	}
	for _, inf := range infos {
		if inf.Mode().IsDir() {
			if _, err := ioutil.ReadDir(filepath.Join(path, inf.Name())); err == nil {
				rets = append(rets, inf.Name())
			}
		}
	}
	return rets, nil
}
