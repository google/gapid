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

package dependencygraph

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
)

var footprintBuildCounter = benchmark.Duration("footprint.build")

// Footprint contains a list of command and a list of Behaviors which
// describes the side effect of executing the commands in that list.
type Footprint struct {
	Commands         []api.Cmd
	Behaviors        []*Behavior
	BehaviorIndices  map[*Behavior]uint64
	cmdIdxToBehavior api.SubCmdIdxTrie
}

// NewEmptyFootprint creates a new Footprint with an empty command list, and
// returns a pointer to that Footprint.
func NewEmptyFootprint(ctx context.Context) *Footprint {
	return &Footprint{
		Commands:         []api.Cmd{},
		Behaviors:        []*Behavior{},
		BehaviorIndices:  map[*Behavior]uint64{},
		cmdIdxToBehavior: api.SubCmdIdxTrie{},
	}
}

// NewFootprint takes a list of commands and creates a new Footprint with
// that list of commands, and returns a pointer to that Footprint.
func NewFootprint(ctx context.Context, cmds []api.Cmd) *Footprint {
	return &Footprint{
		Commands:         cmds,
		Behaviors:        make([]*Behavior, 0, len(cmds)),
		BehaviorIndices:  map[*Behavior]uint64{},
		cmdIdxToBehavior: api.SubCmdIdxTrie{},
	}
}

// Behavior contains a set of read and write operations as side effect of
// executing the command to whom it belongs. Behavior also contains a
// reference to the back-propagation machine which should be used to process
// the Behavior to determine its liveness for dead code elimination.
type Behavior struct {
	Reads   []DefUseVariable
	Writes  []DefUseVariable
	Owner   api.SubCmdIdx
	Alive   bool
	Aborted bool
	Machine BackPropagationMachine
}

// NewBehavior creates a new Behavior which belongs to the command indexed by
// the given SubCmdIdx and shall be process by the given back-propagation
// machine. Returns a pointer to the created Behavior.
func NewBehavior(fullCommandIndex api.SubCmdIdx, m BackPropagationMachine) *Behavior {
	return &Behavior{
		Owner:   fullCommandIndex,
		Machine: m,
	}
}

// Read records a read operation of the given DefUseVariable to the Behavior
func (b *Behavior) Read(c DefUseVariable) {
	b.Reads = append(b.Reads, c)
}

// Write records a write operation of the given DefUseVariable to the Behavior
func (b *Behavior) Write(c DefUseVariable) {
	b.Writes = append(b.Writes, c)
}

// Modify records a read and a write operation of the given DefUseVariable to the
// Behavior
func (b *Behavior) Modify(c DefUseVariable) {
	b.Read(c)
	b.Write(c)
}

// DefUseVariable is a tag to data that should be considered as the logical
// representation of a variable in
// liveness analysis(https://en.wikipedia.org/wiki/Live_variable_analysis).
// All sorts data to be tracked in the
// def-use chain(https://en.wikipedia.org/wiki/Use-define_chain), which is to
// be used for the liveness analysis, should be tagged as DefUseVariable.
// In the context of GAPID, any pieces of the whole API state can be tagged as
// DefUseVariable, e.g. a piece of memory, a handle, an object state, etc.
// But eventually those DefUseVariables wiil be handed in the
// BackPropagationMachine, which is referred in the Behavior that records
// read or write operations of the DefUseVariables. So the corresponding
// BackPropagationMachine must be able to handle the concrete type of those
// DefUseVariables.
type DefUseVariable interface {
	DefUseVariable()
}

// BackPropagationMachine determines the liveness of Behaviors along the
// back propagation of the operations recorded in Footprint's Behaviors.
type BackPropagationMachine interface {
	// IsAlive checks whether a given behavior, which is specified by its index
	// and the footprint to which it belongs, should be kept alive. It should
	// takes the written DefUseVariable of the given behavior, check every of
	// them with its internal state to determine the liveness of the Behavior.
	// Returns true if the behavior should be kept alive, otherwise returns
	// false.
	IsAlive(behaviorIndex uint64, ft *Footprint) bool
	// RecordBehaviorEffects records the read/write operations of the given
	// behavior specified by its index in the given footprint, and returns the
	// indices of the alive behaviors that should be kept alive due to recording
	// of the given behavior.
	RecordBehaviorEffects(behaviorIndex uint64, ft *Footprint) []uint64
	// Clear clears the internal state of the BackPropagationMachine.
	Clear()
	// FramebufferRequest marks the behavior indexed by the given index as where
	// framebuffer will be requested, so a 'use' will be added for each image
	// attachment.
	FramebufferRequest(behaviorIndex uint64, ft *Footprint)
}

