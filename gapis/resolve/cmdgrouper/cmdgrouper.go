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

// Package cmdgrouper provides types for grouping together commands.
package cmdgrouper

import (
	"context"
	"fmt"

	"github.com/google/gapid/gapis/api"
)

// Group is the product of a Grouper.
type Group struct {
	Start    api.CmdID
	End      api.CmdID
	Name     string
	UserData interface{}
}

// Grouper is the interface implemented by types that build groups.
type Grouper interface {
	// Process considers the command for inclusion in the group.
	Process(context.Context, api.CmdID, api.Cmd, *api.GlobalState)
	// Build returns the groups built and resets the state of the grouper.
	Build(end api.CmdID) []Group
}

// RunPred is the predicate used by the Run grouper.
// Consecutive values returned by RunPred will be grouped together under the
// group with name.
type RunPred func(cmd api.Cmd, s *api.GlobalState) (value interface{}, name string)

// Run returns a grouper that groups commands together that form a run.
func Run(pred RunPred) Grouper {
	return &run{f: pred}
}

// run is a grouper that groups consecutive runs of commands
type run struct {
	f       func(cmd api.Cmd, s *api.GlobalState) (value interface{}, name string)
	start   api.CmdID
	current interface{}
	name    string
	out     []Group
}

func (g *run) Process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState) {
	val, name := g.f(cmd, s)
	if val != g.current {
		if g.current != nil {
			g.out = append(g.out, Group{g.start, id, g.name, nil})
		}
		g.start = id
	}
	g.current, g.name = val, name
}

func (g *run) Build(end api.CmdID) []Group {
	if g.current != nil && g.start != end {
		g.out = append(g.out, Group{g.start, end, g.name, nil})
	}
	out := g.out
	g.out, g.start, g.current, g.name = nil, 0, nil, ""
	return out
}

// Marker returns a grouper that groups based on user marker commands.
func Marker() Grouper {
	return &marker{}
}

type marker struct {
	stack []Group
	count int
	out   []Group
}

func (g *marker) Process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState) {
	flags := cmd.CmdFlags()
	if flags.IsPushUserMarker() {
		g.push(ctx, id, cmd, s)
	}
	if flags.IsPopUserMarker() && len(g.stack) > 0 {
		g.pop(id)
	}
}

func (g *marker) Build(end api.CmdID) []Group {
	for len(g.stack) > 0 {
		g.pop(end - 1)
	}
	out := g.out
	g.stack, g.count, g.out = nil, 0, nil
	return out
}

// push enters a group at the specified id.
// If the cmd implements api.Labeled then the group will use this label as the
// group name.
func (g *marker) push(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState) {
	var name string
	if l, ok := cmd.(api.Labeled); ok {
		name = l.Label(ctx, s)
	}
	if len(name) > 0 {
		g.stack = append(g.stack, Group{Start: id, Name: fmt.Sprintf("\"%s\"", name)})
	} else {
		g.stack = append(g.stack, Group{Start: id, Name: fmt.Sprintf("Marker %d", g.count)})
		g.count++
	}
}

// pop closes the group most recently entered.
func (g *marker) pop(id api.CmdID) {
	m := g.stack[len(g.stack)-1]
	m.End = id + 1 // +1 to include pop marker
	g.out = append(g.out, m)
	g.stack = g.stack[:len(g.stack)-1]
}
