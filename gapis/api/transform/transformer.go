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

package transform

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

// Transformer is the interface that wraps the basic Transform method.
type Transformer interface {
	// Transform takes a given command and identifier and Writes out a possibly
	// transformed set of commands to the output.
	// Transform must not modify cmd in any way.
	Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, output Writer) error
	// Flush is called at the end of a command stream to cause Transformers
	// that cache commands to send any they have stored into the output.
	Flush(ctx context.Context, output Writer) error
	// Preloop is called at the beginning of the loop if the trace is going to
	// be looped.
	PreLoop(ctx context.Context, output Writer)
	// PostLoop is called at the end of the loop if the trace is going to
	// be looped.
	PostLoop(ctx context.Context, output Writer)

	// BuffersCommands returns true if the transformer is going to buffer
	// commands, return false otherwise.
	BuffersCommands() bool
}

// Writer is the interface which consumes the output of an Transformer.
// It also keeps track of state changes caused by all commands written to it.
// Conceptually, each Writer object contains its own separate State object,
// which is modified when MutateAndWrite is called.
// This allows the transform to access the state both before and after the
// mutation of state happens. It is also possible to omit/insert commands.
// In practice, single state object can be shared by all transforms for
// performance (i.e. the mutation is done only once at the very end).
// This potentially allows state changes to leak upstream so care is needed.
// There is a configuration flag to switch between the shared/separate modes.
type Writer interface {
	// State returns the state object associated with this writer.
	State() *api.GlobalState
	// MutateAndWrite mutates the state object associated with this writer,
	// and it passes the command to further consumers.
	MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error
	//Notify next transformer it's ready to start loop the trace.
	NotifyPreLoop(ctx context.Context)
	//Notify next transformer it's the end of the loop.
	NotifyPostLoop(ctx context.Context)
}
