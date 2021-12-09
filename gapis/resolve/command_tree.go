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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
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

func (t *commandTree) indices(cmdIdx []uint64, preferGroup bool) []uint64 {
	out := []uint64{}
	group := t.root

	for i := 0; i < len(cmdIdx); i++ {
		cmdId := api.CmdID(cmdIdx[i])
		idx := group.IndexOf(cmdId)
		out = append(out, idx)

		switch item := group.Index(idx).(type) {
		case api.CmdIDGroup:
			group = item
			if group.Bounds().Last() > cmdId {
				// We are looking for something inside this group, not the group itself, so go around again.
				i--
			}
		case api.SubCmdRoot:
			group = item.SubGroup
			// This SubCmdRoot may skip over some command indecies.
			i = len(item.Id) - 1
		default:
			return out
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
			ExpandByDefault:      true,
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

	// Build the command tree
	out := &commandTree{
		path: p,
		root: api.CmdIDGroup{
			Name:  "root",
			Range: api.CmdIDRange{End: api.CmdID(len(c.Commands))},
		},
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
	if p.GroupBySubmission && !p.Filter.GetSuppressHostCommands() {
		addContainingGroups(ctx, p, out, c.Commands,
			func(id api.CmdID, cmd api.Cmd, s *api.GlobalState, idx api.SubCmdIdx) bool {
				return cmd.CmdFlags().IsSubmission()
			},
			"Host Coordination")
	}

	// Now we have all the groups, we finally need to add the filtered commands.
	s := c.NewState(ctx)
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

			// In the current design, it's easier to flatten the tree after the fact,
			// rather than avoid creating the tree structure in the first place.
			// TODO(pmuetschard): re-design this to suit Vulkan and avoid this.
			if p.SuppressSubmitInfoNodes && cmd.CmdFlags().IsSubmission() {
				subr.SubGroup.Flatten()
			}

			return nil
		}

		out.root.AddCommand(id)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cluster the commands
	out.root.Cluster(uint64(p.MaxChildren), uint64(p.MaxNeighbours))

	// Set group representations.
	setRepresentations(ctx, &out.root)

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
	eofCommands := []int{}
	for idx, cmd := range cmds {
		if cmd.CmdFlags().IsEndOfFrame() {
			eofCommands = append(eofCommands, idx)
		}
	}

	// No end of frame commands in the trace, no grouping necessary.
	if len(eofCommands) == 0 {
		return
	}

	needIncompleteFrame := eofCommands[len(eofCommands)-1] < len(cmds)-1
	if len(eofCommands) == 1 && !needIncompleteFrame {
		// Only one frame, so the group would span the entire range. Don't bother adding it.
		return
	}

	frameCount := 0
	startFrame := 0
	for _, idx := range eofCommands {
		t.root.AddGroup(api.CmdID(startFrame), api.CmdID(idx+1), fmt.Sprintf("Frame %v", frameCount+1), []api.SubCmdIdx{})
		startFrame = idx + 1
		frameCount++
	}

	if p.AllowIncompleteFrame && needIncompleteFrame {
		t.root.AddGroup(api.CmdID(startFrame), api.CmdID(len(cmds)), fmt.Sprintf("[Incomplete] Frame", frameCount+1), []api.SubCmdIdx{})
	}
}

func setRepresentations(ctx context.Context, g *api.CmdIDGroup) {
	data, _ := g.UserData.(*CmdGroupData)
	if data == nil {
		data = &CmdGroupData{Representation: api.CmdNoID}
		g.UserData = data
	}
	if data.Representation == api.CmdNoID {
		data.Representation = g.Range.Last()
	}

	for _, s := range g.Spans {
		if subgroup, ok := s.(*api.CmdIDGroup); ok {
			setRepresentations(ctx, subgroup)
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
