// Copyright (C) 2018 Google Inc.
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

package testutils

import (
	"bytes"
	"context"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

// Cmd is a custom implementation of the api.Cmd interface that simplifies
// testing compiler generated commands.
type Cmd struct {
	N string  // Command name
	D []byte  // Encoded command used by the compiler generated execute function
	E *Extras // Command extras
	T uint64  // Command thread
}

var _ api.Cmd = &Cmd{}

// API stubs the api.Cmd interface.
func (c *Cmd) API() api.API { return nil }

// Thread stubs the api.Cmd interface.
func (c *Cmd) Thread() uint64 { return c.T }

// SetThread stubs the api.Cmd interface.
func (c *Cmd) SetThread(thread uint64) { c.T = thread }

// CmdName stubs the api.Cmd interface.
func (c *Cmd) CmdName() string { return c.N }

// CmdParams stubs the api.Cmd interface.
func (c *Cmd) CmdParams() api.Properties { return nil }

// CmdResult stubs the api.Cmd interface.
func (c *Cmd) CmdResult() *api.Property { return nil }

// CmdFlags stubs the api.Cmd interface.
func (c *Cmd) CmdFlags() api.CmdFlags { return 0 }

// Extras stubs the api.Cmd interface.
func (c *Cmd) Extras() *api.CmdExtras {
	if c.E == nil {
		return &api.CmdExtras{}
	}
	return c.E.e
}

// Mutate stubs the api.Cmd interface.
func (c *Cmd) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder, api.StateWatcher) error {
	return nil
}

// Terminated stubs the api.Cmd interface.
func (c *Cmd) Terminated() bool {
	return true
}

// SetTerminated stubs the api.Cmd interface.
func (c *Cmd) SetTerminated(terminated bool) {
	return
}

// Encode implements the executor.Encodable interface to encode the command to
// a buffer used by the compiler generated execute function.
func (c Cmd) Encode(out []byte) bool {
	w := endian.Writer(bytes.NewBuffer(out), device.LittleEndian)
	w.Uint64(c.T)
	copy(out[8:], c.D)
	return true
}

// Clone makes a shallow copy of this command.
func (c *Cmd) Clone() api.Cmd {
	clone := *c
	return &clone
}

// Alive returns true if this command should be marked alive for DCE
func (c *Cmd) Alive() bool {
	return false
}

// Extras is a helper wrapper around an api.CmdExtras has helpers methods for
// adding read and writ observations.
type Extras struct {
	e *api.CmdExtras
}

// R adds a read using the given range and data, returning this Extras so calls
// can be chained.
func (e *Extras) R(base uint64, size uint64, id id.ID) *Extras {
	if e.e == nil {
		e.e = &api.CmdExtras{}
	}
	e.e.GetOrAppendObservations().AddRead(memory.Range{Base: base, Size: size}, id)
	return e
}

// W adds a write using the given range and data, returning this Extras so calls
// can be chained.
func (e *Extras) W(base uint64, size uint64, id id.ID) *Extras {
	if e.e == nil {
		e.e = &api.CmdExtras{}
	}
	e.e.GetOrAppendObservations().AddWrite(memory.Range{Base: base, Size: size}, id)
	return e
}

// R creates and returns a new Extras containing a single read using the given
// range and data.
func R(base uint64, size uint64, id id.ID) *Extras {
	e := &Extras{}
	return e.R(base, size, id)
}

// W creates and returns a new Extras containing a single write using the given
// range and data.
func W(base uint64, size uint64, id id.ID) *Extras {
	e := &Extras{}
	return e.W(base, size, id)
}
