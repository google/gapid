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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"

	common "protos/perfetto/common"
)

type queryVerb struct {
}

func init() {
	verb := &queryVerb{}
	app.AddVerb(&app.Verb{
		Name:      "query",
		ShortHelp: "Queries the Perfetto service state",
		Action:    verb,
	})
}

func (verb *queryVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	ctx = setupContext(ctx)
	time.Sleep(2 * time.Second) // Give the producers some time to register.

	for _, d := range bind.GetRegistry(ctx).Devices() {
		if !d.SupportsPerfetto(ctx) {
			log.I(ctx, "Device %s doesn't support Perfetto", d)
			continue
		}

		c, err := connectToPerfetto(ctx, d)
		if err != nil {
			return err
		}
		defer c.Close(ctx)

		if err := c.Query(ctx, func(r *common.TracingServiceState) error {
			jsonBytes, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				fmt.Printf("%v\n", log.Err(ctx, err, "Couldn't marshal response to JSON"))
				return err
			}
			fmt.Println(string(jsonBytes))
			return nil
		}); err != nil {
			log.E(ctx, "Failed to query Perfetto: %s", err)
			return err
		}
	}
	return nil
}