// dummyMachine does nothing but marks all the incoming Behaviors as alive.
type dummyMachine struct{}

func (m *dummyMachine) IsAlive(uint64, *Footprint) bool { return true }
func (m *dummyMachine) RecordBehaviorEffects(behaviorIndex uint64,
	ft *Footprint) []uint64 {
	return []uint64{behaviorIndex}
}

func (m *dummyMachine) Clear()                                {}
func (m *dummyMachine) FramebufferRequest(uint64, *Footprint) {}

// BehaviorIndex returns the index of the last Behavior in the Footprint
// which belongs to the command or subcomand indexed by the given SubCmdIdx. In
// case the SubCmdIdx is invalid or a valid Behavior index is not found, error
// will be logged and uint64(0) will be returned.
func (f *Footprint) BehaviorIndex(ctx context.Context,
	fci api.SubCmdIdx) uint64 {
	v := f.cmdIdxToBehavior.Value(fci)
	if v != nil {
		if u, ok := v.(uint64); ok {
			return u
		}
		log.E(ctx, "Invalid behavior index: %v is not a uint64. Request command index: %v", v, fci)
		return uint64(0)
	}
	log.E(ctx, "Cannot get behavior index for command indexed with: %v", fci)
	return uint64(0)
}

// AddBehavior adds the given Behavior to the Footprint and updates the
// internal mapping from SubCmdIdx to the last Behavior that belongs to that
// command or subcommand.
func (f *Footprint) AddBehavior(ctx context.Context, b *Behavior) bool {
	bi := uint64(len(f.Behaviors))
	fci := b.Owner
	f.cmdIdxToBehavior.SetValue(fci, bi)
	f.Behaviors = append(f.Behaviors, b)
	f.BehaviorIndices[b] = bi
	return true
}

// FootprintBuilderProvider provides FootprintBuilder
type FootprintBuilderProvider interface {
	FootprintBuilder(context.Context) FootprintBuilder
}

// FootprintBuilder incrementally builds Footprint one command by one command.
type FootprintBuilder interface {
	BuildFootprint(context.Context, *api.GlobalState, *Footprint, api.CmdID, api.Cmd)
}

// GetFootprint returns a pointer to the resolved Footprint.
func GetFootprint(ctx context.Context) (*Footprint, error) {
	r, err := database.Build(ctx, &FootprintResolvable{Capture: capture.Get(ctx)})
	if err != nil {
		return nil, fmt.Errorf("Counld not get execution foot print: %v", err)
	}
	return r.(*Footprint), nil
}

// Resolve implements the database.Resolver interface.
func (r *FootprintResolvable) Resolve(ctx context.Context) (interface{}, error) {
	c, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	cmds := c.Commands
	builders := map[api.API]FootprintBuilder{}

	ft := NewFootprint(ctx, cmds)

	s := c.NewState()
	t0 := footprintBuildCounter.Start()
	defer footprintBuildCounter.Stop(t0)
	api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		a := cmd.API()
		if _, ok := builders[cmd.API()]; !ok {
			if bp, ok := cmd.API().(FootprintBuilderProvider); ok {
				builders[cmd.API()] = bp.FootprintBuilder(ctx)
			} else {
				// API does not provide execution footprint info, always keep commands
				// from such APIs alive.
				bh := NewBehavior(api.SubCmdIdx{uint64(id)}, &dummyMachine{})
				bh.Alive = true
				// Even if the command does not belong to an API that provides
				// execution footprint info, we still need to mutate it in the new
				// state, because following commands in other APIs may depends on the
				// side effect of the this command.
				if err := cmd.Mutate(ctx, id, s, nil); err != nil {
					bh.Aborted = true
					// Continue the footprint building even if errors are found. It is
					// following mutate calls, which are to build the replay
					// instructions, that are responsible to catch the error.
					// TODO: This error should be moved to report view.
				}
				ft.AddBehavior(ctx, bh)
				return nil
			}
		}
		builders[a].BuildFootprint(ctx, s, ft, id, cmd)
		return nil
	})
	return ft, nil
}
