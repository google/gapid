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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
)

type dumpShadersVerb struct{ DumpShadersFlags }

func init() {
	verb := &dumpShadersVerb{
		DumpShadersFlags{
			Atom: -1,
		},
	}
	app.AddVerb(&app.Verb{
		Name:      "dump_resources",
		ShortHelp: "Dump all shaders at a particular atom from a .gfxtrace",
		Auto:      verb,
	})
}

func (verb *dumpShadersVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("Could not find capture file '%s': %v", flags.Arg(0), err)
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return fmt.Errorf("Failed to connect to the GAPIS server: %v", err)
	}
	defer client.Close()

	capture, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return fmt.Errorf("Failed to load the capture file '%v': %v", filepath, err)
	}

	boxedResources, err := client.Get(ctx, capture.Resources().Path())
	if err != nil {
		return fmt.Errorf("Could not find the capture's resources: %v", err)
	}
	resources := boxedResources.(*service.Resources)

	if verb.Atom == -1 {
		boxedAtoms, err := client.Get(ctx, capture.Commands().Path())
		if err != nil {
			return fmt.Errorf("Failed to acquire the capture's atoms: %v", err)
		}
		atoms := boxedAtoms.(*atom.List).Atoms
		verb.Atom = len(atoms) - 1
	}

	for _, types := range resources.GetTypes() {
		if types.Type == gfxapi.ResourceType_ShaderResource {
			for _, v := range types.GetResources() {

				resourcePath := capture.Commands().Index(uint64(verb.Atom)).ResourceAfter(v.GetId())
				shaderData, err := client.Get(ctx, resourcePath.Path())
				if err != nil {
					fmt.Printf("Could not get data for shader: %v %v\n", v, err)
					continue
				}
				shaderSource := shaderData.(*gfxapi.Shader).GetSource()

				f, err := os.Create(v.GetHandle())
				if err != nil {
					fmt.Printf("Could open file to write %s %v", v.GetHandle(), err)
					continue
				}
				defer f.Close()
				f.WriteString(shaderSource)
			}
		}
	}

	return nil
}
