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

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/gapis/replay/builder"
)

// Cmd is the interface implemented by all graphics API commands.
type Cmd interface {
	// All commands belong to an API
	APIObject

	// Thread returns the thread index this command was executed on.
	Thread() uint64

	// SetThread changes the thread index.
	SetThread(uint64)

	// CmdName returns the name of the command.
	CmdName() string

	// CmdParams returns the command's parameters.
	CmdParams() Properties

	// CmdResult returns the command's result value, or nil if there is no
	// result value.
	CmdResult() *Property

	// CmdFlags returns the flags of the command.
	CmdFlags() CmdFlags

	// Extras returns all the Extras associated with the command.
	Extras() *CmdExtras

	// Mutate mutates the State using the command. If the builder argument is
	// not nil then it will call the replay function on the builder.
	Mutate(context.Context, CmdID, *GlobalState, *builder.Builder, StateWatcher) error

	// Clone makes a shallow copy of this command.
	Clone() Cmd

	// Alive returns true if this command should be marked alive for DCE
	Alive() bool

	// Terminated returns true if this command did terminate during capture
	Terminated() bool

	// SetTerminated sets whether this command has terminated or not
	SetTerminated(terminated bool)
}

const (
	// ErrParameterNotFound is the error returned by GetParameter() and
	// SetParameter() when the command does not have the named parameter.
	ErrParameterNotFound = fault.Const("Parameter not found")
	// ErrResultNotFound is the error returned by GetResult() and SetResult()
	// when the command does not have a result value.
	ErrResultNotFound = fault.Const("Result not found")
)

// GetParameter returns the parameter value with the specified name.
func GetParameter(c Cmd, name string) (interface{}, error) {
	if p := c.CmdParams().Find(name); p != nil {
		return p.Get(), nil
	}
	return nil, ErrParameterNotFound
}

// SetParameter sets the parameter with the specified name with val.
func SetParameter(c Cmd, name string, val interface{}) error {
	if p := c.CmdParams().Find(name); p != nil {
		p.Set(val)
		return nil
	}
	return ErrParameterNotFound
}

// GetResult returns the command's result value.
func GetResult(c Cmd) (interface{}, error) {
	if p := c.CmdResult(); p != nil {
		return p.Get(), nil
	}
	return nil, ErrResultNotFound
}

// SetResult sets the commands result value to val.
func SetResult(c Cmd, val interface{}) error {
	if p := c.CmdResult(); p != nil {
		p.Set(val)
		return nil
	}
	return ErrResultNotFound
}
