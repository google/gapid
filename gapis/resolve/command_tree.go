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
	"sort"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type CommandTreeNodeUserData struct {
	Thumbnail api.CmdID
}

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
	root api.CmdIDGroup
}

func (t *commandTree) index(indices []uint64) api.SpanItem {
	group := api.CmdGroupOrRoot(t.root)
	for _, idx := range indices {
		switch item := group.Index(idx).(type) {
		case api.CmdIDGroup:
			group = item
		case api.SubCmdRoot:
			group = item
		default:
			return item
		}
	}
	return group
}

func (t *commandTree) indices(id api.CmdID) []uint64 {
	out := []uint64{}
	group := t.root
	for {
		i := group.IndexOf(id)
		out = append(out, i)
		switch item := group.Index(i).(type) {
		case api.CmdIDGroup:
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
	case api.SubCmdIdx:
		return &service.CommandTreeNode{
			NumChildren: 0, // TODO: Subcommands
			Commands:    cmdTree.path.Capture.SubCommandRange(item, item),
		}, nil
	case api.CmdIDGroup:
		return &service.CommandTreeNode{
			NumChildren: item.Count(),
			Commands:    cmdTree.path.Capture.CommandRange(uint64(item.Range.First()), uint64(item.Range.Last())),
			Group:       item.Name,
			NumCommands: item.DeepCount(func(g api.CmdIDGroup) bool { return true /* TODO: Subcommands */ }),
		}, nil
	case api.SubCmdRoot:
		count := uint64(1)
		g := ""
		commandStart := append([]uint64{}, item.Id...)
		commandEnd := append([]uint64{}, item.Id...)
		if len(item.Id) > 1 {
			g = fmt.Sprintf("%v", item.Id)
			commandStart = append(commandStart, uint64(0))
			commandEnd = append(commandEnd, uint64(item.SubGroup.Count()-1))
			count = uint64(item.SubGroup.Count())
		}
		return &service.CommandTreeNode{
			NumChildren: item.SubGroup.Bounds().Length(),
			Commands:    cmdTree.path.Capture.SubCommandRange(commandStart, commandEnd),
			Group:       g,
			NumCommands: count,
		}, nil
	default:
		panic(fmt.Errorf("Unexpected type: %T, cmdTree.index(c.Indices): %v, indices: %v",
			item, cmdTree.index(c.Indices), c.Indices))
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

	cmdIdx := p.Command.Indices[0]
	if len(p.Command.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported for Command Tree") // TODO: Subcommands
	}

	return &path.CommandTreeNode{
		Tree:    p.Tree,
		Indices: cmdTree.indices(api.CmdID(cmdIdx)),
	}, nil
}

type group struct {
	start api.CmdID
	end   api.CmdID
	name  string
}

type grouper interface {
	process(context.Context, api.CmdID, api.Cmd, *api.State)
	flush(count uint64)
	groups() []group
}

type runGrouper struct {
	f       func(cmd api.Cmd, s *api.State) (value interface{}, name string)
	start   api.CmdID
	current interface{}
	name    string
	out     []group
}

func (g *runGrouper) process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State) {
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
	end := api.CmdID(count)
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

func (g *markerGrouper) push(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State) {
	var name string
	if l, ok := cmd.(api.Labeled); ok {
		name = l.Label(ctx, s)
	}
	if len(name) > 0 {
		g.stack = append(g.stack, group{start: id, name: fmt.Sprintf("\"%s\"", name)})
	} else {
		g.stack = append(g.stack, group{start: id, name: fmt.Sprintf("Marker %d", g.count)})
		g.count++
	}
}

func (g *markerGrouper) pop(id api.CmdID) {
	m := g.stack[len(g.stack)-1]
	m.end = id + 1 // +1 to include pop marker
	g.out = append(g.out, m)
	g.stack = g.stack[:len(g.stack)-1]
}

func (g *markerGrouper) process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State) {
	flags := cmd.CmdFlags(ctx, s)
	if flags.IsPushUserMarker() {
		g.push(ctx, id, cmd, s)
	}
	if flags.IsPopUserMarker() && len(g.stack) > 0 {
		g.pop(id)
	}
}

func (g *markerGrouper) flush(count uint64) {
	for len(g.stack) > 0 {
		g.pop(api.CmdID(count) - 1)
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

	syncData, err := database.Build(ctx, &SynchronizationResolvable{p.Capture})
	if err != nil {
		return nil, log.Errf(ctx, nil, "Error building sync data")
	}
	snc, ok := syncData.(*sync.Data)
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find valid Synchronization Data")
	}

	filter, err := buildFilter(ctx, p.Capture, p.Filter, snc)
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

	// Walk the list of unfiltered commands to build the groups.
	s := c.NewState()
	api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, s, nil)
		if filter(id, cmd, s) {
			for _, g := range groupers {
				g.process(ctx, id, cmd, s)
			}
		}
		return nil
	})
	for _, g := range groupers {
		g.flush(uint64(len(c.Commands)))
	}

	// Build the command tree
	out := &commandTree{
		path: p,
		root: api.CmdIDGroup{
			Name:  "root",
			Range: api.CmdIDRange{End: api.CmdID(len(c.Commands))},
		},
	}
	for _, g := range groupers {
		for _, l := range g.groups() {
			out.root.AddGroup(l.start, l.end, l.name)
		}
	}

	if p.GroupByDrawCall || p.GroupByFrame {
		events, err := Events(ctx, &path.Events{
			Capture:            p.Capture,
			Filter:             p.Filter,
			DrawCalls:          true,
			TransformFeedbacks: true,
			FirstInFrame:       true,
			LastInFrame:        true,
		})
		if err != nil {
			return nil, log.Errf(ctx, err, "Couldn't get events")
		}
		if p.GroupByFrame {
			addFrameGroups(ctx, events, p, out, api.CmdID(len(c.Commands)))
		}
		if p.GroupByTransformFeedback {
			addFrameEventGroups(ctx, events, p, out, api.CmdID(len(c.Commands)),
				service.EventKind_TransformFeedback, "Transform Feedback")
		}
		if p.GroupByDrawCall {
			addFrameEventGroups(ctx, events, p, out, api.CmdID(len(c.Commands)),
				service.EventKind_DrawCall, "Draw")
		}
	}

	drawOrClearCmds := api.Spans{} // All the spans will have length 1

	// Now we have all the groups, we finally need to add the filtered commands.
	s = c.NewState()
	api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, s, nil)

		if !filter(id, cmd, s) {
			return nil
		}

		if v, ok := snc.SubcommandGroups[id]; ok {
			r := out.root.AddRoot([]uint64{uint64(id)})
			// subcommands are added before nested SubCmdRoots.
			cv := append([]api.SubCmdIdx{}, v...)
			sort.SliceStable(cv, func(i, j int) bool { return len(cv[i]) < len(cv[j]) })
			for _, x := range cv {
				r.Insert([]uint64{uint64(id)}, append([]uint64{}, x...))
			}
			return nil
		}

		out.root.AddCommand(id)

		if flags := cmd.CmdFlags(ctx, s); flags.IsDrawCall() || flags.IsClear() {
			drawOrClearCmds = append(drawOrClearCmds, &api.CmdIDRange{
				Start: id, End: id + 1,
			})
		}

		return nil
	})

	// Finally cluster the commands
	out.root.Cluster(uint64(p.MaxChildren), uint64(p.MaxNeighbours))

	setThumbnails(ctx, &out.root, drawOrClearCmds)

	return out, nil
}

