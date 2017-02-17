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

package compare

// Handler is the type for a callback that differences are delivered to.
// It may panic with LimitReached to abort further difference processing.
type Handler func(Path)

// collect is used to collect a set of differences together, up to the cap limit.
type collect []Path

func (d *collect) add(p Path) {
	if len(*d) == cap(*d) {
		panic(LimitReached)
	}
	*d = append(*d, p)
}

// test is a difference handler for equality testing.
type test bool

func (d *test) set(Path) {
	*d = true
	panic(LimitReached)
}
