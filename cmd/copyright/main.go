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

// copyright is a tool to maintain copyright headers.
package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/copyright"
)

var (
	noactions = flag.Bool("n", false,
		"don't perform any actions, just print information")
	noold = flag.Bool("o", false,
		"don't update the copyright if it's just old")
)

func main() {
	app.ShortHelp = "copyright finds and fixes bad copyright headers."
	app.UsageFooter = `
The search is rooted at the current working directory.
It will attempt to fix incorrect copyright headers unless you specify the
-n flag, and will only replace copyright headers patterns that it knows about.
For unknown comments it will just prepend the correct copyright header.
It will not touch file extensions that it does not know the header type for.
`
	app.Run(run)
}

func update(ctx log.Context, path string, reason string, header string, body []byte) error {
	ctx.Notice().Logf("Copyright on %s was %s", path, reason)
	if *noactions {
		return nil
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(header)
	if err != nil {
		return err
	}
	_, err = file.Write(body)
	if err != nil {
		return err
	}
	return nil
}

func run(ctx log.Context) error {
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "third_party" || info.Name() == "vendor" || info.Name() == "external" {
					return filepath.SkipDir
				}
				return nil
			}
			l := copyright.FindExtension(path)
			if l == nil {
				return nil
			}
			file, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			if i := l.MatchCurrent(file); i > 0 {
				return nil
			}
			if i := copyright.MatchExternal(file); i > 0 {
				return nil
			}
			if i := copyright.MatchGenerated(file); i > 0 {
				return nil
			}
			if i := l.MatchOld(file); i > 0 {
				if !*noold {
					return update(ctx, path, "out of date", l.Emit, file[i:])
				}
				return nil
			}
			if i := copyright.MatchNormal(file); i > 0 {
				return update(ctx, path, "invalid", l.Emit, file[i:])
			}
			return update(ctx, path, "missing", l.Emit, file)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
