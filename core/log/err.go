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
	"fmt"
)

// Err creates a new error that wraps cause with the current logging
// information.
func (l *Logger) Err(cause error, msg string) error {
	return &err{cause, l.Message(Error, false, msg)}
}

// Errf creates a new error that wraps cause with the current logging
// information.
func (l *Logger) Errf(cause error, fmt string, args ...interface{}) error {
	return &err{cause, l.Messagef(Error, false, fmt, args...)}
}

type err struct {
	cause error
	msg   *Message
}

func (e err) Cause() error {
	return e.cause
}

func (e err) Error() string {
	if e.cause == nil {
		return e.msg.Text
	}
	return fmt.Sprintf("%v\n   Cause: %v", e.msg.Text, e.cause)
}

// Err creates a new error that wraps cause with the current logging
// information.
func Err(ctx context.Context, cause error, msg string) error {
	return From(ctx).Err(cause, msg)
}

// Errf creates a new error that wraps cause with the current logging
// information.
func Errf(ctx context.Context, cause error, fmt string, args ...interface{}) error {
	return From(ctx).Errf(cause, fmt, args...)
}
