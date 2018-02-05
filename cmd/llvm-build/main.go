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

// Command llvm-build generates bazel build rules for the LLVM dependency.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	name string
	mode string
	path string
	deps []string
}

func main() {
	base := os.Args[1]
	libs := []entry{}
	filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if info.Name() != "LLVMBuild.txt" {
			return nil
		}
		dir := filepath.Dir(path)
		rel, err := filepath.Rel(base, dir)
		if err != nil {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		scanner := bufio.NewScanner(file)
		lib := entry{path: rel}
		key := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 {
				continue
			}
			if line[0] == ';' {
				continue
			}
			if line[0] == '[' {
				libs = append(libs, lib)
				lib = entry{path: rel}
				continue
			}
			if words := strings.SplitN(line, "=", 2); len(words) == 2 {
				key = strings.TrimSpace(words[0])
				line = strings.TrimSpace(words[1])
			}
			if line == "" {
				continue
			}
			switch key {
			case "name":
				lib.name += line
			case "type":
				lib.mode += line
			case "required_libraries":
				lib.deps = append(lib.deps, strings.Split(line, " ")...)
			}
		}
		libs = append(libs, lib)
		return nil
	})
	fmt.Printf(`# AUTOGENERATED FILE
# This file is automatically generated from the LLVMBuild.txt files
# Do not change this file by hand.
# See cmd/llvm-build/main.go for details.
# To update this file run
# bazel run //cmd/llvm-build $(bazel info output_base)/external/llvm > $(bazel info workspace)/tools/build/third_party/llvm/libs.bzl

load("@//tools/build/third_party:llvm/rules.bzl", "llvmLibrary")

def llvm_auto_libs(**kwargs):
`)
	for _, lib := range libs {
		if lib.name == "" || lib.mode != "Library" {
			continue
		}
		fmt.Printf(`	llvm%v(
		name="%v",
		path="%v",
		deps=[`, lib.mode, lib.name, lib.path)
		for i, dep := range lib.deps {
			if i > 0 {
				fmt.Printf(`, `)
			}
			fmt.Printf(`":%v"`, dep)
		}
		fmt.Printf(`],
		**kwargs
	)
`)
	}
}
