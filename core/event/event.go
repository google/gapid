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

package event

import "context"

// Handler is the type for functions to which events can be delivered.
type Handler func(ctx context.Context, event interface{}) error

// Producer is the type for a function that generates events.
type Producer func(ctx context.Context) interface{}

// Predicate is the signature for a function that tests an event for a boolean property.
type Predicate func(ctx context.Context, event interface{}) bool

// Listener is the signature for a function that accepts handlers to send events to.
type Listener func(ctx context.Context, handler Handler)

// Source is the type for a closable event producer.
// This allows a consumer to indicate that they no longer need the source.
type Source interface {
	// Next is a method that matches the Producer signature.
	// It will respond with the events in stream order.
	Next(ctx context.Context) interface{}
	// Close can be used to notify the stream that no more events are desired.
	Close(ctx context.Context)
}
