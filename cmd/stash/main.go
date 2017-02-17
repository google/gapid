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
	"flag"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/stash"
	_ "github.com/google/gapid/core/data/stash/grpc"
	_ "github.com/google/gapid/core/data/stash/local"
	"github.com/google/gapid/core/log"
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
	app.Version = app.VersionSpec{Major: 0, Minor: 1}
	app.Run(app.VerbMain)
}

type storeTask func(log.Context, *stash.Client) error

func withStore(ctx log.Context, isServer bool, task storeTask) error {
	if stashAddr == "" {
		if isServer {
			stashAddr = defaultStash
		} else {
			stashAddr = defaultStashServer
		}
	}
	client, err := stash.Dial(ctx, stashAddr)
	if err != nil {
		return err
	}
	defer client.Close()
	return task(ctx, client)
}
