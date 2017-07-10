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

package web

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type static string
type staticFile struct {
	name string
	*bytes.Reader
}

func (s static) Open(name string) (http.File, error) {
	fullname := string(s) + name
	body, found := embedded[fullname]
	f := &staticFile{name: filepath.Base(name)}
	if strings.HasSuffix(f.name, string(filepath.Separator)) {
		return f, nil
	}
	if !found {
		return nil, os.ErrNotExist
	}
	if embedded_utf8[fullname] {
		f.Reader = bytes.NewReader(([]byte)(body))
	} else {
		data, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return nil, err
		}
		f.Reader = bytes.NewReader(data)
	}
	return f, nil
}

func (staticFile) Close() error {
	return nil
}

func (f *staticFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (f *staticFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *staticFile) Name() string {
	return f.name
}

func (f *staticFile) Mode() os.FileMode {
	return 0777
}

func (f *staticFile) ModTime() time.Time {
	return time.Now()
}

func (f *staticFile) IsDir() bool {
	return f.Reader == nil
}

func (f *staticFile) Sys() interface{} {
	return nil
}
