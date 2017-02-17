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

package ast

import "fmt"

// ErrorCollector is an object which collects parsing errors.
type ErrorCollector []error

// Errorf adds an error defined by a format string to the collector.
func (e *ErrorCollector) Errorf(message string, args ...interface{}) {
	e.Error(fmt.Errorf(message, args...))
}

// Error adds the arguments to the collection of errors.
func (e *ErrorCollector) Error(err ...error) { *e = append(*e, err...) }

// GetErrors returns the collected errors.
func (e *ErrorCollector) GetErrors() []error { return []error(*e) }

// Concat concatenates two lists of errors.
func ConcatErrors(e1, e2 []error) []error {
	return append(
		append(
			make([]error, 0, len(e1)+len(e2)),
			e1...,
		),
		e2...,
	)
}
