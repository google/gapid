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
	"strings"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve/cmdgrouper"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// CmdGroupData is the additional metadata assigned to api.CmdIDGroups UserData
// field.
type CmdGroupData struct {
	Representation api.CmdID
}

// CommandTree resolves the specified command tree path.
func CommandTree(ctx context.Context, c *path.CommandTree, r *path.ResolveConfig) (*service.CommandTree, error) {
	id, err := database.Store(ctx, &CommandTreeResolvable{Path: c, Config: r})
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

func (t *commandTree) index(indices []uint64) (api.SpanItem, api.SubCmdIdx) {
	group := api.CmdGroupOrRoot(t.root)
	subCmdRootID := api.SubCmdIdx{}
	for _, idx := range indices {
		switch item := group.Index(idx).(type) {
		case api.CmdIDGroup:
			group = item
		case api.SubCmdRoot:
			// Each SubCmdRoot contains its absolute sub command index.
			subCmdRootID = item.Id
			group = item
		case api.SubCmdIdx:
			id := append(subCmdRootID, item...)
			return id, id
		default:
			return item, subCmdRootID
		}
	}
	return group, subCmdRootID
}

func (t *commandTree) indices(idx []uint64, preferGroup bool) []uint64 {
	out := []uint64{}
	group := t.root

	for _, id := range idx {
		brk := false
		for {
			if brk {
				break
			}
			i := group.IndexOf(api.CmdID(id))
			out = append(out, i)
			switch item := group.Index(i).(type) {
			case api.CmdIDGroup:
				group = item
				if preferGroup {
					brk = true
				}
			case api.SubCmdRoot:
				group = item.SubGroup
				brk = true
			default:
				return out
			}
		}
	}
	return out
}

// CommandTreeNode resolves the specified command tree node path.
func CommandTreeNode(ctx context.Context, c *path.CommandTreeNode, r *path.ResolveConfig) (*service.CommandTreeNode, error) {
	boxed, err := database.Resolve(ctx, c.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxed.(*commandTree)

	rawItem, absID := cmdTree.index(c.Indices)
	switch item := rawItem.(type) {
	case api.SubCmdIdx:
		cmdPath := cmdTree.path.Capture.Command(item[0], item[1:]...)
		cmd, err := Cmd(ctx, cmdPath, r)
		if err != nil {
			return nil, err
		}

		experimentalCmds := []*path.Command{}
		if cmd.CmdFlags().IsExecutedDraw() || cmd.CmdFlags().IsExecutedDispatch() {
			experimentalCmds = []*path.Command{&path.Command{Indices: cmdPath.Indices}}
		}

		return &service.CommandTreeNode{
			Representation:       cmdPath,
			NumChildren:          0, // TODO: Subcommands
			Commands:             cmdTree.path.Capture.SubCommandRange(item, item),
			ExperimentalCommands: experimentalCmds,
		}, nil
	case api.CmdIDGroup:
		representation := cmdTree.path.Capture.Command(uint64(item.Range.Last()))
		if data, ok := item.UserData.(*CmdGroupData); ok {
			representation = cmdTree.path.Capture.Command(uint64(data.Representation))
		}

		if len(absID) == 0 {
			// Not a CmdIDGroup under SubCmdRoot, does not contain Subcommands
			return &service.CommandTreeNode{
				Representation: representation,
				NumChildren:    item.Count(),
				Commands:       cmdTree.path.Capture.CommandRange(uint64(item.Range.First()), uint64(item.Range.Last())),
				Group:          item.Name,
				NumCommands:    item.DeepCount(func(g api.CmdIDGroup) bool { return true /* TODO: Subcommands */ }),
			}, nil
		}
		// Is a CmdIDGroup under SubCmdRoot, contains only Subcommands
		startID := append(absID, uint64(item.Range.First()))
		endID := append(absID, uint64(item.Range.Last()))
		representation = cmdTree.path.Capture.Command(endID[0], endID[1:]...)

		experimentalCmds := []*path.Command{}
		for _, e := range item.ExperimentableCmds {
			experimentalCmds = append(experimentalCmds, &path.Command{Indices: e})
		}

		if aliasRepId := getOpenGLAliasRepresentation(c.Indices, &item, cmdTree); aliasRepId != api.CmdNoID {
			aliasId := append(absID, uint64(aliasRepId))
			representation = cmdTree.path.Capture.Command(aliasId[0], aliasId[1:]...)
		}

		return &service.CommandTreeNode{
			Representation:       representation,
			NumChildren:          item.Count(),
			Commands:             cmdTree.path.Capture.SubCommandRange(startID, endID),
			Group:                item.Name,
			NumCommands:          item.DeepCount(func(g api.CmdIDGroup) bool { return true /* TODO: Subcommands */ }),
			ExperimentalCommands: experimentalCmds,
		}, nil

	case api.SubCmdRoot:
		count := uint64(1)
		g := ""
		if len(item.Id) > 1 {
			g = fmt.Sprintf("%v", item.SubGroup.Name)
			count = uint64(item.SubGroup.Count())
		}

		experimentalCmds := []*path.Command{}
		cmdPath := cmdTree.path.Capture.Command(item.Id[0], item.Id[1:]...)
		if cmd, _ := Cmd(ctx, cmdPath, r); cmd != nil {
			if cmd.CmdFlags().IsExecutedCommandBuffer() {
				experimentalCmds = []*path.Command{&path.Command{Indices: cmdPath.Indices}}
			}
		}

		return &service.CommandTreeNode{
			Representation:       cmdPath,
			NumChildren:          item.SubGroup.Count(),
			Commands:             cmdTree.path.Capture.SubCommandRange(item.Id, item.Id),
			Group:                g,
			NumCommands:          count,
			ExperimentalCommands: experimentalCmds,
		}, nil
	default:
		panic(fmt.Errorf("Unexpected type: %T, cmdTree.index(c.Indices): (%v, %v), indices: %v",
			item, rawItem, absID, c.Indices))
	}
}

// CommandTreeNodeForCommand returns the path to the CommandTreeNode that
// represents the specified command.
func CommandTreeNodeForCommand(ctx context.Context, p *path.CommandTreeNodeForCommand, r *path.ResolveConfig) (*path.CommandTreeNode, error) {
	boxed, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxed.(*commandTree)

	return &path.CommandTreeNode{
		Tree:    p.Tree,
		Indices: cmdTree.indices(p.Command.Indices, p.PreferGroup),
	}, nil
}

// Resolve builds and returns a *commandTree for the path.CommandTreeNode.
// Resolve implements the database.Resolver interface.
func (r *CommandTreeResolvable) Resolve(ctx context.Context) (interface{}, error) {
	p := r.Path
	ctx = SetupContext(ctx, p.Capture, r.Config)

	c, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}

	snc, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	filter, err := buildFilter(ctx, p.Capture, p.Filter, snc, r.Config)
	if err != nil {
		return nil, err
	}

	groupers := []cmdgrouper.Grouper{}

	if p.GroupByApi {
		groupers = append(groupers, cmdgrouper.Run(
			func(cmd api.Cmd, s *api.GlobalState) (interface{}, string) {
				if api := cmd.API(); api != nil {
					return api.ID(), api.Name()
				}
				return nil, "No context"
			}))
	}

	if p.GroupByThread {
		groupers = append(groupers, cmdgrouper.Run(
			func(cmd api.Cmd, s *api.GlobalState) (interface{}, string) {
				thread := cmd.Thread()
				return thread, fmt.Sprintf("Thread: 0x%x", thread)
			}))
	}

	// Walk the list of unfiltered commands to build the groups.
	s := c.NewState(ctx)
	err = api.ForeachCmd(ctx, c.Commands, false, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}
		if filter(id, cmd, s, api.SubCmdIdx([]uint64{uint64(id)})) {
			for _, g := range groupers {
				g.Process(ctx, id, cmd, s)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
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
		for _, l := range g.Build(api.CmdID(len(c.Commands))) {
			if group, err := out.root.AddGroup(l.Start, l.End, l.Name, []api.SubCmdIdx{}); err == nil {
				group.UserData = l.UserData
			}
		}
	}

	if p.GroupByFrame {
		addFrameGroups(ctx, p, out, c.Commands)
	}
	if p.GroupByTransformFeedback {
		addFrameEventGroups(ctx, p, out, c.Commands,
			func(id api.CmdID, cmd api.Cmd, s *api.GlobalState, idx api.SubCmdIdx) bool {
				return cmd.CmdFlags().IsTransformFeedback()
			},
			"Transform Feedback")
	}
	if p.GroupByDrawCall {
		addFrameEventGroups(ctx, p, out, c.Commands,
			func(id api.CmdID, cmd api.Cmd, s *api.GlobalState, idx api.SubCmdIdx) bool {
				return cmd.CmdFlags().IsDrawCall()
			},
			"Draw")
	}
	if p.GroupBySubmission {
		addContainingGroups(ctx, p, out, c.Commands,
			func(id api.CmdID, cmd api.Cmd, s *api.GlobalState, idx api.SubCmdIdx) bool {
				return cmd.CmdFlags().IsSubmission()
			},
			"Host Coordination")
	}

	drawOrClearCmds := api.Spans{} // All the spans will have length 1

	// Now we have all the groups, we finally need to add the filtered commands.
	s = c.NewState(ctx)
	err = api.ForeachCmd(ctx, c.Commands, false, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}

		if !filter(id, cmd, s, api.SubCmdIdx([]uint64{uint64(id)})) {
			return nil
		}

		if v, ok := snc.SubcommandGroups[id]; ok {
			subr := out.root.AddRoot([]uint64{uint64(id)}, snc.SubcommandNames)
			// subcommands are added before nesting SubCmdRoots.
			cv := append([]api.SubCmdIdx{}, v...)
			sort.SliceStable(cv, func(i, j int) bool { return len(cv[i]) < len(cv[j]) })
			for _, x := range cv {
				// subcommand marker groups are added before subcommands. And groups with
				// shorter indices are added before groups with longer indices.
				// SubCmdRoot will be created when necessary.
				parentIdx := append([]uint64{uint64(id)}, x[0:len(x)-1]...)
				// if snc.SubCommandMarkerGroups.Value(parentIdx) != nil && !p.OnlyExecutedDraws {
				if snc.SubCommandMarkerGroups.Value(parentIdx) != nil {
					markers := snc.SubCommandMarkerGroups.Value(parentIdx).([]*api.CmdIDGroup)
					subr.AddSubCmdMarkerGroups(x[0:len(x)-1], markers, snc.SubcommandNames)
				}
				subr.InsertWithFilter(append([]uint64{}, x...), snc.SubcommandNames, func(id api.CmdID) bool {
					subCmd, err := Cmd(ctx, &path.Command{
						Capture: p.Capture,
						Indices: append(parentIdx, uint64(id)),
					}, r.Config)
					if err == nil {
						return filter(id, subCmd, s, api.SubCmdIdx(append(parentIdx, uint64(id))))
					}

					return false
				})
			}
			return nil
		}

		out.root.AddCommand(id)

		if flags := cmd.CmdFlags(); flags.IsDrawCall() || flags.IsClear() {
			drawOrClearCmds = append(drawOrClearCmds, &api.CmdIDRange{
				Start: id, End: id + 1,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cluster the commands
	out.root.Cluster(uint64(p.MaxChildren), uint64(p.MaxNeighbours))

	// Set group representations.
	setRepresentations(ctx, &out.root, drawOrClearCmds)

	return out, nil
}

func addFrameEventGroups(
	ctx context.Context,
	p *path.CommandTree,
	t *commandTree,
	cmds []api.Cmd,
	filter CommandFilter,
	prefix string) {

	count := 0
	for i, e := range cmds {
		i := api.CmdID(i)
		if filter(i, e, nil, api.SubCmdIdx([]uint64{uint64(i)})) {
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

			t.root.AddGroup(start, i+1, fmt.Sprintf("%v %v", prefix, count+1), []api.SubCmdIdx{})
			count++
		}

		if e.CmdFlags().IsEndOfFrame() {
			count = 0
		}
	}
}

func addContainingGroups(
	ctx context.Context,
	p *path.CommandTree,
	t *commandTree,
	cmds []api.Cmd,
	filter CommandFilter,
	label string) {

	realFilter := func(id api.CmdID, cmd api.Cmd, s *api.GlobalState, idx api.SubCmdIdx) bool {
		return cmd.CmdFlags().IsEndOfFrame() || filter(id, cmd, s, idx)
	}

	lastLeft := api.CmdID(0)
	for i, e := range cmds {
		i := api.CmdID(i)
		if realFilter(i, e, nil, api.SubCmdIdx([]uint64{uint64(i)})) {
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
			if lastLeft != 0 && start < lastLeft+1 {
				start = lastLeft + 1
			}
			end := i
			lastLeft = end
			if start < end {
				t.root.AddGroup(start, end, label, []api.SubCmdIdx{})
			}
		}
	}
}

type frame struct {
	index int
	start api.CmdID
	end   api.CmdID
	repr  api.CmdID
}

func (f frame) addGroup(t *commandTree) {
	group, _ := t.root.AddGroup(f.start, f.end+1, fmt.Sprintf("Frame %v", f.index), []api.SubCmdIdx{})
	if group != nil {
		group.UserData = &CmdGroupData{Representation: f.repr}
	}
}

func addFrameGroups(ctx context.Context, p *path.CommandTree, t *commandTree, cmds []api.Cmd) {

	eofCommands := make([]api.Cmd, 0)
	for _, cmd := range cmds {
		if cmd.CmdFlags().IsEndOfFrame() {
			eofCommands = append(eofCommands, cmd)
		}
	}

	frameCount := 0
	startFrame := 0

	for i, cmd := range cmds {
		if cmd.CmdFlags().IsEndOfFrame() {
			t.root.AddGroup(api.CmdID(startFrame), api.CmdID(i+1), fmt.Sprintf("Frame %v", frameCount+1), []api.SubCmdIdx{})
			startFrame = i + 1
			frameCount++
		}
	}

	if p.AllowIncompleteFrame && frameCount > 0 && len(cmds) > startFrame {
		t.root.AddGroup(api.CmdID(startFrame), api.CmdID(len(cmds)), fmt.Sprintf("[Incomplete] Frame", frameCount+1), []api.SubCmdIdx{})
	}
}

func setRepresentations(ctx context.Context, g *api.CmdIDGroup, drawOrClearCmds api.Spans) {
	data, _ := g.UserData.(*CmdGroupData)
	if data == nil {
		data = &CmdGroupData{Representation: api.CmdNoID}
		g.UserData = data
	}
	if data.Representation == api.CmdNoID {
		if s, c := interval.Intersect(drawOrClearCmds, g.Bounds().Span()); c > 0 {
			data.Representation = drawOrClearCmds[s+c-1].Bounds().Start
		} else {
			data.Representation = g.Range.Last()
		}
	}

	for _, s := range g.Spans {
		if subgroup, ok := s.(*api.CmdIDGroup); ok {
			setRepresentations(ctx, subgroup, drawOrClearCmds)
		}
	}
}

func getOpenGLAliasRepresentation(indices []uint64, item *api.CmdIDGroup, cmdTree *commandTree) api.CmdID {
	aliasRepId := api.CmdNoID
	if item.Name == "OpenGL ES Commands" {
		parentIndices := indices[:len(indices)-1]
		pItem, _ := cmdTree.index(parentIndices)
		parentItem := pItem.(api.CmdIDGroup)
		aliasRepId = parentItem.Range.Last() - 1
	} else if strings.HasPrefix(item.Name, "glDraw") || strings.HasPrefix(item.Name, "glMultiDraw") || strings.HasPrefix(item.Name, "glClear(") {
		pindices := indices[:len(indices)-1]
		pItem, _ := cmdTree.index(pindices)
		if parentItem, ok := pItem.(api.CmdIDGroup); ok {
			if parentItem.Name == "OpenGL ES Commands" {
				grandparentIndices := indices[:len(indices)-2]
				gpItem, _ := cmdTree.index(grandparentIndices)
				if gparentItem, ok := gpItem.(api.CmdIDGroup); ok {
					aliasRepId = gparentItem.Range.Last() - 1
				}
			}
		}
	}
	return aliasRepId
}
