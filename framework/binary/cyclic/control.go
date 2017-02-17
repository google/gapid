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

package cyclic

import (
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
)

// Control represents a control block in a stream.
type Control struct {
	Mode binary.Mode
}

const (
	ControlVersion           uint32 = 0
	ErrInvalidControlVersion        = fault.Const("Invalid control block version")
)

func (c *Control) write(e *encoder) {
	e.Uint32(ControlVersion)
	e.Uint32(uint32(c.Mode))
}

func (c *Control) read(d *decoder) {
	version := d.Uint32()
	switch version {
	case 0:
		c.Mode = binary.Mode(d.Uint32())
	default:
		ctx := log.TODO()
		d.SetError(cause.Wrap(ctx, ErrInvalidControlVersion).With("version", version))
	}
}
