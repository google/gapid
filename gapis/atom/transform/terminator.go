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
	"github.com/google/gapid/gapis/atom"
)

// Terminator is an interface that rewrites a set of atoms to only
// mutate the given commands
type Terminator interface {
	// Adds the given atom, and subcommand as the last atom that
	// must be observed
	Add(context.Context, atom.ID, []uint64) error
	// The transformer interface
	Transform(context.Context, atom.ID, api.Cmd, Writer)
	Flush(context.Context, Writer)
}
