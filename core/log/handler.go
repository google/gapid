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

package log

import (
	"context"

	"github.com/google/gapid/core/context/keys"
)

// Handler is the handler of log messages.
type Handler interface {
	Handle(*Message)
	Close()
}

// handler is a simple implementation of the Handler interface
type handler struct {
	handle func(*Message)
	close  func()
}

func (h handler) Handle(m *Message) { h.handle(m) }
func (h handler) Close() {
	if h.close != nil {
		h.close()
	}
}

// NewHandler returns a Handler that calls handle for each message and close
// when the handler is closed. close can be nil.
func NewHandler(handle func(*Message), close func()) Handler {
	return &handler{handle, close}
}

type handlerKeyTy string

const handlerKey handlerKeyTy = "log.handlerKey"

// PutHandler returns a new context with the Handler assigned to w.
func PutHandler(ctx context.Context, w Handler) context.Context {
	return keys.WithValue(ctx, handlerKey, w)
}

// GetHandler returns the Handler assigned to ctx.
func GetHandler(ctx context.Context) Handler {
	out, _ := ctx.Value(handlerKey).(Handler)
	return out
}
