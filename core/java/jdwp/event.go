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

// events is a collection of events.
type events struct {
	Policy SuspendPolicy
	Events []Event
}

// Event is the interface implemented by all events raised by the VM.
type Event interface {
	request() EventRequestID
	Kind() EventKind
}

// EventVMStart represents an event raised when the virtual machine is started.
type EventVMStart struct {
	Request EventRequestID
	Thread  ThreadID
}

// EventVMDeath represents an event raised when the virtual machine is stopped.
type EventVMDeath struct {
	Request EventRequestID
}

// EventSingleStep represents an event raised when a single-step has been completed.
type EventSingleStep struct {
	Request  EventRequestID
	Thread   ThreadID
	Location Location
}

// EventBreakpoint represents an event raised when a breakpoint has been hit.
type EventBreakpoint struct {
	Request  EventRequestID
	Thread   ThreadID
	Location Location
}

// EventMethodEntry represents an event raised when a method has been entered.
type EventMethodEntry struct {
	Request  EventRequestID
	Thread   ThreadID
	Location Location
}

// EventMethodExit represents an event raised when a method has been exited.
type EventMethodExit struct {
	Request  EventRequestID
	Thread   ThreadID
	Location Location
}

// EventException represents an event raised when an exception is thrown.
type EventException struct {
	Request       EventRequestID
	Thread        ThreadID
	Location      Location
	Exception     TaggedObjectID
	CatchLocation Location
}

// EventThreadStart represents an event raised when a new thread is started.
type EventThreadStart struct {
	Request EventRequestID
	Thread  ThreadID
}

// EventThreadDeath represents an event raised when a thread is stopped.
type EventThreadDeath struct {
	Request EventRequestID
	Thread  ThreadID
}

// EventClassPrepare represents an event raised when a class enters the prepared state.
type EventClassPrepare struct {
	Request   EventRequestID
	Thread    ThreadID
	ClassKind TypeTag
	ClassType ReferenceTypeID
	Signature string
	Status    ClassStatus
}

// EventClassUnload represents an event raised when a class is unloaded.
type EventClassUnload struct {
	Request   EventRequestID
	Signature string
}

// EventFieldAccess represents an event raised when a field is accessed.
type EventFieldAccess struct {
	Request   EventRequestID
	Thread    ThreadID
	Location  Location
	FieldKind TypeTag
	FieldType ReferenceTypeID
	Field     FieldID
	Object    TaggedObjectID
}

// EventFieldModification represents an event raised when a field is modified.
type EventFieldModification struct {
	Request   EventRequestID
	Thread    ThreadID
	Location  Location
	FieldKind TypeTag
	FieldType ReferenceTypeID
	Field     FieldID
	Object    TaggedObjectID
	NewValue  Value
}

func (e EventVMStart) request() EventRequestID           { return e.Request }
func (e EventVMDeath) request() EventRequestID           { return e.Request }
func (e EventSingleStep) request() EventRequestID        { return e.Request }
func (e EventBreakpoint) request() EventRequestID        { return e.Request }
func (e EventMethodEntry) request() EventRequestID       { return e.Request }
func (e EventMethodExit) request() EventRequestID        { return e.Request }
func (e EventException) request() EventRequestID         { return e.Request }
func (e EventThreadStart) request() EventRequestID       { return e.Request }
func (e EventThreadDeath) request() EventRequestID       { return e.Request }
func (e EventClassPrepare) request() EventRequestID      { return e.Request }
func (e EventClassUnload) request() EventRequestID       { return e.Request }
func (e EventFieldAccess) request() EventRequestID       { return e.Request }
func (e EventFieldModification) request() EventRequestID { return e.Request }

// Kind returns VMStart
func (EventVMStart) Kind() EventKind { return VMStart }

// Kind returns VMDeath
func (EventVMDeath) Kind() EventKind { return VMDeath }

// Kind returns SingleStep
func (EventSingleStep) Kind() EventKind { return SingleStep }

// Kind returns Breakpoint
func (EventBreakpoint) Kind() EventKind { return Breakpoint }

// Kind returns MethodEntry
func (EventMethodEntry) Kind() EventKind { return MethodEntry }

// Kind returns MethodExit
func (EventMethodExit) Kind() EventKind { return MethodExit }

// Kind returns Exception
func (EventException) Kind() EventKind { return Exception }

// Kind returns ThreadStart
func (EventThreadStart) Kind() EventKind { return ThreadStart }

// Kind returns ThreadDeath
func (EventThreadDeath) Kind() EventKind { return ThreadDeath }

// Kind returns ClassPrepare
func (EventClassPrepare) Kind() EventKind { return ClassPrepare }

// Kind returns ClassUnload
func (EventClassUnload) Kind() EventKind { return ClassUnload }

// Kind returns FieldAccess
func (EventFieldAccess) Kind() EventKind { return FieldAccess }

// Kind returns FieldModification
func (EventFieldModification) Kind() EventKind { return FieldModification }
