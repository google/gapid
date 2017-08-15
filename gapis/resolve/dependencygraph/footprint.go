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

var footprintBuildCounter = benchmark.GlobalCounters.Duration("footprint.build")

// Footprint contains a list of command and a list of Behaviours which
// describes the side effect of executing the commands in that list.
type Footprint struct {
	Commands                                []api.Cmd
	Behaviours                              []*Behaviour
	commandIndexToBehaviourIndexLookupTable *api.SubCmdIdxTrie
	BehaviourIndices                        map[*Behaviour]uint64
}

// NewEmptyFootprint creates a new Footprint with an empty command list, and
// returns a pointer to that Footprint.
func NewEmptyFootprint(ctx context.Context) *Footprint {
	return &Footprint{
		Commands:                                []api.Cmd{},
		Behaviours:                              []*Behaviour{},
		commandIndexToBehaviourIndexLookupTable: &api.SubCmdIdxTrie{},
		BehaviourIndices:                        map[*Behaviour]uint64{},
	}
}

// NewFootprint takes a list of commands and creates a new Footprint with
// that list of commands, and returns a pointer to that Footprint.
func NewFootprint(ctx context.Context, cmds []api.Cmd) *Footprint {
	return &Footprint{
		Commands:                                cmds,
		Behaviours:                              make([]*Behaviour, 0, len(cmds)),
		commandIndexToBehaviourIndexLookupTable: &api.SubCmdIdxTrie{},
		BehaviourIndices:                        map[*Behaviour]uint64{},
	}
}

// Behaviour contains a set of read and write operations as side effect of
// executing the command to whom it belongs. Behaviour also contains a refernce
// to the back-propagation machine which should be used to process the
// Behaviour to determine its liveness for dead code elimination.
type Behaviour struct {
	Reads    []DefUseAtom
	Writes   []DefUseAtom
	BelongTo api.SubCmdIdx
	Alive    bool
	Aborted  bool
	Machine  BackPropagationMachine
}

// NewBehaviour creates a new Behaviour which belongs to the command indexed by
// the given SubCmdIdx and shall be process by the given back-propagation
// machine. Returns a pointer to the created Behaviour.
func NewBehaviour(fullCommandIndex api.SubCmdIdx, m BackPropagationMachine) *Behaviour {
	return &Behaviour{
		BelongTo: fullCommandIndex,
		Machine:  m,
	}
}

// Read records a read operation of the given DefUseAtom to the Behaviour
func (b *Behaviour) Read(c DefUseAtom) {
	b.Reads = append(b.Reads, c)
}

// Write records a write operation of the given DefUseAtom to the Behaviour
func (b *Behaviour) Write(c DefUseAtom) {
	b.Writes = append(b.Writes, c)
}

// Modify records a read and a write operation of the given DefUseAtom to the
// Behaviour
func (b *Behaviour) Modify(c DefUseAtom) {
	b.Read(c)
	b.Write(c)
}

// DefUseAtom represents a piece of state which can be altered as commands
// being executed.
type DefUseAtom interface {
	DefUseAtom()
}

// BackPropagationMachine determines the liveness of Behaviours along the
// back propagation of the operations recorded in Footprint's Behaviours.
type BackPropagationMachine interface {
	// IsAlive check whether a given behaviour, which is specified by its index
	// and the footprint to which it belongs, should be kept alive. Returns true
	// if the behaviour should be kept alive, otherwise returns false.
	IsAlive(behaviourIndex uint64, ft *Footprint) bool
	// RecordBehaviourEffects records the read/write operations of the given
	// behaviour specified by its index in the given footprint, and returns the
	// indices of the alive behaviours that should be kept alive due to recording
	// of the given behaviour.
	RecordBehaviourEffects(behaviourIndex uint64, ft *Footprint) []uint64
	// Clear clears the internal state of the BackPropagationMachine.
	Clear()
}

// dummyMachine does nothing but marks all the incoming Behaviours as alive.
type dummyMachine struct{}

func (m *dummyMachine) IsAlive(uint64, *Footprint) bool { return true }
func (m *dummyMachine) RecordBehaviourEffects(behaviourIndex uint64,
	ft *Footprint) []uint64 {
	return []uint64{behaviourIndex}
}

