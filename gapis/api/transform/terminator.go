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

// Terminator is an Transformer that prevents commands passing-through it after
// a certain point in the stream.
type Terminator interface {
	Transformer

	// Add relaxes the termination limit to pass-through all commands before and
	// including the command or subcommand.
	Add(context.Context, api.CmdID, api.SubCmdIdx) error
}
