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

package log

import (
	"sync"
)

// Channel is a log handler that passes log messages to another Handler through
// a chan.
// This makes this Handler safe to use from multiple threads.
func Channel(to Handler, size int) (h Handler) {
	c := make(chan *Message, size)
	done := make(chan struct{})
	go func() {
		for r := range c {
			to.Handle(r)
		}
		close(done)
	}()
	once := sync.Once{}
	closer := func() {
		once.Do(func() {
			close(c)
			<-done
		})
		to.Close()
	}
	return &handler{func(m *Message) { c <- m }, closer}
}
