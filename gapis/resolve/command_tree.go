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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// CommandTree resolves the specified command tree path.
func CommandTree(ctx context.Context, c *path.CommandTree) (*service.CommandTree, error) {
	id, err := database.Store(ctx, &CommandTreeResolvable{c})
	if err != nil {
		return nil, err
	}
	return &service.CommandTree{
		Root: &path.CommandTreeNode{Tree: path.NewID(id)},
	}, nil
}

type commandTree struct {
	path *path.CommandTree
	root atom.Group
}

func (t *commandTree) index(indices []uint64) atom.GroupOrID {
	group := t.root
	for _, idx := range indices {
		switch item := group.Index(idx).(type) {
		case atom.Group:
			group = item
		default:
			return item
		}
	}
	return group
}

func (t *commandTree) indices(id atom.ID) []uint64 {
	out := []uint64{}
	group := t.root
	for {
		i := group.IndexOf(id)
		switch item := group.Index(i).(type) {
		case atom.Group:
			out = append(out, i)
			group = item
		default:
			return out
		}
	}
}

// CommandTreeNode resolves the specified command tree node path.
func CommandTreeNode(ctx context.Context, c *path.CommandTreeNode) (*service.CommandTreeNode, error) {
	boxed, err := database.Resolve(ctx, c.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxed.(*commandTree)

	switch item := cmdTree.index(c.Index).(type) {
	case atom.ID:
		return &service.CommandTreeNode{
			NumChildren: 0, // TODO: Subcommands
			Commands:    cmdTree.path.Capture.CommandRange(uint64(item), uint64(item)),
		}, nil
	case atom.Group:
		return &service.CommandTreeNode{
			NumChildren: item.Count(),
			Commands:    cmdTree.path.Capture.CommandRange(uint64(item.Range.First()), uint64(item.Range.Last())),
			Group:       item.Name,
		}, nil
	default:
		panic(fmt.Errorf("Unexpected type: %T", item))
	}
}

// CommandTreeNodeForCommand returns the path to the CommandTreeNode that
// represents the specified command.
func CommandTreeNodeForCommand(ctx context.Context, p *path.CommandTreeNodeForCommand) (*path.CommandTreeNode, error) {
	boxed, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxed.(*commandTree)

	atomIdx := p.Command.Index[0]
	if len(p.Command.Index) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}

	return &path.CommandTreeNode{
		Tree:  p.Tree,
		Index: cmdTree.indices(atom.ID(atomIdx)),
	}, nil
}

type group struct {
	start atom.ID
	end   atom.ID
	name  string
}

type grouper interface {
	process(ctx context.Context, id atom.ID, a atom.Atom, s *gfxapi.State)
	flush(count uint64)
	groups() []group
}

type runGrouper struct {
	f       func(a atom.Atom, s *gfxapi.State) (value interface{}, name string)
	start   atom.ID
	current interface{}
	name    string
	out     []group
}

func (g *runGrouper) process(ctx context.Context, id atom.ID, a atom.Atom, s *gfxapi.State) {
	val, name := g.f(a, s)
	if val != g.current {
		if g.current != nil {
			g.out = append(g.out, group{g.start, id, g.name})
		}
		g.start = id
	}
	g.current, g.name = val, name
}

func (g *runGrouper) flush(count uint64) {
	end := atom.ID(count)
	if g.current != nil && g.start != end {
		g.out = append(g.out, group{g.start, end, g.name})
	}
}

func (g *runGrouper) groups() []group { return g.out }

type markerGrouper struct {
	stack []group
	count int
	out   []group
}

func (g *markerGrouper) push(ctx context.Context, id atom.ID, a atom.Atom, s *gfxapi.State) {
	if l, ok := a.(atom.Labeled); ok {
		g.stack = append(g.stack, group{start: id, name: l.Label(ctx, s)})
	} else {
		g.stack = append(g.stack, group{start: id, name: fmt.Sprintf("Marker %d", g.count)})
		g.count++
	}
}

func (g *markerGrouper) pop(id atom.ID) {
	m := g.stack[len(g.stack)-1]
	m.end = id + 1 // +1 to include pop marker
	g.out = append(g.out, m)
	g.stack = g.stack[:len(g.stack)-1]
}

