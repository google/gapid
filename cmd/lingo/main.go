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

// The syntax command is used to rewrite recursive descent parser syntax declaration files.
package main

import (
	"context"
	"flag"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/test/robot/lingo/generator"
)

var (
	base string
)

func main() {
	app.ShortHelp = "syntax: A recursive descent parser generator."
	flag.StringVar(&base, "base", base, "don't update the copyright if it's just old")
	app.Run(run)
}

func run(ctx context.Context) error {
	args := flag.Args()
	if len(args) < 1 {
		app.Usage(ctx, "Expect at least one lingo file")
		return nil
	}
	return generator.RewriteFiles(ctx, base, args...)
}
