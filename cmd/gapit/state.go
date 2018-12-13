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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type stateVerb struct{ StateFlags }

func init() {
	verb := &stateVerb{
		StateFlags{
			At:     flags.U64Slice{},
			Depth:  -1,
			Filter: flags.StringSlice{},
		},
	}

	app.AddVerb(&app.Verb{
		Name:      "state",
		ShortHelp: "Prints the state tree for a point in a .gfxtrace file",
		Action:    verb,
	})
}

func (verb *stateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, c, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	if len(verb.At) == 0 {
		boxedCapture, err := client.Get(ctx, c.Path(), nil)
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = []uint64{uint64(boxedCapture.(*service.Capture).NumCommands) - 1}
	}

	boxedTree, err := client.Get(ctx, c.Command(uint64(verb.At[0]), verb.At[1:]...).StateAfter().Tree().Path(), nil)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the command tree")
	}

	tree := boxedTree.(*service.StateTree)

	return traverseStateTree(ctx, client, tree.Root, verb.Depth, verb.Filter, func(n *service.StateTreeNode, prefix string) error {
		name := n.Name + ":"
		if n.Preview != nil {
			v := n.Preview.Get()
			if n.Constants != nil {
				constants, err := getConstantSet(ctx, client, n.Constants)
				if err != nil {
					return log.Err(ctx, err, "Couldn't fetch constant set")
				}
				v = constants.Sprint(v)
			}
			fmt.Fprintln(os.Stdout, prefix, name, v)
		} else {
			fmt.Fprintln(os.Stdout, prefix, name)
		}
		return nil
	}, "", true)
}

func traverseStateTree(
	ctx context.Context,
	c client.Client,
	p *path.StateTreeNode,
	depth int,
	filter flags.StringSlice,
	f func(n *service.StateTreeNode, prefix string) error,
	prefix string,
	last bool) error {

	if task.Stopped(ctx) {
		return task.StopReason(ctx)
	}

	boxedNode, err := c.Get(ctx, p.Path(), nil)
	if err != nil {
		return log.Errf(ctx, err, "Failed to load the node at: %v", p)
	}

	n := boxedNode.(*service.StateTreeNode)

	curPrefix := prefix
	if len(p.Indices) > 0 {
		if last {
			curPrefix += "└──"
		} else {
			curPrefix += "├──"
		}
	}

	nextFilter := filter
	if len(filter) != 0 &&
		(filter[0] != n.Name && filter[0] != "*") {
		return nil
	}
	if len(filter) != 0 {
		nextFilter = filter[1:]
	}

	if err := f(n, curPrefix); err != nil {
		return err
	}

	if last {
		prefix += "    "
	} else {
		prefix += "│   "
	}
	if depth != 0 {
		for i := uint64(0); i < n.NumChildren; i++ {
			err := traverseStateTree(ctx, c, p.Index(i), depth-1, nextFilter, f, prefix, i == n.NumChildren-1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
