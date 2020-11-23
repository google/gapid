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

package dependencygraph2

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/service/path"
)

var (
	dCE2Counter         = benchmark.Duration("DCE2")
	dCE2CmdDeadCounter  = benchmark.Integer("DCE2.cmd.dead")
	dCE2CmdLiveCounter  = benchmark.Integer("DCE2.cmd.live")
	dCE2DrawDeadCounter = benchmark.Integer("DCE2.draw.dead")
	dCE2DrawLiveCounter = benchmark.Integer("DCE2.draw.live")
	dCE2DataDeadCounter = benchmark.Integer("DCE2.data.dead")
	dCE2DataLiveCounter = benchmark.Integer("DCE2.data.live")
)

// DCECapture returns a new capture containing only the requested commands and their dependencies.
func DCECapture(ctx context.Context, name string, p *path.Capture, requestedCmds []*path.Command) (*path.Capture, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	ctx = log.Enter(ctx, "DCECapture")

	cfg := DependencyGraphConfig{
		MergeSubCmdNodes:       !config.DeadSubCmdElimination,
		IncludeInitialCommands: false,
	}
	graph, err := GetDependencyGraph(ctx, p, cfg)
	if err != nil {
		return nil, fmt.Errorf("Could not build dependency graph for DCE: %v", err)
	}
	builder := NewDCEBuilder(graph)
	for _, cmd := range requestedCmds {
		id := cmd.Indices[0]
		log.D(ctx, "Requested (%d) %v\n", id, c.Commands[id])
		err := builder.Request(ctx, api.SubCmdIdx(cmd.Indices))
		if err != nil {
			return nil, err
		}
	}
	builder.Build(ctx)

	if gc, err := capture.NewGraphicsCapture(ctx, name, c.Header, c.InitialState, builder.LiveCmds()); err != nil {
		return nil, err
	} else {
		return capture.New(ctx, gc)
	}
}

// DCEBuilder tracks the data necessary to perform dead-command-eliminition on a capture
type DCEBuilder struct {
	graph            DependencyGraph
	requestedNodes   []NodeID
	isLive           []bool
	liveCmds         []api.Cmd
	origCmdIDs       []api.CmdID
	liveCmdIDs       map[api.CmdID]api.CmdID
	numLiveInitCmds  int
	orphanObs        []ObsNode
	numDead, numLive int
	deadMem, liveMem uint64
}

func keepCommandAlive(builder *DCEBuilder, id api.CmdID) {
	nodeID := builder.graph.GetCmdNodeID(id, api.SubCmdIdx{})
	if nodeID != NodeNoID && !builder.isLive[nodeID] {
		builder.isLive[nodeID] = true
		builder.requestedNodes = append(builder.requestedNodes, nodeID)
	}
}

// NewDCEBuilder creates a new DCEBuiler using the specified dependency graph
func NewDCEBuilder(graph DependencyGraph) *DCEBuilder {
	b := &DCEBuilder{
		graph:      graph,
		isLive:     make([]bool, graph.NumNodes()),
		liveCmds:   make([]api.Cmd, 0, len(graph.Capture().Commands)),
		origCmdIDs: make([]api.CmdID, 0, len(graph.Capture().Commands)),
		liveCmdIDs: make(map[api.CmdID]api.CmdID),
	}
	for i := 0; i < b.graph.NumInitialCommands(); i++ {
		cmdID := api.CmdID(i).Derived()
		cmd := b.graph.GetCommand(cmdID)
		if cmd.Alive() || config.AllInitialCommandsLive {
			keepCommandAlive(b, cmdID)
		}
	}
	for i, cmd := range b.graph.Capture().Commands {
		if cmd.Alive() {
			keepCommandAlive(b, (api.CmdID)(i))
		}
	}
	for _, cmdID := range b.graph.GetUnopenedForwardDependencies() {
		keepCommandAlive(b, cmdID)
	}
	return b
}

// LiveCmdID maps CmdIDs from the original capture to CmdIDs in within the live commands.
// If the old CmdID refers to a dead command, the returned command will refer to the next live command; if there is no next live command, api.CmdNoID is returned.
func (b *DCEBuilder) LiveCmdID(oldCmdID api.CmdID) api.CmdID {
	liveCmdID, ok := b.liveCmdIDs[oldCmdID]
	if !ok {
		return api.CmdNoID
	}
	return liveCmdID
}

// OriginalCmdIDs maps a live CmdID to the CmdID of the corresponding command in the original capture
func (b *DCEBuilder) OriginalCmdID(liveCmdID api.CmdID) api.CmdID {
	if liveCmdID.IsReal() {
		liveCmdID += api.CmdID(b.numLiveInitCmds)
	} else {
		liveCmdID = liveCmdID.Real()
	}
	return b.origCmdIDs[liveCmdID]
}

// NumLiveInitiCmds returns the number of live commands which are initial commands.
// (Initial commands are generated commands to recreate the initial state).
func (b *DCEBuilder) NumLiveInitCmds() int {
	return b.numLiveInitCmds
}