func (g *markerGrouper) process(ctx context.Context, id atom.ID, a atom.Atom, s *gfxapi.State) {
	if a.AtomFlags().IsPushUserMarker() {
		g.push(ctx, id, a, s)
	}
	if a.AtomFlags().IsPopUserMarker() && len(g.stack) > 0 {
		g.pop(id)
	}
}

func (g *markerGrouper) flush(count uint64) {
	for len(g.stack) > 0 {
		g.pop(atom.ID(count))
	}
}

func (g *markerGrouper) groups() []group { return g.out }

// Resolve builds and returns a *commandTree for the path.CommandTreeNode.
// Resolve implements the database.Resolver interface.
func (r *CommandTreeResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	filters := filters{}

	if r.Path.Context.IsValid() {
		if err := filters.addContextFilter(ctx, r.Path.Capture.Context(r.Path.Context)); err != nil {
			return nil, err
		}
	}

	// TODO: Thread filter.

	groupers := []grouper{}

	groupers = append(groupers, &runGrouper{f: func(a atom.Atom, s *gfxapi.State) (interface{}, string) {
		if api := a.API(); api != nil {
			if context := api.Context(s); context != nil {
				return api.ID(), api.Name()
			}
		}
		return nil, "No context"
	}})

	// TODO: Threads
	// groupers = append(groupers, &runGrouper{f: func(a atom.Atom, s *gfxapi.State) (interface{}, string) {
	// 	return s.Thread, fmt.Sprintf("Thread: 0x%x", s.Thread)
	// }})

	groupers = append(groupers, &markerGrouper{})

	// Walk the list of unfiltered atoms to build the groups.
	s := c.NewState()
	for i, a := range c.Atoms {
		a.Mutate(ctx, s, nil)
		if filters.pass(a, s) {
			for _, g := range groupers {
				g.process(ctx, atom.ID(i), a, s)
			}
		}
	}
	for _, g := range groupers {
		g.flush(uint64(len(c.Atoms)))
	}

	// Build the command tree
	out := &commandTree{
		path: r.Path,
		root: atom.Group{
			Name:  "root",
			Range: atom.Range{End: atom.ID(len(c.Atoms))},
		},
	}
	for _, g := range groupers {
		for _, l := range g.groups() {
			out.root.AddGroup(l.start, l.end, l.name)
		}
	}

	addDrawAndFrameEvents(ctx, r.Path, out)

	// Now we have all the groups, we finally need to add the filtered atoms.

	s = c.NewState()
	out.root.AddAtoms(func(i atom.ID) bool {
		a := c.Atoms[i]
		a.Mutate(ctx, s, nil)
		return filters.pass(a, s)
	})

	log.W(ctx, "%+v", out.root)

	return out, nil
}

func addDrawAndFrameEvents(ctx context.Context, p *path.CommandTree, t *commandTree) error {
	events, err := Events(ctx, &path.Events{
		Commands:     t.path.Capture.Commands(),
		Context:      p.Context,
		Thread:       p.Thread,
		DrawCalls:    true,
		FirstInFrame: true,
		LastInFrame:  true,
	})
	if err != nil {
		return log.Errf(ctx, err, "Couldn't get events")
	}

	drawCount, drawStart := 0, atom.ID(0)
	frameCount, frameStart := 0, atom.ID(0)

	for _, e := range events.List {
		i := atom.ID(e.Command.Index[0])
		switch e.Kind {
		case service.EventKind_DrawCall:
			err := t.root.AddGroup(drawStart, i+1, fmt.Sprintf("Draw %v", drawCount+1))
			if err != nil {
				log.W(ctx, "Draw AddGroup errored: %v", err)
			}

			drawCount++
			drawStart = i + 1

		case service.EventKind_FirstInFrame:
			drawCount, drawStart, frameStart = 0, i, i

		case service.EventKind_LastInFrame:
			err := t.root.AddGroup(frameStart, i+1, fmt.Sprintf("Frame %v", frameCount+1))
			if err != nil {
				log.W(ctx, "Frame AddGroup errored: %v", err)
			}
			frameCount++
		}
	}
	return nil
}
