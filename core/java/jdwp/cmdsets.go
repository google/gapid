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

// cmdSet is the namespace for a command identifier.
type cmdSet uint8

const (
	cmdSetVirtualMachine       = cmdSet(1)
	cmdSetReferenceType        = cmdSet(2)
	cmdSetClassType            = cmdSet(3)
	cmdSetArrayType            = cmdSet(4)
	cmdSetInterfaceType        = cmdSet(5)
	cmdSetMethod               = cmdSet(6)
	cmdSetField                = cmdSet(8)
	cmdSetObjectReference      = cmdSet(9)
	cmdSetStringReference      = cmdSet(10)
	cmdSetThreadReference      = cmdSet(11)
	cmdSetThreadGroupReference = cmdSet(12)
	cmdSetArrayReference       = cmdSet(13)
	cmdSetClassLoaderReference = cmdSet(14)
	cmdSetEventRequest         = cmdSet(15)
	cmdSetStackFrame           = cmdSet(16)
	cmdSetClassObjectReference = cmdSet(17)
	cmdSetEvent                = cmdSet(64)
)
