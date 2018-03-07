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

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/event/task"
)

// EventRequestID is an identifier of an event request.
type EventRequestID int

const cmdCompositeEvent = cmdID(100)

// WatchEvents sets an event watcher, calling handler for each received event.
// WatchEvents will continue to watch for events until handler returns false or
// the context is cancelled.
func (c *Connection) WatchEvents(
	ctx context.Context,
	kind EventKind,
	suspendPolity SuspendPolicy,
	handler func(Event) bool,
	modifiers ...EventModifier) error {

	req := struct {
		Kind          EventKind
		SuspendPolicy SuspendPolicy
		Modifiers     []EventModifier
	}{
		Kind:          kind,
		SuspendPolicy: suspendPolity,
		Modifiers:     modifiers,
	}

	var id EventRequestID
	err := c.get(cmdEventRequestSet, req, &id)
	if err != nil {
		return err
	}

	events := make(chan Event, 8)
	c.Lock()
	c.events[id] = events
	c.Unlock()

	defer func() {
		c.Lock()
		delete(c.events, id)
		c.Unlock()
	}()

	if err := c.ResumeAll(); err != nil {
		return err
	}

run: // Consume events until the handler returns false or the context is cancelled.
	for {
		select {
		case event := <-events:
			if !handler(event) {
				break run
			}
		case <-task.ShouldStop(ctx):
			break run
		}
	}

	//  Clear the event.
	clear := struct {
		Kind EventKind
		ID   EventRequestID
	}{
		Kind: kind,
		ID:   id,
	}

	if err := c.get(cmdEventRequestClear, clear, nil); err != nil {
		return err
	}

flush: // Consume any remaining events in the pipe.
	for {
		select {
		case event := <-events:
			handler(event)
		default:
			break flush
		}
	}

	return nil
}

// EventModifier is the interface implemented by all event modifier types.
// These are filters on the events that are raised.
// See http://docs.oracle.com/javase/1.5.0/docs/guide/jpda/jdwp/jdwp-protocol.html#JDWP_EventRequest_Set
// for detailed descriptions and rules for each of the EventModifiers.
type EventModifier interface {
	modKind() uint8
}

// CountEventModifier is an EventModifier that limits the number of times an
// event is fired. For example, using a CountEventModifier of 2 will only let
// two events fire.
type CountEventModifier int

// ThreadOnlyEventModifier is an EventModifier that filters the events to those
// that are raised on the specified thread.
type ThreadOnlyEventModifier ThreadID

// ClassOnlyEventModifier is an EventModifier that filters the events to those
// that are associated with the specified class.
type ClassOnlyEventModifier ClassID

// ClassMatchEventModifier is an EventModifier that filters the events to those
// that are associated with class names that match the pattern. The pattern can
// be an exact class name match, for use a '*' wildcard at the start or end of
// the string. Examples:
// • "java.lang.String"
// • "*.String"
// • "java.lang.*"
type ClassMatchEventModifier string

// ClassExcludeEventModifier is an EventModifier that filters the events to
// those that are not associated with class names that match the pattern.
// See ClassMatchEventModifier for the permitted patterns.
type ClassExcludeEventModifier string

// LocationOnlyEventModifier is an EventModifier that filters the events to
// those that only originate at the specified location.
type LocationOnlyEventModifier Location

// ExceptionOnlyEventModifier is an EventModifier that filters exception events.
// Can only be used for exception events.
type ExceptionOnlyEventModifier struct {
	ExceptionOrNull ReferenceTypeID // If not nil, only permit exceptions of this type.
	Caught          bool            // Report caught exceptions
	Uncaught        bool            // Report uncaught exceptions
}

// FieldOnlyEventModifier is an EventModifier that filters events to those
// relating to the specified field.
// Can only be used for field access or field modified events.
type FieldOnlyEventModifier struct {
	Type  ReferenceTypeID
	Field FieldID
}

// StepEventModifier is an EventModifier that filters step events to those which
// satisfy depth and size constraints.
// Can only be used with step events.
type StepEventModifier struct {
	Thread ThreadID
	Size   int
	Depth  int
}

// InstanceOnlyEventModifier is an EventModifier that filters events to those
// which have the specified 'this' object.
type InstanceOnlyEventModifier ObjectID

func (CountEventModifier) modKind() uint8         { return 1 }
func (ThreadOnlyEventModifier) modKind() uint8    { return 3 }
func (ClassOnlyEventModifier) modKind() uint8     { return 4 }
func (ClassMatchEventModifier) modKind() uint8    { return 5 }
func (ClassExcludeEventModifier) modKind() uint8  { return 6 }
func (LocationOnlyEventModifier) modKind() uint8  { return 7 }
func (ExceptionOnlyEventModifier) modKind() uint8 { return 8 }
func (FieldOnlyEventModifier) modKind() uint8     { return 9 }
func (StepEventModifier) modKind() uint8          { return 10 }
func (InstanceOnlyEventModifier) modKind() uint8  { return 11 }

func (m CountEventModifier) String() string {
	return fmt.Sprintf("CountEventModifier<%v>", int(m))
}
func (m ThreadOnlyEventModifier) String() string {
	return fmt.Sprintf("ThreadOnlyEventModifier<%v>", int(m))
}
func (m ClassOnlyEventModifier) String() string {
	return fmt.Sprintf("ClassOnlyEventModifier<%v>", int(m))
}
func (m ClassMatchEventModifier) String() string {
	return fmt.Sprintf("ClassMatchEventModifier<%v>", string(m))
}
func (m ClassExcludeEventModifier) String() string {
	return fmt.Sprintf("ClassExcludeEventModifier<%v>", string(m))
}
func (m LocationOnlyEventModifier) String() string {
	return fmt.Sprintf("LocationOnlyEventModifier<%v>", Location(m))
}
func (m ExceptionOnlyEventModifier) String() string {
	return fmt.Sprintf("ExceptionOnlyEventModifier<Exception: %v, Caught: %v, Uncaught: %v>",
		m.ExceptionOrNull, m.Caught, m.Uncaught)
}
func (m FieldOnlyEventModifier) String() string {
	return fmt.Sprintf("FieldOnlyEventModifier<Type: %v, Field: %v>", m.Type, m.Field)
}
func (m StepEventModifier) String() string {
	return fmt.Sprintf("StepEventModifier<Thread: %v, Size: %v, Depth: %v>",
		m.Thread, m.Size, m.Depth)
}
func (m InstanceOnlyEventModifier) String() string {
	return fmt.Sprintf("InstanceOnlyEventModifier<%v>", ObjectID(m))
}
