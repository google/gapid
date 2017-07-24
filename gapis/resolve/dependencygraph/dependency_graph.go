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

// The following are the imports that generated source files pull in when present
// Having these here helps out tools that can't cope with missing dependancies
import (
	_ "github.com/google/gapid/gapis/service/path"
)

var dependencyGraphBuildCounter = benchmark.GlobalCounters.Duration("dependencyGraph.build")

type DependencyGraph struct {
	Commands   []api.Cmd             // Atom list which this graph was build for.
	Behaviours []AtomBehaviour       // State reads/writes for each atom (graph edges).
	Roots      map[StateAddress]bool // State to mark live at requested atoms.
	addressMap addressMapping        // Remap state keys to integers for performance.
}

func (g *DependencyGraph) GetStateAddressOf(key StateKey) StateAddress {
	return g.addressMap.addressOf(key)
}

func (g *DependencyGraph) GetHierarchyStateMap() map[StateAddress]StateAddress {
	return g.addressMap.parent
}

func (g *DependencyGraph) SetRoot(key StateKey) {
	g.Roots[g.GetStateAddressOf(key)] = true
}

func (g *DependencyGraph) Print(ctx context.Context, b *AtomBehaviour) {
	for _, read := range b.Reads {
		key := g.addressMap.key[read]
		log.I(ctx, " - read [%v]%T%+v", read, key, key)
	}
	for _, modify := range b.Modifies {
		key := g.addressMap.key[modify]
		log.I(ctx, " - modify [%v]%T%+v", modify, key, key)
	}
	for _, write := range b.Writes {
		key := g.addressMap.key[write]
		log.I(ctx, " - write [%v]%T%+v", write, key, key)
	}
	if b.Aborted {
		log.I(ctx, " - aborted")
	}
}

type StateAddress uint32

const NullStateAddress = StateAddress(0)

// State key uniquely represents part of the GL state.
// Think of it as memory range (which stores the state data).
type StateKey interface {
	// Parent returns enclosing state (and this state is strict subset of it).
	// This allows efficient implementation of operations which access a lot state.
	Parent() StateKey
}

type AtomBehaviour struct {
	Reads     []StateAddress // States read by an atom.
	Modifies  []StateAddress // States read and written by an atom.
	Writes    []StateAddress // States written by an atom.
	Roots     []StateAddress // States labeled as root by an atom.
	KeepAlive bool           // Force the atom to be live.
	Aborted   bool           // Mutation of this command aborts.
}

type addressMapping struct {
	address map[StateKey]StateAddress
	key     map[StateAddress]StateKey
	parent  map[StateAddress]StateAddress
}

func (m *addressMapping) addressOf(state StateKey) StateAddress {
	if a, ok := m.address[state]; ok {
		return a
	}
	address := StateAddress(len(m.address))
	m.address[state] = address
	m.key[address] = state
	m.parent[address] = m.addressOf(state.Parent())
	return address
}

func (b *AtomBehaviour) Read(g *DependencyGraph, state StateKey) {
	if state != nil {
		b.Reads = append(b.Reads, g.addressMap.addressOf(state))
	}
}

func (b *AtomBehaviour) Modify(g *DependencyGraph, state StateKey) {
	if state != nil {
		b.Modifies = append(b.Modifies, g.addressMap.addressOf(state))
	}
}

func (b *AtomBehaviour) Write(g *DependencyGraph, state StateKey) {
	if state != nil {
		b.Writes = append(b.Writes, g.addressMap.addressOf(state))
	}
}

type DependencyGraphBehaviourProvider interface {
	GetDependencyGraphBehaviourProvider(ctx context.Context) BehaviourProvider
}

type BehaviourProvider interface {
	GetBehaviourForAtom(context.Context, *api.State, api.CmdID, api.Cmd, *DependencyGraph) AtomBehaviour
}

func GetDependencyGraph(ctx context.Context) (*DependencyGraph, error) {
	r, err := database.Build(ctx, &DependencyGraphResolvable{Capture: capture.Get(ctx)})
	if err != nil {
		return nil, fmt.Errorf("Could not calculate dependency graph: %v", err)
	}
	return r.(*DependencyGraph), nil
}

func (r *DependencyGraphResolvable) Resolve(ctx context.Context) (interface{}, error) {
	c, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	cmds := c.Commands
	behaviourProviders := map[api.API]BehaviourProvider{}

	g := &DependencyGraph{
		Commands:   cmds,
		Behaviours: make([]AtomBehaviour, len(cmds)),
		Roots:      map[StateAddress]bool{},
		addressMap: addressMapping{
			address: map[StateKey]StateAddress{nil: NullStateAddress},
			key:     map[StateAddress]StateKey{NullStateAddress: nil},
			parent:  map[StateAddress]StateAddress{NullStateAddress: NullStateAddress},
		},
	}

	s := c.NewState()
	t0 := dependencyGraphBuildCounter.Start()
	err = api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		a := cmd.API()
		if _, ok := behaviourProviders[a]; !ok {
			if bp, ok := a.(DependencyGraphBehaviourProvider); ok {
				behaviourProviders[a] = bp.GetDependencyGraphBehaviourProvider(ctx)
			} else {
				// API does not provide dependency information, always keep atoms for
				// such APIs.
				g.Behaviours[id].KeepAlive = true
				// Even if the atom does not belong to an API that provides dependency
				// info, we still need to mutate it in the new state, because following
				// atoms in other APIs may depends on the side effect of the current
				// atom.
				return cmd.Mutate(ctx, s, nil /* builder */)
			}
		}
		g.Behaviours[id] = behaviourProviders[a].GetBehaviourForAtom(ctx, s, id, cmd, g)
		return nil
	})
	if err != nil {
		return AtomBehaviour{Aborted: true}, nil
	}
	dependencyGraphBuildCounter.Stop(t0)
	return g, nil
}
