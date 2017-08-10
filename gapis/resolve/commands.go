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

package resolve

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Commands resolves and returns the command list from the path p.
func Commands(ctx context.Context, p *path.Commands) (*service.Commands, error) {
	c, err := capture.ResolveFromPath(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	atomIdxFrom, atomIdxTo := p.From[0], p.To[0]
	if len(p.From) > 1 || len(p.To) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported for Commands") // TODO: Subcommands
	}
	count := uint64(len(c.Commands))
	if count == 0 {
		return nil, fmt.Errorf("No commands in capture")
	}
	atomIdxFrom = u64.Min(atomIdxFrom, count-1)
	atomIdxTo = u64.Min(atomIdxTo, count-1)
	if atomIdxFrom > atomIdxTo {
		atomIdxFrom, atomIdxTo = atomIdxTo, atomIdxFrom
	}
	count = atomIdxTo - atomIdxFrom
	paths := make([]*path.Command, count)
	for i := uint64(0); i < count; i++ {
		paths[i] = p.Capture.Command(i)
	}
	return &service.Commands{List: paths}, nil
}
