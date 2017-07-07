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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
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
		out = append(out, i)
		switch item := group.Index(i).(type) {
		case atom.Group:
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

	switch item := cmdTree.index(c.Indices).(type) {
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
			NumCommands: item.DeepCount(func(g atom.Group) bool { return true /* TODO: Subcommands */ }),
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

	atomIdx := p.Command.Indices[0]
	if len(p.Command.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported for Command Tree") // TODO: Subcommands
	}

	return &path.CommandTreeNode{
		Tree:    p.Tree,
		Indices: cmdTree.indices(atom.ID(atomIdx)),
	}, nil
}

type group struct {
	start atom.ID
	end   atom.ID
	name  string
}

type grouper interface {
	process(context.Context, atom.ID, api.Cmd, *api.State)
	flush(count uint64)
	groups() []group
}

type runGrouper struct {
	f       func(cmd api.Cmd, s *api.State) (value interface{}, name string)
	start   atom.ID
	current interface{}
	name    string
	out     []group
}

func (g *runGrouper) process(ctx context.Context, id atom.ID, cmd api.Cmd, s *api.State) {
	val, name := g.f(cmd, s)
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

func (g *markerGrouper) push(ctx context.Context, id atom.ID, cmd api.Cmd, s *api.State) {
	var name string
	if l, ok := cmd.(atom.Labeled); ok {
		name = l.Label(ctx, s)
	}
	if len(name) > 0 {
		g.stack = append(g.stack, group{start: id, name: name})
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

func (g *markerGrouper) process(ctx context.Context, id atom.ID, cmd api.Cmd, s *api.State) {
	if cmd.CmdFlags().IsPushUserMarker() {
		g.push(ctx, id, cmd, s)
	}
	if cmd.CmdFlags().IsPopUserMarker() && len(g.stack) > 0 {
		g.pop(id)
	}
}

func (g *markerGrouper) flush(count uint64) {
	for len(g.stack) > 0 {
		g.pop(atom.ID(count) - 1)
	}
}

func (g *markerGrouper) groups() []group { return g.out }

// Resolve builds and returns a *commandTree for the path.CommandTreeNode.
// Resolve implements the database.Resolver interface.
func (r *CommandTreeResolvable) Resolve(ctx context.Context) (interface{}, error) {
	p := r.Path
	ctx = capture.Put(ctx, p.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	filter, err := buildFilter(ctx, p.Capture, p.Filter)
	if err != nil {
		return nil, err
	}

	groupers := []grouper{}

	if p.GroupByApi {
		groupers = append(groupers, &runGrouper{f: func(cmd api.Cmd, s *api.State) (interface{}, string) {
			if api := cmd.API(); api != nil {
				return api.ID(), api.Name()
			}
			return nil, "No context"
		}})
	}

	if p.GroupByContext {
		var noContextID interface{}
		if p.IncludeNoContextGroups {
			noContextID = api.ContextID{}
		}
		groupers = append(groupers, &runGrouper{f: func(cmd api.Cmd, s *api.State) (interface{}, string) {
			if api := cmd.API(); api != nil {
				if context := api.Context(s, cmd.Thread()); context != nil {
					return context.ID(), context.Name()
				}
			}
			return noContextID, "No context"
		}})
	}

	if p.GroupByThread {
		groupers = append(groupers, &runGrouper{f: func(cmd api.Cmd, s *api.State) (interface{}, string) {
			thread := cmd.Thread()
			return thread, fmt.Sprintf("Thread: 0x%x", thread)
		}})
	}

	if p.GroupByUserMarkers {
		groupers = append(groupers, &markerGrouper{})
	}

	// Walk the list of unfiltered atoms to build the groups.
	s := c.NewState()
	for i, cmd := range c.Commands {
		id := atom.ID(i)
		if err := cmd.Mutate(ctx, s, nil); err != nil && err == context.Canceled {
			return nil, err
		}
		if filter(cmd, s) {
			for _, g := range groupers {
				g.process(ctx, id, cmd, s)
			}
		}
	}
	for _, g := range groupers {
		g.flush(uint64(len(c.Commands)))
	}

	// Build the command tree
	out := &commandTree{
		path: p,
		root: atom.Group{
			Name:  "root",
			Range: atom.Range{End: atom.ID(len(c.Commands))},
		},
	}
	for _, g := range groupers {
		for _, l := range g.groups() {
			out.root.AddGroup(l.start, l.end, l.name)
		}
	}

	if p.GroupByDrawCall || p.GroupByFrame {
		addDrawAndFrameEvents(ctx, p, out, atom.ID(len(c.Commands)))
	}

	// Now we have all the groups, we finally need to add the filtered atoms.

	s = c.NewState()
	out.root.AddAtoms(func(i atom.ID) bool {
		cmd := c.Commands[i]
		cmd.Mutate(ctx, s, nil)
		return filter(cmd, s)
	}, uint64(p.MaxChildren))

	return out, nil
}

func addDrawAndFrameEvents(ctx context.Context, p *path.CommandTree, t *commandTree, last atom.ID) error {
	events, err := Events(ctx, &path.Events{
		Capture:      t.path.Capture,
		Filter:       p.Filter,
		DrawCalls:    p.GroupByDrawCall,
		FirstInFrame: true,
		LastInFrame:  true,
	})
	if err != nil {
		return log.Errf(ctx, err, "Couldn't get events")
	}

	drawCount, drawStart := 0, atom.ID(0)
	frameCount, frameStart, frameEnd := 0, atom.ID(0), atom.ID(0)

	for _, e := range events.List {
		i := atom.ID(e.Command.Indices[0])
		switch e.Kind {
		case service.EventKind_DrawCall:
			t.root.AddGroup(drawStart, i+1, fmt.Sprintf("Draw %v", drawCount+1))
			drawCount++
			drawStart = i + 1

		case service.EventKind_FirstInFrame:
			drawCount, drawStart, frameStart = 0, i, i

		case service.EventKind_LastInFrame:
			if p.GroupByFrame {
				t.root.AddGroup(frameStart, i+1, fmt.Sprintf("Frame %v", frameCount+1))
				frameCount++
				frameEnd = i
			}
		}
	}

	if p.AllowIncompleteFrame && p.GroupByFrame && frameCount > 0 && frameStart > frameEnd {
		t.root.AddGroup(frameStart, last, "Incomplete Frame")
	}
	return nil
}
