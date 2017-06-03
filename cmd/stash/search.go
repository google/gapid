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
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/search/script"
	"github.com/google/gapid/test/robot/stash"
)

func init() {
	verb := &app.Verb{
		Name:      "search",
		ShortHelp: "Prints information about stash entries",
		Action:    &infoVerb{},
	}
	app.AddVerb(verb)
}

type infoVerb struct {
	Monitor bool `help:"Monitor for changes"`
}

func (v *infoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	return withStore(ctx, false, func(ctx context.Context, client *stash.Client) error {
		return getInfo(ctx, client, strings.Join(flags.Args(), " "), v.Monitor)
	})
}

func getInfo(ctx context.Context, client *stash.Client, expression string, monitor bool) error {
	expr, err := script.Parse(ctx, expression)
	if err != nil {
		return log.Err(ctx, err, "Malformed search query")
	}
	query := expr.Query()
	query.Monitor = monitor
	err = client.Search(ctx, query, func(ctx context.Context, entry *stash.Entity) error {
		log.I(ctx, "%s", entry)
		return nil
	})
	if err == nil && monitor {
		os.Stdin.Read([]byte{0})
	}
	return err
}