func addFrameEventGroups(
	ctx context.Context,
	events *service.Events,
	p *path.CommandTree,
	t *commandTree,
	last api.CmdID,
	kind service.EventKind,
	prefix string) {

	count := 0
	for _, e := range events.List {
		i := api.CmdID(e.Command.Indices[0])
		switch e.Kind {
		case kind:
			// Find group which contains this event
			group := &t.root
			for true {
				if idx := group.Spans.IndexOf(i); idx != -1 {
					if subgroup, ok := group.Spans[idx].(*api.CmdIDGroup); ok {
						group = subgroup
						continue
					}
				}
				break
			}

			// Start with group of size 1 and grow it backward as long as nothing gets in the way.
			start := i
			for start >= group.Bounds().Start+1 && group.Spans.IndexOf(start-1) == -1 {
				start--
			}

			t.root.AddGroup(start, i+1, fmt.Sprintf("%v %v", prefix, count+1))
			count++

		case service.EventKind_LastInFrame:
			count = 0
		}
	}
}

func addFrameGroups(ctx context.Context, events *service.Events, p *path.CommandTree, t *commandTree, last api.CmdID) {
	frameCount, frameStart, frameEnd := 0, api.CmdID(0), api.CmdID(0)
	for _, e := range events.List {
		i := api.CmdID(e.Command.Indices[0])
		switch e.Kind {
		case service.EventKind_FirstInFrame:
			frameStart = i

			// If the start is within existing group, move it past the end of the group
			if idx := t.root.Spans.IndexOf(frameStart); idx != -1 {
				if subgroup, ok := t.root.Spans[idx].(*api.CmdIDGroup); ok {
					frameStart = subgroup.Range.End
				}
			}

		case service.EventKind_LastInFrame:
			frameCount++
			frameEnd = i

			// If the end is within existing group, move it to the end of the group
			if idx := t.root.Spans.IndexOf(frameEnd); idx != -1 {
				if subgroup, ok := t.root.Spans[idx].(*api.CmdIDGroup); ok {
					frameEnd = subgroup.Range.Last()
				}
			}

			// If the app properly annotates frames as well, we will end up with
			// both groupings, where one is the only child of the other.
			// However, we can not reliably detect this situation as the user
			// group might be surrounded by (potentially filtered) commands.

			t.root.AddGroup(frameStart, frameEnd+1, fmt.Sprintf("Frame %v", frameCount))
		}
	}
	if p.AllowIncompleteFrame && frameCount > 0 && frameStart > frameEnd {
		t.root.AddGroup(frameStart, last, "Incomplete Frame")
	}
}

func setThumbnails(ctx context.Context, g *api.CmdIDGroup, drawOrClearCmds api.Spans) {
	if s, c := interval.Intersect(drawOrClearCmds, g.Bounds().Span()); c > 0 {
		thumbnail := drawOrClearCmds[s+c-1].Bounds().Start
		g.UserData = &CommandTreeNodeUserData{Thumbnail: thumbnail}
	}

	for _, s := range g.Spans {
		if subgroup, ok := s.(*api.CmdIDGroup); ok {
			setThumbnails(ctx, subgroup, drawOrClearCmds)
		}
	}
}
