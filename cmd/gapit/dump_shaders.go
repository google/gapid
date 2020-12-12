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
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type dumpShadersVerb struct{ DumpShadersFlags }

func init() {
	verb := &dumpShadersVerb{
		DumpShadersFlags{
			At: -1,
		},
	}
	app.AddVerb(&app.Verb{
		Name:      "dump_resources",
		ShortHelp: "Dump all shaders at a particular command from a .gfxtrace",
		Action:    verb,
	})
}

func (verb *dumpShadersVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	device, err := getDevice(ctx, client, capture, verb.Gapir)
	if err != nil {
		return err
	}

	resolveConfig := path.ResolveConfig{ReplayDevice: device}

	boxedResources, err := client.Get(ctx, capture.Resources().Path(), &resolveConfig)
	if err != nil {
		return log.Err(ctx, err, "Could not find the capture's resources")
	}
	resources := boxedResources.(*service.Resources)

	if verb.At == -1 {
		boxedCapture, err := client.Get(ctx, capture.Path(), &resolveConfig)
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = int(boxedCapture.(*service.Capture).NumCommands) - 1
	}

	for _, types := range resources.GetTypes() {
		if types.Type == path.ResourceType_ShaderResource {
			for _, v := range types.GetResources() {
				if !v.ID.IsValid() {
					log.E(ctx, "Got resource with invalid ID!\n%+v", v)
					continue
				}
				resourcePath := capture.Command(uint64(verb.At)).ResourceAfter(v.ID)
				resourceData, err := client.Get(ctx, resourcePath.Path(), &resolveConfig)
				if err != nil {
					log.E(ctx, "Could not get data for shader: %v %v", v, err)
					continue
				}

				shaderSource := resourceData.(*api.ResourceData).GetShader().GetSource()
				shaderType := resourceData.(*api.ResourceData).GetShader().GetType()
				filename := file.SanitizePath(v.GetID().ID().String() + "." + shaderType.Extension())
				f, err := os.Create(filename)
				if err != nil {
					log.E(ctx, "Could open file to write %s %v", v.GetHandle(), err)
					continue
				}
				defer f.Close()
				f.WriteString(shaderSource)
			}
		}
	}

	return nil
}