// LiveCmds returns the live commands
func (b *DCEBuilder) LiveCmds() []api.Cmd {
	return b.liveCmds
}

// Build runs the dead-code-elimination.
// The subcommands specified in cmds are marked alive, along with their transitive dependencies.
func (b *DCEBuilder) Build(ctx context.Context) error {
	ctx = log.Enter(ctx, "DCEBuilder.Build")

	t0 := dCE2Counter.Start()
	// Mark all the transitive dependencies of the live nodes as also being alive.
	b.markDependencies()
	dCE2Counter.Stop(t0)

	b.buildLiveCmds(ctx)

	b.LogStats(ctx)
	if config.DebugDeadCodeElimination {
		b.printAllCmds(ctx)
	}
	return nil
}

func (b *DCEBuilder) LogStats(ctx context.Context) {
	numCmd := len(b.graph.Capture().Commands) + b.graph.NumInitialCommands()
	dCE2CmdDeadCounter.Add(int64(b.numDead))
	dCE2CmdLiveCounter.Add(int64(b.numLive))
	dCE2DataDeadCounter.Add(int64(b.deadMem))
	dCE2DataLiveCounter.Add(int64(b.liveMem))
	log.I(ctx, "DCE2: dead: %v%% %v cmds %v MB, live: %v%% %v cmds %v MB, time: %v",
		100*b.numDead/numCmd, b.numDead, b.deadMem/1024/1024,
		100*b.numLive/numCmd, b.numLive, b.liveMem/1024/1024,
		dCE2Counter.Get())
}

// Mark as alive all the transitive dependencies of live nodes.
// This is just BFS.
func (b *DCEBuilder) markDependencies() {
	// TODO: See if a more efficient queue is necessary
	queue := make([]NodeID, len(b.requestedNodes))
	copy(queue, b.requestedNodes)
	for len(queue) > 0 {
		src := queue[0]
		queue = queue[1:]
		b.graph.ForeachDependencyFrom(src, func(tgt NodeID) error {
			if !b.isLive[tgt] {
				b.isLive[tgt] = true
				queue = append(queue, tgt)
			}
			return nil
		})
	}
}

func (b *DCEBuilder) printAllCmds(ctx context.Context) {
	b.graph.ForeachNode(func(nodeID NodeID, node Node) error {
		alive := b.isLive[nodeID]
		status := "ALIVE"
		if !alive {
			status = "DEAD "
		}
		if cmdNode, ok := node.(CmdNode); ok && len(cmdNode.Index) == 1 {
			cmdID := api.CmdID(cmdNode.Index[0])
			cmd := b.graph.GetCommand(cmdID)
			log.D(ctx, "[ %s ] (%v / %v) %v\n", status, cmdID, b.LiveCmdID(cmdID), cmd)
		} else if obsNode, ok := node.(ObsNode); ok {
			cmdStatus := "DEAD "
			if obsNode.CmdID != api.CmdNoID {
				ownerIdx := api.SubCmdIdx{uint64(obsNode.CmdID)}
				ownerNodeID := b.graph.GetCmdNodeID(api.CmdID(ownerIdx[0]), ownerIdx[1:])
				if b.isLive[ownerNodeID] {
					cmdStatus = "ALIVE"
				}
			}
			liveCmdID := b.LiveCmdID(obsNode.CmdID)
			log.D(ctx, "[ %s ] (%v [%s] / %v) Range: %v  Pool: %v  ID: %v\n", status, obsNode.CmdID, cmdStatus, liveCmdID, obsNode.CmdObservation.Range, obsNode.CmdObservation.Pool, obsNode.CmdObservation.ID)
		}
		return nil
	})
}

// Builds liveCmds, containing the commands marked alive.
// These commands may be cloned and modified from the commands in the original capture.
// Live memory observations associated with dead commands are moved to the next live command.
func (b *DCEBuilder) buildLiveCmds(ctx context.Context) {
	// Process each node in chronological order
	// (see DependencyGraph.ForeachNode for clarification).
	b.graph.ForeachNode(func(nodeID NodeID, node Node) error {
		alive := b.isLive[nodeID]
		if cmdNode, ok := node.(CmdNode); ok {
			if len(cmdNode.Index) > 1 {
				return nil
			}
			cmdID := api.CmdID(cmdNode.Index[0])
			cmd := b.graph.GetCommand(cmdID)
			if alive {
				b.numLive++
				b.processLiveCmd(ctx, cmdID, cmd)
			} else {
				b.numDead++
			}
		} else if obsNode, ok := node.(ObsNode); ok {
			if alive {
				b.liveMem += obsNode.CmdObservation.Range.Size
				b.processLiveObs(ctx, obsNode)
			} else {
				b.deadMem += obsNode.CmdObservation.Range.Size
			}
		}
		return nil
	})
}

