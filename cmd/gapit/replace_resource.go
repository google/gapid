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

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
)

type replaceResourceVerb struct{ ReplaceResourceFlags }

func init() {
	verb := &replaceResourceVerb{
		ReplaceResourceFlags{
			At:              -1,
			OutputTraceFile: "newcapture.gfxtrace",
		},
	}
	app.AddVerb(&app.Verb{
		Name:      "replace_resource",
		ShortHelp: "Produce a new trace with the given resource replaced at the given command",
		Action:    verb,
	})
}

func (verb *replaceResourceVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	if verb.Handle == "" {
		app.Usage(ctx, "-handle argument is required")
		return nil
	}

	if verb.ResourcePath == "" {
		app.Usage(ctx, "-resourcepath argument is required")
		return nil
	}

	captureFilepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return log.Errf(ctx, err, "Could not find capture file '%s'", flags.Arg(0))
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	capture, err := client.LoadCapture(ctx, captureFilepath)
	if err != nil {
		return log.Errf(ctx, err, "Failed to load the capture file '%v'", captureFilepath)
	}

	boxedResources, err := client.Get(ctx, capture.Resources().Path())
	if err != nil {
		return log.Err(ctx, err, "Could not find the capture's resources")
	}
	resources := boxedResources.(*service.Resources)

	if verb.At == -1 {
		boxedCapture, err := client.Get(ctx, capture.Path())
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = int(boxedCapture.(*service.Capture).NumCommands) - 1
	}

	for _, types := range resources.GetTypes() {
		if types.Type == api.ResourceType_ShaderResource {
			var matchedResource *service.Resource
			for _, v := range types.GetResources() {
				if strings.Contains(v.GetHandle(), verb.Handle) {
					if matchedResource != nil {
						return fmt.Errorf("Multiple resources matched: %s, %s", matchedResource.GetHandle(), v.GetHandle())
					}
					matchedResource = v
				}
			}
			resourcePath := capture.Command(uint64(verb.At)).ResourceAfter(matchedResource.Id)
			newResourceBytes, err := ioutil.ReadFile(verb.ResourcePath)
			if err != nil {
				return log.Errf(ctx, err, "Could not read resource file %s", verb.ResourcePath)
			}

			newResourceData := api.NewResourceData(&api.Shader{Type: api.ShaderType_SpirvBinary, Source: string(newResourceBytes)})
			newResourcePath, err := client.Set(ctx, resourcePath.Path(), newResourceData)
			if err != nil {
				return log.Errf(ctx, err, "Could not update data for shader: %v", matchedResource)
			}
			newCapture := newResourcePath.GetResourceData().GetAfter().GetCapture()
			newCaptureFilepath, err := filepath.Abs(verb.OutputTraceFile)
			err = client.SaveCapture(ctx, newCapture, newCaptureFilepath)

			log.I(ctx, "Capture written to: %v", newCaptureFilepath)
			return nil
		}
	}

	return fmt.Errorf("Failed to find the resource with the handle %s", verb.Handle)
}
