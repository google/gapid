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

// CommandTreeNode resolves the specified command tree node path.
func CommandTreeNode(ctx context.Context, c *path.CommandTreeNode) (*service.CommandTreeNode, error) {
	boxed, err := database.Resolve(ctx, c.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxed.(*commandTree)

	group := &cmdTree.root
	for _, idx := range c.Index {
		var i uint64
		i, group = group.Index(idx)
		if group == nil {
			return &service.CommandTreeNode{
				NumChildren: 0, // TODO: Subcommands
				Data: &service.CommandTreeNode_Command{
					Command: cmdTree.path.Capture.Command(i),
				},
			}, nil
		}
	}

	return &service.CommandTreeNode{
		NumChildren: group.Count(),
		Data: &service.CommandTreeNode_Group{
			Group: group.Name,
		},
	}, nil
}

type group struct {
	start uint64
	end   uint64
	name  string
}

type grouper interface {
	process(ctx context.Context, i uint64, a atom.Atom, s *gfxapi.State)
	flush(count uint64)
	groups() []group
}

type runGrouper struct {
	f       func(a atom.Atom, s *gfxapi.State) (value interface{}, name string)
	start   uint64
	current interface{}
	name    string
	out     []group
}

func (g *runGrouper) process(ctx context.Context, i uint64, a atom.Atom, s *gfxapi.State) {
	val, name := g.f(a, s)
	if val != g.current && g.current != nil {
		g.out = append(g.out, group{g.start, i, g.name})
	}
	g.start, g.current, g.name = i, val, name
}

func (g *runGrouper) flush(count uint64) {
	g.out = append(g.out, group{g.start, count, g.name})
}

func (g *runGrouper) groups() []group { return g.out }

type markerGrouper struct {
	stack []group
	count int
	out   []group
}

func (g *markerGrouper) push(ctx context.Context, i uint64, a atom.Atom, s *gfxapi.State) {
	if l, ok := a.(atom.Labeled); ok {
		g.stack = append(g.stack, group{start: i, name: l.Label(ctx, s)})
	} else {
		g.stack = append(g.stack, group{start: i, name: fmt.Sprintf("Marker %d", g.count)})
		g.count++
	}
}

func (g *markerGrouper) pop(i uint64) {
	m := g.stack[len(g.stack)-1]
	m.end = i + 1 // +1 to include pop marker
	g.out = append(g.out, m)
	g.stack = g.stack[:len(g.stack)-1]
}

func (g *markerGrouper) process(ctx context.Context, i uint64, a atom.Atom, s *gfxapi.State) {
	if a.AtomFlags().IsPushUserMarker() {
		g.push(ctx, i, a, s)
	}
	if a.AtomFlags().IsPopUserMarker() && len(g.stack) > 0 {
		g.pop(i)
	}
}

func (g *markerGrouper) flush(count uint64) {
	for len(g.stack) > 0 {
		g.pop(count)
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

	// TODO: Filtering.

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
		for _, g := range groupers {
			g.process(ctx, uint64(i), a, s)
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
			Range: atom.Range{End: uint64(len(c.Atoms))},
		},
	}
	for _, g := range groupers {
		for _, l := range g.groups() {
			out.root.SubGroups.Add(l.start, l.end, l.name)
		}
	}

	addDrawAndFrameEvents(ctx, out)

	return out, nil
}

func addDrawAndFrameEvents(ctx context.Context, t *commandTree) error {
	events, err := Events(ctx, &path.Events{
		Commands:     t.path.Capture.Commands(),
		DrawCalls:    true,
		FirstInFrame: true,
		LastInFrame:  true,
	})
	if err != nil {
		return log.Errf(ctx, err, "Couldn't get events")
	}

	drawCount, drawStart := 0, uint64(0)
	frameCount, frameStart := 0, uint64(0)

	for _, e := range events.List {
		i := e.Command.Index[0]
		switch e.Kind {
		case service.EventKind_DrawCall:
			err := t.root.SubGroups.Add(drawStart, i+1, fmt.Sprintf("Draw %v", drawCount))
			if err != nil {
				log.W(ctx, "Draw SubGroups.Add errored: %v", err)
			}

			drawCount++
			drawStart = i + 1

		case service.EventKind_FirstInFrame:
			drawCount, drawStart, frameStart = 0, i, i

		case service.EventKind_LastInFrame:
			err := t.root.SubGroups.Add(frameStart, i+1, fmt.Sprintf("Frame %v", frameCount))
			if err != nil {
				log.W(ctx, "Frame SubGroups.Add errored: %v", err)
			}
			frameCount++
		}
	}
	return nil
}
