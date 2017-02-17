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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func notFoundErr(t string, sel string, candidates []string) error {
	var part1, part2 string
	if sel == "" {
		part1 = "not specified"
	} else {
		part1 = fmt.Sprintf("not found: %s", sel)
	}
	if len(candidates) == 0 {
		part2 = "None available."
	} else {
		part2 = fmt.Sprintf("Candidates: %s", strings.Join(candidates, ", "))
	}
	return fmt.Errorf("%s %s. %s", t, part1, part2)
}

func selectBenchmark(perfz *Perfz, name string) (*Benchmark, error) {
	var bench *Benchmark
	var ok bool
	candidates := []string{}
	for k := range perfz.Benchmarks {
		candidates = append(candidates, k)
	}

	if name != "" {
		bench, ok = perfz.Benchmarks[name]
	} else if len(candidates) == 1 {
		bench, ok = perfz.Benchmarks[candidates[0]], true
	}
	if !ok {
		return nil, notFoundErr("Benchmark", name, candidates)
	} else {
		return bench, nil
	}
}

func selectLink(perfz *Perfz, selector string, pattern *regexp.Regexp) (bench *Benchmark, link *Link, err error) {
	components := strings.Split(selector, ":")

	var benchmarkSel string
	if len(components) >= 1 {
		benchmarkSel = components[0]
	}
	var linkSel string
	if len(components) >= 2 {
		linkSel = components[1]
	}

	// Select
	bench, err = selectBenchmark(perfz, benchmarkSel)
	if err != nil {
		return nil, nil, err
	}

	// Select link
	ok := false
	candidates := []string{}
	for k := range bench.Links {
		if pattern == nil || pattern.MatchString(k) {
			candidates = append(candidates, k)
		}
	}
	if linkSel != "" {
		link, ok = bench.Links[linkSel]
	} else if len(candidates) == 1 {
		link, ok = bench.Links[candidates[0]], true
	}
	if !ok {
		return bench, nil, notFoundErr("Link", linkSel, candidates)
	}
	return bench, link, nil
}

func writeAllFn(fn string, fun func(io.Writer) error) error {
	var writer io.Writer
	if fn == "" {
		writer = ioutil.Discard
	} else if fn == "-" {
		writer = os.Stdout
	} else {
		f, err := os.OpenFile(fn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	}
	return fun(writer)
}
