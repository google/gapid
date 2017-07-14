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
	"net/url"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/stash"
	_ "github.com/google/gapid/test/robot/stash/grpc"
	_ "github.com/google/gapid/test/robot/stash/local"
)

var (
	stashAddr          = ""
	defaultStash       = "."
	defaultStashServer = "//localhost:8091"
)

func init() {
	flag.StringVar(&stashAddr, "stash", "", "The address of the stash")
}

func main() {
	app.ShortHelp = "Stash is a command line tool for interacting with gapid stash servers."
	app.Run(app.VerbMain)
}

type storeTask func(context.Context, *stash.Client) error

func withStore(ctx context.Context, isServer bool, task storeTask) error {
	if stashAddr == "" {
		if isServer {
			stashAddr = defaultStash
		} else {
			stashAddr = defaultStashServer
		}
	}
	stashURL, err := url.Parse(stashAddr)
	if err != nil {
		return log.Errf(ctx, err, "Invalid shash address %s", stashAddr)
	}
	client, err := stash.Dial(ctx, stashURL)
	if err != nil {
		return err
	}
	defer client.Close()
	return task(ctx, client)
}
