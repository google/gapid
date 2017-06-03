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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/stash"
)

func init() {
	verb := &app.Verb{
		Name:      "upload",
		ShortHelp: "Upload a file to the stash",
		Action:    &uploadVerb{},
	}
	app.AddVerb(verb)
}

type uploadVerb struct{}

func (v *uploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() == 0 {
		app.Usage(ctx, "No files to upload given")
		return nil
	}
	return withStore(ctx, false, func(ctx context.Context, client *stash.Client) error {
		return sendFiles(ctx, client, flags.Args())
	})
}

func sendFiles(ctx context.Context, client *stash.Client, filenames []string) error {
	for _, partial := range filenames {
		id, err := client.UploadFile(ctx, file.Abs(partial))
		if err != nil {
			return log.Err(ctx, err, "Failed calling Upload")
		}
		log.I(ctx, "Uploaded %s as %s", partial, id)
	}
	return nil
}
