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
	"reflect"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/context/memo"
)

// Value returns the value stored against key in this context.
// See context.Context.Value for more details.
func (ctx logContext) Value(key interface{}) interface{} {
	return ctx.internal.Value(key)
}

// WithValue returns a new context with the additional key value pair specified.
// If the logger already has the specified value, it will be overwritten.
func (ctx logContext) WithValue(key string, value interface{}) Context {
	return Wrap(keys.WithValue(ctx.Unwrap(), memo.DetailPair(key), value))
}

// WithValue returns a new logger with the additional key value pair specified.
// If the logger already has the specified value, it will be overwritten.
// If the logger is not active, the function is very cheap to call.
func (l Logger) WithValue(key string, value interface{}) Logger {
	return Logger{l.J.With(memo.DetailPair(key), value)}
}

// V is shorthand for ctx.WithValue(key, value)
func (ctx logContext) V(key string, value interface{}) Context {
	return ctx.WithValue(key, value)
}

// V is shorthand for l.WithValue(key, value)
func (l Logger) V(key string, value interface{}) Logger {
	return l.WithValue(key, value)
}

// S is shorthand for ctx.WithValue(key, value), but is only for string.
func (ctx logContext) S(key string, value string) Context {
	return ctx.WithValue(key, value)
}

// S does the same as WithValue(key, value), but is only for strings and does not cause a boxing allocation if the
// logger is not active.
func (l Logger) S(key string, value string) Logger {
	return l.WithValue(key, value)
}

// I is shorthand for ctx.WithValue(key, value) but is only for int.
func (ctx logContext) I(key string, value int) Context {
	return ctx.WithValue(key, value)
}

// I does the same as WithValue(key, value) but is only for int and does not cause a boxing allocation if the
// logger is not active.
func (l Logger) I(key string, value int) Logger {
	return l.WithValue(key, value)
}

// F is shorthand for ctx.WithValue(key, value) but is only for float64.
func (ctx logContext) F(key string, value float64) Context {
	return ctx.WithValue(key, value)
}

// F does the same as WithValue(key, value) but is only for float64 and does not cause a boxing allocation if the
// logger is not active.
func (l Logger) F(key string, value float64) Logger {
	return l.WithValue(key, value)
}

// T is shorthand for ctx.WithValue(key, reflect.TypeOf(value)).
func (ctx logContext) T(key string, value interface{}) Context {
	return ctx.WithValue(key, reflect.TypeOf(value))
}

// T is shorthand for l.WithValue(key, reflect.TypeOf(value))
func (l Logger) T(key string, value interface{}) Logger {
	return l.WithValue(key, reflect.TypeOf(value))
}
