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

import "fmt"

// EventKind represents the type of event to set, or being raised.
type EventKind uint8

const (
	// SingleStep is the kind of event raised when a single-step has been completed.
	SingleStep = EventKind(1)
	// Breakpoint is the kind of event raised when a breakpoint has been hit.
	Breakpoint = EventKind(2)
	// FramePop is the kind of event raised when a stack-frame is popped.
	FramePop = EventKind(3)
	// Exception is the kind of event raised when an exception is thrown.
	Exception = EventKind(4)
	// UserDefined is the kind of event raised when a user-defind event is fired.
	UserDefined = EventKind(5)
	// ThreadStart is the kind of event raised when a new thread is started.
	ThreadStart = EventKind(6)
	// ThreadDeath is the kind of event raised when a thread is stopped.
	ThreadDeath = EventKind(7)
	// ClassPrepare is the kind of event raised when a class enters the prepared state.
	ClassPrepare = EventKind(8)
	// ClassUnload is the kind of event raised when a class is unloaded.
	ClassUnload = EventKind(9)
	// ClassLoad is the kind of event raised when a class enters the loaded state.
	ClassLoad = EventKind(10)
	// FieldAccess is the kind of event raised when a field is accessed.
	FieldAccess = EventKind(20)
	// FieldModification is the kind of event raised when a field is modified.
	FieldModification = EventKind(21)
	// ExceptionCatch is the kind of event raised when an exception is caught.
	ExceptionCatch = EventKind(30)
	// MethodEntry is the kind of event raised when a method has been entered.
	MethodEntry = EventKind(40)
	// MethodExit is the kind of event raised when a method has been exited.
	MethodExit = EventKind(41)
	// VMStart is the kind of event raised when the virtual machine is initialized.
	VMStart = EventKind(90)
	// VMDeath is the kind of event raised when the virtual machine is shutdown.
	VMDeath = EventKind(99)
)

func (k EventKind) String() string {
	switch k {
	case SingleStep:
		return "SingleStep"
	case Breakpoint:
		return "Breakpoint"
	case FramePop:
		return "FramePop"
	case Exception:
		return "Exception"
	case UserDefined:
		return "UserDefined"
	case ThreadStart:
		return "ThreadStart"
	case ThreadDeath:
		return "ThreadDeath"
	case ClassPrepare:
		return "ClassPrepare"
	case ClassUnload:
		return "ClassUnload"
	case ClassLoad:
		return "ClassLoad"
	case FieldAccess:
		return "FieldAccess"
	case FieldModification:
		return "FieldModification"
	case ExceptionCatch:
		return "ExceptionCatch"
	case MethodEntry:
		return "MethodEntry"
	case MethodExit:
		return "MethodExit"
	case VMStart:
		return "VMStart"
	case VMDeath:
		return "VMDeath"
	default:
		return fmt.Sprintf("EventKind<%d>", int(k))
	}
}

// event returns a default-initialzed Event of the specified kind.
func (k EventKind) event() Event {
	switch k {
	case SingleStep:
		return &EventSingleStep{}
	case Breakpoint:
		return &EventBreakpoint{}
	case Exception:
		return &EventException{}
	case ThreadStart:
		return &EventThreadStart{}
	case ThreadDeath:
		return &EventThreadDeath{}
	case ClassPrepare:
		return &EventClassPrepare{}
	case ClassUnload:
		return &EventClassUnload{}
	case FieldAccess:
		return &EventFieldAccess{}
	case FieldModification:
		return &EventFieldModification{}
	case ExceptionCatch:
		return &EventException{}
	case MethodEntry:
		return &EventMethodEntry{}
	case MethodExit:
		return &EventMethodExit{}
	case VMStart:
		return &EventVMStart{}
	case VMDeath:
		return &EventVMDeath{}
	default:
		return nil
	}
}
