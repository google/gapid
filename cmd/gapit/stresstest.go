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
	"math/rand"
	"sync"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type stresstestVerb struct{ StressTestFlags }

func init() {
	app.AddVerb(&app.Verb{
		Name:      "stress-test",
		ShortHelp: "Performs evil things on GAPIS to try to break it",
		Action:    &stresstestVerb{},
	})
}

func (verb *stresstestVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, c, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	boxedCapture, err := client.Get(ctx, c.Path(), nil)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture")
	}
	count := int(boxedCapture.(*service.Capture).NumCommands)

	wg := sync.WaitGroup{}

	for l := 0; l < 10; l++ {
		for i := 0; i < 10000; i++ {
			at := uint64(rand.Intn(count - 1))
			duration := time.Second + time.Duration(rand.Intn(int(time.Second*10)))
			wg.Add(1)

			const (
				getStateAfter = iota
				getMesh
				getCount
			)
			method := rand.Intn(getCount)

			crash.Go(func() {
				defer wg.Done()
				ctx, _ := task.WithTimeout(ctx, duration)

				switch method {
				case getStateAfter:
					boxedTree, err := client.Get(ctx, c.Command(at).StateAfter().Tree().Path(), nil)
					if err == nil {
						tree := boxedTree.(*service.StateTree)
						client.Get(ctx, tree.Root.Path(), nil)
					}

				case getMesh:
					boxedMesh, err := client.Get(ctx, c.Command(at).Mesh(path.NewMeshOptions(true)).Path(), nil)
					if err == nil {
						mesh := boxedMesh.(*api.Mesh)
						_ = mesh
					}
				}
			})
		}
		wg.Wait()
	}

	return nil
}
