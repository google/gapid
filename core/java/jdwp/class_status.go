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

package jdwp

import "strings"

// ClassStatus is an enumerator of class loading state.
// See https://docs.oracle.com/javase/specs/jvms/se7/html/jvms-5.html#jvms-5.3
// for detailed descriptions of the loading states.
type ClassStatus int

const (
	// StatusVerified is used to describe a class in the verified state.
	StatusVerified = ClassStatus(1)
	// StatusPrepared is used to describe a class in the prepared state.
	StatusPrepared = ClassStatus(2)
	// StatusInitialized is used to describe a class in the initialized state.
	StatusInitialized = ClassStatus(4)
	// StatusError is used to describe a class in the error state.
	StatusError = ClassStatus(8)
)

func (c ClassStatus) String() string {
	parts := []string{}
	if c&StatusVerified != 0 {
		parts = append(parts, "Verified")
	}
	if c&StatusPrepared != 0 {
		parts = append(parts, "Prepared")
	}
	if c&StatusInitialized != 0 {
		parts = append(parts, "Initialized")
	}
	if c&StatusError != 0 {
		parts = append(parts, "Error")
	}
	return strings.Join(parts, ", ")
}