func (m *dummyMachine) Clear() {}

// GetBehaviourIndex returns the index of the last Behaviour in the Footprint
// which belongs to the command or subcomand indexed by the given SubCmdIdx. In
// case the SubCmdIdx is invalid or the Behaviour index is not found, error
// will be logged and uint64(0) will be returned.
func (fpt *Footprint) GetBehaviourIndex(ctx context.Context,
	fci api.SubCmdIdx) uint64 {
	switch len(fci) {
	case 1,
		// Only API command ID
		4,
		// For Vulkan: ID of VkQueueSubmit -> SubmitInfo Index ->
		// CommandBuffer Index -> Command Index
		6:
		// For Vulkan: ID of VkQueueSubmit -> SubmitInfo Index ->
		// CommandBuffer Index -> Index of vkCmdExecuteCommands ->
		// secondary CommandBuffer Index -> Command Index
		v := fpt.commandIndexToBehaviourIndexLookupTable.Value(fci)
		if u, ok := v.(uint64); ok {
			return u
		} else {
			log.E(ctx, "Invalid value of behaviour index: %v, (not a uint64)", v)
			return uint64(0)
		}
	default:
		log.E(ctx, "Invalid length of SubCmdIdx (full command index): %v", len(fci))
		return 0
	}
}

// AddBehaviour adds the given Behaviour to the Footprint and updates the
// internal mapping from SubCmdIdx to the last Behaviour that belongs to that
// command or subcommand.
func (fpt *Footprint) AddBehaviour(ctx context.Context, b *Behaviour) bool {
	bi := uint64(len(fpt.Behaviours))
	fci := b.BelongTo
	switch len(fci) {
	case 1, 4, 6:
		fpt.commandIndexToBehaviourIndexLookupTable.SetValue(fci, bi)
		fpt.Behaviours = append(fpt.Behaviours, b)
		fpt.BehaviourIndices[b] = bi
		return true
	default:
		log.E(ctx, "Invalid length of SubCmdIdx (full command index): %v", len(fci))
		return false
	}
}

// FootprintBuilderAPI provides FootprintBuilder
type FootprintBuilderAPI interface {
	FootprintBuilder(context.Context) FootprintBuilder
}

// FootprintBuilder builds Footprint from a stream of commands.
type FootprintBuilder interface {
	BuildFootprint(context.Context, *api.State, *Footprint, api.CmdID, api.Cmd)
}

func GetFootprint(ctx context.Context) (*Footprint, error) {
	r, err := database.Build(ctx, &FootprintResolvable{Capture: capture.Get(ctx)})
	if err != nil {
		return nil, fmt.Errorf("Counld not get execution foot print: %v", err)
	}
	return r.(*Footprint), nil
}

func (r *FootprintResolvable) Resolve(ctx context.Context) (interface{},
	error) {
	c, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	cmds := c.Commands
	builders := map[api.API]FootprintBuilder{}

	ft := NewFootprint(ctx, cmds)

	s := c.NewState()
	t0 := footprintBuildCounter.Start()
	for i, cmd := range cmds {
		a := cmd.API()
		if _, ok := builders[a]; !ok {
			if bp, ok := a.(FootprintBuilderAPI); ok {
				builders[a] = bp.FootprintBuilder(ctx)
			} else {
				// API does not provide execution footprint info, always keep commands
				// fromt such APIs alive.
				bh := NewBehaviour(api.SubCmdIdx{uint64(i)}, &dummyMachine{})
				bh.Alive = true
				// Even if the command does not belong to an API that provides
				// execution footprint info, we still need to mutate it in the new
				// state, because following commands in other APIs may depends on the
				// side effect of the this command.
				if err := cmd.Mutate(ctx, s, nil); err != nil {
					log.W(ctx, "Command %v %v: %v", api.CmdID(i), cmd, err)
					bh.Aborted = true
					return bh, nil
				}
				ft.AddBehaviour(ctx, bh)
				continue
			}
		}
		builders[a].BuildFootprint(ctx, s, ft, api.CmdID(i), cmd)
	}
	footprintBuildCounter.Stop(t0)
	return ft, nil
}
