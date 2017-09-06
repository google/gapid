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

package api

import (
	"context"
	"reflect"

	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/gapis/replay/builder"
)

// Cmd is the interface implemented by all graphics API commands.
type Cmd interface {
	// All commands belong to an API
	APIObject

	// Caller returns the identifier of the command that called this command,
	// or CmdNoID if the command has no caller.
	Caller() CmdID

	// SetCaller sets the identifier of the command that called this command.
	SetCaller(CmdID)

	// Thread returns the thread index this command was executed on.
	Thread() uint64

	// SetThread changes the thread index.
	SetThread(uint64)

	// CmdName returns the name of the command.
	CmdName() string

	// CmdFlags returns the flags of the command.
	CmdFlags(context.Context, CmdID, *GlobalState) CmdFlags

	// Extras returns all the Extras associated with the dynamic command.
	Extras() *CmdExtras

	// Mutate mutates the State using the command. If the builder argument is
	// not nil then it will call the replay function on the builder.
	Mutate(context.Context, CmdID, *GlobalState, *builder.Builder) error
}

const (
	// ErrParameterNotFound is the error returned by GetParameter() and
	// SetParameter() when the command does not have the named parameter.
	ErrParameterNotFound = fault.Const("Parameter not found")
	// ErrResultNotFound is the error returned by GetResult() and SetResult()
	// when the command does not have a result value.
	ErrResultNotFound = fault.Const("Result not found")

	paramTag    = "param"
	resultTag   = "result"
	constsetTag = "constset"
)

// GetParameter returns the parameter value with the specified name.
func GetParameter(ctx context.Context, c Cmd, name string) (interface{}, error) {
	v := reflect.ValueOf(c)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if n, ok := t.Tag.Lookup(paramTag); ok {
			if name == n {
				return f.Interface(), nil
			}
		}
	}
	return nil, ErrParameterNotFound
}

// SetParameter sets the parameter with the specified name with val.
func SetParameter(ctx context.Context, c Cmd, name string, val interface{}) error {
	v := reflect.ValueOf(c)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if n, ok := t.Tag.Lookup(paramTag); ok {
			if name == n {
				return deep.Copy(f.Addr().Interface(), val)
			}
		}
	}
	return ErrParameterNotFound
}

// GetResult returns the command's result value.
func GetResult(ctx context.Context, c Cmd) (interface{}, error) {
	v := reflect.ValueOf(c)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if _, ok := t.Tag.Lookup(resultTag); ok {
			return f.Interface(), nil
		}
	}
	return nil, ErrResultNotFound
}

// SetResult sets the commands result value to val.
func SetResult(ctx context.Context, c Cmd, val interface{}) error {
	v := reflect.ValueOf(c)
	for v.Kind() != reflect.Struct {
		v = v.Elem()
	}
	t := v.Type()
	for i, count := 0, t.NumField(); i < count; i++ {
		f, t := v.Field(i), t.Field(i)
		if _, ok := t.Tag.Lookup(resultTag); ok {
			return deep.Copy(f.Addr().Interface(), val)
		}
	}
	return ErrResultNotFound
}
