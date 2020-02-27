// Copyright (C) 2019 Google Inc.
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

package client

import (
	"context"

	"github.com/google/gapid/core/event/task"
)

// BindSync is a sync helper to turn async Bind calls into sync ones. Use the
// Handler struct member in calling Bind() and then call Wait.
type BindSync struct {
	Handler BindHandler
	methods map[string]*Method
	err     error
	wait    task.Signal
}

// NewBindSync returns a new BindSync.
func NewBindSync(ctx context.Context) *BindSync {
	wait, fire := task.NewSignal()
	s := &BindSync{wait: wait}
	s.Handler = func(methods map[string]*Method, err error) {
		s.methods = methods
		s.err = err
		fire(ctx)
	}
	return s
}

// Wait waits on the sync and returns the result of the Bind call.
func (s *BindSync) Wait(ctx context.Context) (map[string]*Method, error) {
	if !s.wait.Wait(ctx) {
		return nil, task.StopReason(ctx)
	}
	return s.methods, s.err
}

// InvokeSync is a sync helper to turn async Invoke calls into sync ones. Use
// the Handler struct member in calling Invoke() and then call Wait.
type InvokeSync struct {
	Handler InvokeHandler
	err     error
	wait    task.Signal
}

// NewInvokeSync returns a new InvokeSync. The given callback is called upon
// every successful streamed result.
func NewInvokeSync(ctx context.Context, cb func(data []byte) error) *InvokeSync {
	wait, fire := task.NewSignal()
	s := &InvokeSync{wait: wait}
	s.Handler = func(data []byte, more bool, err error) {
		if err != nil {
			s.err = err
		} else {
			s.err = cb(data)
		}

		if !more || s.err != nil {
			fire(ctx)
		}
	}
	return s
}

// Wait waits on the sync and returns the error the Invoke call.
func (s *InvokeSync) Wait(ctx context.Context) error {
	if !s.wait.Wait(ctx) {
		return task.StopReason(ctx)
	}
	return s.err
}
