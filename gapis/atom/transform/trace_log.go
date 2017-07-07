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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

// Trace is an implementation of Transformer that records each atom id and atom
// value that passes through Trace to Logger. Atoms passing through Trace are
// written to the output Writer unaltered.
type Trace struct{}

func (t Trace) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) {
	log.I(ctx, "id: %v, atom: %v", id, cmd)
	out.MutateAndWrite(ctx, id, cmd)
}

func (t Trace) Flush(ctx context.Context, out Writer) {}
