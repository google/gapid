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

package fault

type (
	// List is the type for a list of errors.
	List []error
	// One is the type for something that collects only the first error.
	One struct{ err error }
)

// First returns the first error added to it.
func (l *List) First() error {
	if len(*l) <= 0 {
		return nil
	}
	return (*l)[0]
}

// Collect adds an error to the list.
func (l *List) Collect(err error) {
	*l = append(*l, err)
}

// First returns the first error added to it.
func (o *One) First() error {
	return o.err
}

// Collect adds an error to the list.
func (o *One) Collect(err error) {
	if o.err != nil {
		return
	}
	o.err = err
}
