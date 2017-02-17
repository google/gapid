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

// InvokeOptions is a collection of bit flags controlling an invoke.
type InvokeOptions int

const (
	// InvokeSingleThreaded prevents the resume of all other threads when
	// performing the invoke. Once the invoke has finished, the single thread will
	// suspended again.
	InvokeSingleThreaded = InvokeOptions(1)

	// InvokeNonvirtual invokes the method without using regular, virtual
	// invocation.
	InvokeNonvirtual = InvokeOptions(2)
)

func (i InvokeOptions) String() string {
	parts := []string{}
	if i&InvokeSingleThreaded != 0 {
		parts = append(parts, "InvokeSingleThreaded")
	}
	if i&InvokeNonvirtual != 0 {
		parts = append(parts, "InvokeNonvirtual")
	}
	return strings.Join(parts, ", ")
}