// Process a live command.
// This involves possibly cloning and modifying the command, and then adding it to liveCmds.
func (b *DCEBuilder) processLiveCmd(ctx context.Context, id api.CmdID, cmd api.Cmd) {
	// Helper to clone the command if we need to modify it.
	// Cloning is necessary before any modification because this command is
	// still used by another capture.
	// This clones the command at most one time, even if we make multiple modifications.
	isCloned := false
	cloneCmd := func() {
		if !isCloned {
			isCloned = true
			cmd = cmd.Clone()
		}
	}

	// Attach any orphan observations to this command
	if len(b.orphanObs) > 0 {
		cloneCmd()
		b.attachOrphanObs(ctx, id, cmd)
	}

	// compute the cmdID within the live commands
	liveCmdID := api.CmdID(len(b.liveCmds))
	if id.IsReal() {
		liveCmdID -= api.CmdID(b.numLiveInitCmds)
	} else {
		liveCmdID = liveCmdID.Derived()
		b.numLiveInitCmds++
	}

	b.liveCmdIDs[id] = liveCmdID
	b.liveCmds = append(b.liveCmds, cmd)
	b.origCmdIDs = append(b.origCmdIDs, id)
}

// Adds the current orphan observations as read observations of the specified command.
func (b *DCEBuilder) attachOrphanObs(ctx context.Context, id api.CmdID, cmd api.Cmd) {
	extras := make(api.CmdExtras, len(cmd.Extras().All()))
	copy(extras, cmd.Extras().All())
	obs := extras.Observations()
	oldReads := []api.CmdObservation{}
	if obs == nil {
		obs = &api.CmdObservations{
			Reads: make([]api.CmdObservation, 0, len(b.orphanObs)),
		}
		extras.Add(obs)
	} else {
		oldReads = obs.Reads
		oldObs := obs
		obs = &api.CmdObservations{
			Reads:  make([]api.CmdObservation, 0, len(b.orphanObs)+len(oldReads)),
			Writes: make([]api.CmdObservation, len(oldObs.Writes)),
		}
		copy(obs.Writes, oldObs.Writes)
		extras.Replace(oldObs, obs)
	}

	for _, o := range b.orphanObs {
		obs.Reads = append(obs.Reads, o.CmdObservation)
		if config.DebugDeadCodeElimination {
			cmdObservations := b.graph.GetCommand(o.CmdID).Extras().Observations()
			var cmdObs api.CmdObservation
			if o.IsWrite {
				cmdObs = cmdObservations.Writes[o.Index]
			} else {
				cmdObs = cmdObservations.Reads[o.Index]
			}
			log.D(ctx, "Adding orphan obs: [%v] %v\n", id, o, cmdObs)
		}
	}
	b.orphanObs = b.orphanObs[:0]
	obs.Reads = append(obs.Reads, oldReads...)
	*cmd.Extras() = extras
}

// Process a live memory observation.
// If this observation is attached to a non-live command, it will be saved into
// `orphanObs` to be attached to the next live command.
func (b *DCEBuilder) processLiveObs(ctx context.Context, obs ObsNode) {
	if obs.CmdID == api.CmdNoID {
		return
	}
	cmdNode := b.graph.GetCmdNodeID(obs.CmdID, api.SubCmdIdx{})
	if !b.isLive[cmdNode] {
		b.orphanObs = append(b.orphanObs, obs)
	}
}

// Request added a requsted command or subcommand, represented by its full
// command index, to the DCE.
func (b *DCEBuilder) Request(ctx context.Context, fci api.SubCmdIdx) error {
	if config.DebugDeadCodeElimination {
		log.D(ctx, "Requesting [%v] %v", fci[0], b.graph.GetCommand(api.CmdID(fci[0])))
	}
	nodeID := b.graph.GetCmdNodeID((api.CmdID)(fci[0]), fci[1:])
	if nodeID == NodeNoID {
		return fmt.Errorf("Requested dependencies of cmd not in graph: %v", fci)
	}
	if !b.isLive[nodeID] {
		b.isLive[nodeID] = true
		b.requestedNodes = append(b.requestedNodes, nodeID)
	}
	return nil
}

// Transform is to comform the interface of Transformer, but does not accept
// any input.
func (*DCEBuilder) Transform(ctx context.Context, id api.CmdID, c api.Cmd, out transform.Writer) error {
	panic(fmt.Errorf("This transform does not accept input commands"))
}

// Flush is to comform the interface of Transformer. Flush performs DCE, and
// sends the live commands to the writer
func (b *DCEBuilder) Flush(ctx context.Context, out transform.Writer) error {
	b.Build(ctx)
	for i, cmd := range b.LiveCmds() {
		liveCmdID := api.CmdID(i)
		if i < b.numLiveInitCmds {
			liveCmdID = liveCmdID.Derived()
		} else {
			liveCmdID -= api.CmdID(b.numLiveInitCmds)
		}
		cmdID := b.OriginalCmdID(liveCmdID)
		if config.DebugDeadCodeElimination {
			log.D(ctx, "Flushing [%v / %v] %v", cmdID, liveCmdID, cmd)
		}
		if err := out.MutateAndWrite(ctx, cmdID, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (b *DCEBuilder) PreLoop(ctx context.Context, out transform.Writer)  {}
func (b *DCEBuilder) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *DCEBuilder) BuffersCommands() bool                              { return false }
