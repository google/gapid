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
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// Resource is an Atom that embeds a blob of memory into the atom stream. These
// atoms are typically only used for .gfxtrace files as they are stripped from
// the stream on import and their resources are placed into the database.
type Resource struct {
	binary.Generate
	ID   id.ID  // The resource identifier holding the memory that was observed.
	Data []byte // The resource data
}

func (a *Resource) String() string {
	return fmt.Sprintf("ID: %s - 0x%x bytes", a.ID, len(a.Data))
}

func (a *Resource) API() gfxapi.API  { return nil }
func (a *Resource) AtomFlags() Flags { return 0 }
func (a *Resource) Extras() *Extras  { return nil }
func (a *Resource) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	return nil
}
func (a *Resource) Convert(ctx log.Context, out atom_pb.Handler) error {
	return out(ctx, &atom_pb.Resource{
		Id:   a.ID.String(),
		Data: a.Data,
	})
}
func ResourceFrom(from *atom_pb.Resource) Resource {
	r := Resource{}
	r.ID.Parse(from.Id)
	r.Data = from.Data
	return r
}
