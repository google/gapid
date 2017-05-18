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

package memory

import (
	"fmt"
	"io"
)

// Writer returns a binary writer for the specified memory pool and range.
func Writer(p *Pool, rng Range) io.Writer {
	b := make([]byte, rng.Size)
	p.Write(rng.Base, Blob(b))
	w := writer(b)
	return &w
}

type writer []byte

func (w *writer) Write(p []byte) (n int, err error) {
	n = copy(*w, p)
	if n < len(p) {
		err = fmt.Errorf("Write overflowed buffer (buffer len: %v, data len: %v)", len(*w), len(p))
	}
	*w = (*w)[n:]
	return
}
