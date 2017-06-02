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

// The calc command implements a simple calculator.
// This is an example of using the lingo system to generate an ast, and
// then evaluating the ast afterwards.
// This is the most complete of the examples, and also the pattern that most
// parsers would follow.

package main

import (
	"context"
	"flag"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	_ "github.com/google/gapid/test/robot/lingo"
)

func main() {
	app.ShortHelp = "calc: A simple calculator."
	app.Run(run)
}

func run(ctx context.Context) error {
	args := flag.Args()
	if len(args) < 1 {
		app.Usage(ctx, "Missing expression")
		return nil
	}
	input := strings.Join(args, " ")
	value, err := Parse(ctx, "command_line", input)
	if err != nil {
		return err
	}
	log.I(ctx, "%v = %v\n", value, value.Eval())
	return nil
}
