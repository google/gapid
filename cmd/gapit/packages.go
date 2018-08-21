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
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type packagesVerb struct{ PackagesFlags }

func init() {
	app.AddVerb(&app.Verb{
		Name:      "packages",
		ShortHelp: "Prints information about packages installed on a device",
		Action: &packagesVerb{
			PackagesFlags{
				Icons:       false,
				IconDensity: 1.0,
			},
		},
	})
}

func (verb *packagesVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	devices, err := filterDevices(ctx, &verb.DeviceFlags, client)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		fmt.Fprintf(os.Stderr, "Cannot find device to get packages")
	}

	for _, p := range devices {
		o, err := client.Get(ctx, p.Path(), nil)
		if err != nil {
			fmt.Fprintf(os.Stdout, "%v\n", log.Err(ctx, err, "Couldn't resolve device"))
			continue
		}
		d := o.(*device.Instance)

		cfg, err := client.Get(ctx, (&path.DeviceTraceConfiguration{Device: p}).Path(), nil)
		if err != nil {
			fmt.Fprintf(os.Stdout, "%v\n", log.Err(ctx, err, "Couldn't get device config"))
			return err
		}
		c := cfg.(*service.DeviceTraceConfiguration)

		fmt.Fprintf(os.Stdout, "Device %v\n", d.Name)
		if err := verb.traversePackageTree(ctx, client, p, 0, c.PreferredRootUri, "", false); err != nil {
			fmt.Fprintf(os.Stdout, "%v\n", err)
			return err
		}
	}
	return err
}

func (verb *packagesVerb) traversePackageTree(
	ctx context.Context,
	c client.Client,
	d *path.Device,
	depth int,
	uri string,
	prefix string,
	last bool) error {
	if task.Stopped(ctx) {
		return task.StopReason(ctx)
	}

	node, err := c.TraceTargetTreeNode(ctx, &service.TraceTargetTreeNodeRequest{
		Device:  d,
		Uri:     uri,
		Density: float32(verb.IconDensity),
	})
	if err != nil {
		return log.Errf(ctx, err, "Failed to load the node at: %v", uri)
	}

	curPrefix := prefix
	if depth > 0 {
		if last {
			curPrefix += "└──"
		} else {
			curPrefix += "├──"
		}
	}

	suffix := ""
	if node.GetTraceUri() != "" {
		suffix = " ( " + node.GetTraceUri() + " )"
	}

	fmt.Fprintln(os.Stdout, curPrefix, node.Name, suffix)

	if depth > 0 {
		if last {
			prefix += "    "
		} else {
			prefix += "│   "
		}
	}

	for i, u := range node.ChildrenUris {
		err := verb.traversePackageTree(ctx, c, d, depth+1, u, prefix, i == (len(node.ChildrenUris)-1))
		if err != nil {
			return err
		}
	}

	return nil
}
