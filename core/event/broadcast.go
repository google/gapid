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

import (
	"context"
	"io"
)

// Broadcast implements a list of handlers that itself is a handler.
// Events written to the broadcast are sent to all handlers in the list
type Broadcast []Handler

// Listen adds a new handler to the set.
// It conforms to the Listener signature.
func (b *Broadcast) Listen(ctx context.Context, h Handler) {
	*b = append(*b, h)
}

// Send implements Handler to send (sequentially) to all current handlers.
// Handlers that return an error will be dropped from the broadcast list.
func (b *Broadcast) Send(ctx context.Context, event interface{}) error {
	if len(*b) == 0 {
		return nil
	}
	out := 0
	var result error
	for _, h := range *b {
		res := h(ctx, event)
		switch res {
		case nil:
			(*b)[out] = h
			out++
		case io.EOF:
			if result == nil {
				result = res
			}
		default:
			// TODO: build a compound error if we already have one?
			if result == nil || result == io.EOF {
				result = res
			}
		}
	}
	*b = (*b)[:out]
	return result
}
