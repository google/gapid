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
	"github.com/google/gapid/tools/codergen/template"
)

var (
	noactions = flag.Bool("n", false,
		"don't perform any actions, just print information")
	sectionfiles = flag.Bool("sectionfiles", false,
		"disable section processing, will delete whole files")
)

func main() {
	app.ShortHelp = "clean_generated finds and deletes generated source files."
	app.UsageFooter = `
The search is rooted at the current working directory.
It finds files with a known extension and a known generated file header comment.
If the -n flag is not specified, the file will then be removed.
`
	app.Version = app.VersionSpec{Major: 0, Minor: 1}
	app.Run(run)
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
			if copyright.FindExtension(filepath.Ext(path)) == nil {
				return nil
			}
			file, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			if copyright.MatchGenerated(file) == 0 {
				return nil
			}
			var sections []template.Section
			if !*sectionfiles {
				sections, err = template.SectionSplit(file)
				if err != nil {
					return err
				}
			}
			if len(sections) > 0 {
				ctx.Printf("rewrite %s", path)
				if !*noactions {
					out, err := os.Create(path)
					if err != nil {
						return err
					}
					for _, s := range sections {
						if s.Name == "" {
							// copy the non template section back to the file
							_, err = out.Write(s.Body)
							if err != nil {
								return err
							}
						} else {
							// copy the markers but drop the body
							_, err = out.Write(s.StartMarker)
							if err != nil {
								return err
							}
							_, err = out.Write(s.EndMarker)
							if err != nil {
								return err
							}
						}
					}
				}
			} else {
				ctx.Printf("rm %s", path)
				if !*noactions {
					os.Remove(path)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
