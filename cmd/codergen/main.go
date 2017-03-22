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

// The codergen command parses go code to automatically generate encoders and
// decoders for the structs it finds.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/copyright"
	"github.com/google/gapid/tools/codergen/generate"
	"github.com/google/gapid/tools/codergen/scan"
	"github.com/google/gapid/tools/codergen/template"
)

var (
	nowrite    = flag.Bool("n", false, "don't write the files")
	golang     = flag.Bool("go", false, "generate go code")
	java       = flag.String("java", "", "the path to generate files in")
	cpp        = flag.String("cpp", "", "the path to generate files in")
	workers    = flag.Int("workers", 15, "The numer of output workers to use")
	signatures = flag.String("signatures", "", "the signature file to generate")
	gopath     = flag.String("gopath", "", "the go path to use when looking up packages")
	base       = flag.String("base", "", "the base directory of the package scan")
)

func main() {
	app.ShortHelp = "codergen: A tool to generate coders for go structs."
	app.Run(run)
}

type errors struct {
	list []error
	mu   sync.Mutex
}

func (l *errors) Add(err error) {
	l.mu.Lock()
	l.list = append(l.list, err)
	l.mu.Unlock()
}

func (l *errors) String() string {
	return l.Error()
}

func (l *errors) Error() string {
	if len(l.list) > 0 {
		return l.list[0].Error()
	} else {
		return ""
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup, errs *errors, tasks chan generate.Generate) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := template.New()
		for task := range tasks {
			out := task.Output
			ctx := log.V{"out": out}.Bind(ctx)
			if *nowrite {
				task.Output = ""
			}
			changed, err := t.Generate(task)
			if err != nil {
				errs.Add(err)
			} else if changed {
				if *nowrite {
					log.I(ctx, "Not writing")
				} else {
					log.I(ctx, "Generated")
				}
			} else {
				log.I(ctx, "No change")
			}
		}
	}()
}

func run(ctx context.Context) error {
	wd := *base
	if wd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		wd = cwd
	}
	scanner := scan.New(ctx, wd, filepath.FromSlash(*gopath))
	log.I(ctx, "Scanning")
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, "./...")
	}
	for _, arg := range args {
		if err := scanner.Scan(ctx, arg); err != nil {
			return err
		}
	}
	log.I(ctx, "Processing")
	if err := scanner.Process(ctx); err != nil {
		return err
	}
	modules, err := generate.From(scanner)
	if err != nil {
		return err
	}
	log.I(ctx, "Generating")
	wg := sync.WaitGroup{}
	errs := errors{}
	gen := make(chan generate.Generate)
	for i := 0; i < *workers; i++ {
		worker(ctx, &wg, &errs, gen)
	}
	info := copyright.Info{Tool: scan.Tool, Year: "2015"}
	for _, m := range modules {
		if *golang {
			generate.Go(m, info, gen)
		}
		_, doJava := m.Directives["java.package"]
		if *java != "" && !m.IsTest && doJava {
			generate.Java(m, info, gen, *java)
		}
		_, doCpp := m.Directives["cpp"]
		if *cpp != "" && !m.IsTest && doCpp {
			generate.Cpp(m, info, gen, *cpp)
		}
	}
	close(gen)
	wg.Wait()
	if len(errs.list) > 0 {
		for _, err := range errs.list {
			fmt.Print(err)
		}
		return errs.list[0]
	}
	if *signatures != "" {
		f, err := os.Create(*signatures)
		if err != nil {
			return err
		}
		defer f.Close()
		generate.WriteAllSignatures(f, modules)
	}
	return nil
}
