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

package atom

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/replay/builder"
)

// Resource is a Cmd that embeds a blob of memory into the command stream. These
// commands are typically only used for .gfxtrace files as they are stripped
// from the stream on import and their resources are placed into the database.
type Resource struct {
	ID   id.ID  // The resource identifier holding the memory that was observed.
	Data []byte // The resource data
}

func (a *Resource) String() string {
	return fmt.Sprintf("ID: %s - 0x%x bytes", a.ID, len(a.Data))
}

func (Resource) Thread() uint64         { return 0 }
func (Resource) SetThread(uint64)       {}
func (Resource) CmdName() string        { return "<Resource>" }
func (Resource) API() api.API           { return nil }
func (Resource) CmdFlags() api.CmdFlags { return 0 }
func (Resource) Extras() *api.CmdExtras { return nil }
func (Resource) Mutate(ctx context.Context, s *api.State, b *builder.Builder) error {
	return nil
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a *Resource) (*atom_pb.Resource, error) {
			return &atom_pb.Resource{Id: a.ID.String(), Data: a.Data}, nil
		},
		func(ctx context.Context, a *atom_pb.Resource) (*Resource, error) {
			r := Resource{}
			r.ID.Parse(a.Id)
			r.Data = a.Data
			return &r, nil
		},
	)
}
