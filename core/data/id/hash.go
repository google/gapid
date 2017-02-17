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

package id

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var sha1Pool = sync.Pool{New: func() interface{} { return sha1.New() }}

// Hash creates a new ID by calling f and hashing all data written to w.
func Hash(f func(w io.Writer) error) (ID, error) {
	h := sha1Pool.Get().(hash.Hash)
	defer sha1Pool.Put(h)
	h.Reset()
	err := f(h)
	id := ID{}
	copy(id[:], h.Sum(nil))
	return id, err
}

// OfBytes calculates the ID for the supplied data using the Hash function.
func OfBytes(data ...[]byte) ID {
	id, _ := Hash(func(w io.Writer) error {
		for _, d := range data {
			w.Write(d)
		}
		return nil
	})
	return id
}

// OfString calculates the ID for the supplied strings using the Hash function.
func OfString(strings ...string) ID {
	id, _ := Hash(func(w io.Writer) error {
		for _, s := range strings {
			io.WriteString(w, s)
		}
		return nil
	})
	return id
}

// uniqueNumber is used to generate a unique number per call to the Unique function.
// Needed in case Unique is called faster than the resolution of time.Now
var uniqueNumber uint32

// Unique returns a unique identifier by hashing the current time and machine hostname.
func Unique() ID {
	id, _ := Hash(func(w io.Writer) error {
		name, _ := os.Hostname()
		num := atomic.AddUint32(&uniqueNumber, 1)
		fmt.Fprint(w, "%s:%v:%v", name, time.Now(), num)
		return nil
	})
	return id
}
