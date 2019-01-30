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

// The gapid command launches the GAPID UI. It looks for the JVM (bundled or
// from the system), the GAPIC JAR (bundled or from the build output) and
// launches GAPIC with the correct JVM flags and environment variables.
package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	log "github.com/google/gapid/core/log"
	_ "github.com/google/gapid/gapis/api/gles"
	_ "github.com/google/gapid/gapis/api/gvr"
	_ "github.com/google/gapid/gapis/api/vulkan"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve/initialcmds"
)

var (
	path      = flag.String("file", "capture.gfxtrace", "The capture file to linearize")
	output    = flag.String("out", "capture.linear.gfxtrace", "The output file")
	nCommands = flag.Int("num_commands", -1, "How many commands from the original trace should be included. -1 for all.")
)

func main() {
	app.ShortHelp = "linearize_trace converts a mid-execution capture to a linear trace"
	app.Name = "linearize_trace"
	app.Run(run)
}

func run(ctx context.Context) error {

	ctx = database.Put(ctx, database.NewInMemory(ctx))

	name := filepath.Base(*path)

	p, err := capture.Import(ctx, name, &capture.File{Path: *path})
	if err != nil {
		return err
	}
	// Ensure the capture can be read by resolving it now.
	capt, err := capture.ResolveGraphicsFromPath(ctx, p)
	if err != nil {
		return err
	}

	initialCmds, _, err := initialcmds.InitialCommands(ctx, p)
	if err != nil {
		return err
	}

	log.I(ctx, "Generated %v initial commands", len(initialCmds))

	if *nCommands < 0 {
		if *nCommands == -1 {
			capt.Commands = append(initialCmds, capt.Commands...)
		} else {
			return log.Errf(ctx, nil, "Invalid number of commands requested: %d", *nCommands)
		}
	} else {
		if *nCommands <= len(capt.Commands) {
			capt.Commands = append(initialCmds, capt.Commands[0:*nCommands]...)
		} else {
			return log.Errf(ctx, nil, "Number of commands requested: %d exceeds the total number of commands in the original trace: %d", len(capt.Commands), *nCommands)
		}
	}

	capt.InitialState = nil

	f, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = capt.Export(ctx, f); err != nil {
		return err
	}
	log.I(ctx, "Capture written to: %v", *output)

	return nil
}
