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
	"path/filepath"

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
			At: flags.U64Slice{},
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

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	filepath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": filepath}.Bind(ctx)
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
	}

	c, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture file")
	}

	if len(verb.At) == 0 {
		boxedCapture, err := client.Get(ctx, c.Path())
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = []uint64{uint64(boxedCapture.(*service.Capture).NumCommands) - 1}
	}

	boxedTree, err := client.Get(ctx, c.Command(uint64(verb.At[0]), verb.At[1:]...).StateAfter().Tree().Path())
	if err != nil {
		return log.Err(ctx, err, "Failed to load the command tree")
	}

	tree := boxedTree.(*service.StateTree)

	return traverseStateTree(ctx, client, tree.Root, func(n *service.StateTreeNode, prefix string) error {
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
	f func(n *service.StateTreeNode, prefix string) error,
	prefix string,
	last bool) error {

	if task.Stopped(ctx) {
		return task.StopReason(ctx)
	}

	boxedNode, err := c.Get(ctx, p.Path())
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

	if err := f(n, curPrefix); err != nil {
		return err
	}

	if last {
		prefix += "    "
	} else {
		prefix += "│   "
	}
	for i := uint64(0); i < n.NumChildren; i++ {
		err := traverseStateTree(ctx, c, p.Index(i), f, prefix, i == n.NumChildren-1)
		if err != nil {
			return err
		}
	}

	return nil
}
