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

// +build !windows

package bind

import (
	"context"
	"io/ioutil"
	"path/filepath"
)

// ListExecutables returns the executables in a particular directory as given by path
func (b *Simple) ListExecutables(ctx context.Context, path string) ([]string, error) {
	if path == "" {
		path = "/"
	}
	rets := []string{}
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return rets, nil
	}
	for _, inf := range infos {
		if inf.Mode().IsRegular() && (inf.Mode()&0500) == 0500 {
			rets = append(rets, inf.Name())
		}
	}
	return rets, nil
}

// GetURIRoot returns the root URI for the entire system
func (b *Simple) GetURIRoot() string {
	return "/"
}

// ListDirectories returns a list of directories rooted at a particular path
func (b *Simple) ListDirectories(ctx context.Context, path string) ([]string, error) {
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

// SupportsPerfetto returns true if the given device supports taking a
// Perfetto trace.
func (b *Simple) SupportsPerfetto(ctx context.Context) bool {
	if support, err := b.IsFile(ctx, "/tmp/perfetto-consumer"); err == nil {
		return support
	}
	return false
}
