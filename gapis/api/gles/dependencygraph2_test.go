// Copyright (C) 2019 Google Inc.
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

package gles_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
)

// Test helpers

// testDependencyCmdName is used as a map key to easily express
// expected dependencies
type testDependencyCmdName struct {
	sourceCmdName, targetCmdName string
}

// getCommandName return the command name of a graph node
func getCommandName(graph dependencygraph2.DependencyGraph, nodeId dependencygraph2.NodeID) string {
	node := graph.GetNode(nodeId)
	if cmdNode, ok := node.(dependencygraph2.CmdNode); ok {
		cmdId := cmdNode.Index[0]
		if api.CmdID(cmdId).IsReal() {
			command := graph.GetCommand(api.CmdID(cmdId))
			return command.CmdName()
		}
	}
	return ""
}

// testCaptureAndGraph abstracts the boilerplate for creating a capture
// programmatically, check expected dependencies in its graph, and
// clean out
type testCaptureAndGraph struct {
	ctx   context.Context
	arena arena.Arena
	cb    gles.CommandBuilder
	cmds  []api.Cmd
}

// init runs the boilerplate up to being able to add commands to tc.cmds
func (tc *testCaptureAndGraph) init(t *testing.T) {
	tc.ctx = log.Testing(t)
	tc.ctx = bind.PutRegistry(tc.ctx, bind.NewRegistry())
	tc.ctx = database.Put(tc.ctx, database.NewInMemory(tc.ctx))
	tc.arena = arena.New()
	ctxHandle := memory.BytePtr(1)
	displayHandle := memory.BytePtr(2)
	surfaceHandle := memory.BytePtr(3)
	tc.cb = gles.CommandBuilder{Thread: 0, Arena: tc.arena}

	// Common prologue: make an EGL context
	tc.cmds = []api.Cmd{
		tc.cb.EglCreateContext(displayHandle, surfaceHandle, surfaceHandle, memory.Nullptr, ctxHandle),
		api.WithExtras(
			tc.cb.EglMakeCurrent(displayHandle, surfaceHandle, surfaceHandle, ctxHandle, 0),
			gles.NewStaticContextStateForTest(tc.arena), gles.NewDynamicContextStateForTest(tc.arena, 64, 64, false)),
	}
}

// terminate cleans up what need to be
func (tc *testCaptureAndGraph) terminate() {
	tc.arena.Dispose()
}

// checkDependenciesArePresent generate the graph and check if
// expected dependencies present
func (tc *testCaptureAndGraph) checkDependenciesArePresent(t *testing.T, expected, unexpected []testDependencyCmdName) {
	// Compute dependency graph
	header := &capture.Header{ABI: device.AndroidARM64v8a}
	cap, err := capture.NewGraphicsCapture(tc.ctx, tc.arena, t.Name(), header, nil, tc.cmds)
	if err != nil {
		panic(err)
	}
	capturePath, err := cap.Path(tc.ctx)
	if err != nil {
		panic(err)
	}
	tc.ctx = capture.Put(tc.ctx, capturePath)

	cfg := dependencygraph2.DependencyGraphConfig{
		MergeSubCmdNodes:       true,
		IncludeInitialCommands: false,
	}
	graph, err := dependencygraph2.GetDependencyGraph(tc.ctx, capturePath, cfg)
	if err != nil {
		panic(err)
	}

	// Check dependencies
	mapExpected := map[testDependencyCmdName]bool{}
	for _, dep := range expected {
		mapExpected[dep] = false
	}
	mapUnexpected := map[testDependencyCmdName]bool{}
	for _, dep := range unexpected {
		mapUnexpected[dep] = false
	}

	graph.ForeachDependency(
		func(src, tgt dependencygraph2.NodeID) error {
			srcCmdName := getCommandName(graph, src)
			tgtCmdName := getCommandName(graph, tgt)
			dep := testDependencyCmdName{srcCmdName, tgtCmdName}
			if _, ok := mapExpected[dep]; ok {
				mapExpected[dep] = true
			}
			if _, ok := mapUnexpected[dep]; ok {
				mapUnexpected[dep] = true
			}
			return nil
		})

	for dep, present := range mapExpected {
		assert.For(tc.ctx, fmt.Sprintf("Dependency: %v", dep)).ThatBoolean(present).IsTrue()
	}
	for dep, present := range mapUnexpected {
		assert.For(tc.ctx, fmt.Sprintf("Dependency: %v", dep)).ThatBoolean(present).IsFalse()
	}
}

// Actual tests

// glClear(GL_COLOR_BUFFER_BIT) depends on glClearColor()
func TestDependencyGlClearColor(t *testing.T) {
	var tc testCaptureAndGraph
	tc.init(t)
	defer tc.terminate()

	tc.cmds = append(tc.cmds,
		tc.cb.GlClearColor(1, 1, 1, 1),
		tc.cb.GlClearDepthf(1),
		tc.cb.GlClearStencil(1),
		tc.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT))

	expected := []testDependencyCmdName{
		testDependencyCmdName{"glClear", "glClearColor"},
	}

	unexpected := []testDependencyCmdName{
		testDependencyCmdName{"glClear", "glClearDepthf"},
		testDependencyCmdName{"glClear", "glClearStencil"},
	}

	tc.checkDependenciesArePresent(t, expected, unexpected)
}

// glClear(GL_DEPTH_BUFFER_BIT) depends on glClearDepthf()
func TestDependencyGlClearDepthf(t *testing.T) {
	var tc testCaptureAndGraph
	tc.init(t)
	defer tc.terminate()

	tc.cmds = append(tc.cmds,
		tc.cb.GlClearColor(1, 1, 1, 1),
		tc.cb.GlClearDepthf(1),
		tc.cb.GlClearStencil(1),
		tc.cb.GlClear(gles.GLbitfield_GL_DEPTH_BUFFER_BIT))

	expected := []testDependencyCmdName{
		testDependencyCmdName{"glClear", "glClearDepthf"},
	}

	unexpected := []testDependencyCmdName{
		testDependencyCmdName{"glClear", "glClearColor"},
		testDependencyCmdName{"glClear", "glClearStencil"},
	}

	tc.checkDependenciesArePresent(t, expected, unexpected)
}

// glClear(GL_STENCIL_BUFFER_BIT) depends on glClearStencil()
func TestDependencyGlClearStencil(t *testing.T) {
	var tc testCaptureAndGraph
	tc.init(t)
	defer tc.terminate()

	tc.cmds = append(tc.cmds,
		tc.cb.GlClearColor(1, 1, 1, 1),
		tc.cb.GlClearDepthf(1),
		tc.cb.GlClearStencil(1),
		tc.cb.GlClear(gles.GLbitfield_GL_STENCIL_BUFFER_BIT))

	expected := []testDependencyCmdName{
		testDependencyCmdName{"glClear", "glClearStencil"},
	}

	unexpected := []testDependencyCmdName{
		testDependencyCmdName{"glClear", "glClearColor"},
		testDependencyCmdName{"glClear", "glClearDepthf"},
	}

	tc.checkDependenciesArePresent(t, expected, unexpected)
}
