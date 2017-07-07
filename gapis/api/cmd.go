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

	// CmdFlags returns the flags of the command.
	CmdFlags() CmdFlags

	// Extras returns all the Extras associated with the dynamic command.
	Extras() *CmdExtras

	// Mutate mutates the State using the command. If the builder argument is
	// not nil then it will call the replay function on the builder.
	Mutate(context.Context, *State, *builder.Builder) error
}
