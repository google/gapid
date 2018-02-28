// Copyright (C) 2018 Google Inc.
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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/compiler/mangling/c"
	"github.com/google/gapid/gapil/compiler/mangling/ia64"
	"github.com/google/gapid/gapil/compiler/plugins/encoder"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "compile",
		ShortHelp: "Emits code generated from .api files",
		Action:    &compileVerb{},
	})
}

type symbols int

const (
	cSym = symbols(iota)
	cppSym
)

func (s symbols) String() string {
	switch s {
	case cSym:
		return "c"
	case cppSym:
		return "c++"
	default:
		return ""
	}
}

func (s *symbols) Choose(v interface{}) {
	*s = v.(symbols)
}

type compileVerb struct {
	Target string `help:"The target device ABI"`
	Output string `help:"The output file path"`
	Emit   struct {
		Exec   bool `help:"Emit executor logic"`
		Encode bool `help:"Emit encoder logic"`
	}
	Namespace string  `help:"Dot-delimited root namespace(s)"`
	Symbols   symbols `help:"Symbol generation method."`
	Optimize  bool
	Search    file.PathList `help:"The set of paths to search for includes"`
}

func (v *compileVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	api, mappings, err := resolve(ctx, v.Search, flags)
	if err != nil {
		return err
	}

	var abi *device.ABI
	switch v.Target { // Must match values in: tools/build/BUILD.bazel
	case "":
		abi = nil // host
	case "k8":
		abi = device.LinuxX86_64
	case "darwin_x86_64":
		abi = device.OSXX86_64
	case "x64_windows":
		abi = device.WindowsX86_64
	case "armeabi-v7a":
		abi = device.AndroidARMv7a
	case "arm64-v8a":
		abi = device.AndroidARM64v8a
	case "x86":
		abi = device.AndroidX86
	default:
		return fmt.Errorf("Unrecognised target: '%v'", v.Target)
	}

	var namespaces []string
	if v.Namespace != "" {
		namespaces = strings.Split(v.Namespace, ".")
	}

	settings := compiler.Settings{
		TargetABI:  abi,
		StorageABI: abi,
		Namespaces: namespaces,
		EmitExec:   v.Emit.Exec,
	}

	if v.Emit.Encode {
		settings.Plugins = append(settings.Plugins, encoder.Plugin())
	}

	switch v.Symbols {
	case cSym:
		settings.Mangler = c.Mangle
	default:
		settings.Mangler = ia64.Mangle
	}

	prog, err := compiler.Compile(api, mappings, settings)
	if err != nil {
		return err
	}

	if v.Optimize {
		prog.Module.Optimize()
	}

	obj, err := prog.Module.Object(v.Optimize)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(v.Output, obj, 0666); err != nil {
		return err
	}

	return nil
}
