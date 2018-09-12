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
	"os"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/compiler/mangling/c"
	"github.com/google/gapid/gapil/compiler/mangling/ia64"
	"github.com/google/gapid/gapil/compiler/plugins/cloner"
	"github.com/google/gapid/gapil/compiler/plugins/encoder"
	"github.com/google/gapid/gapil/compiler/plugins/replay"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
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
		Clone   bool `help:"Emit clone methods"`
		Encode  bool `help:"Emit encoder logic"`
		Exec    bool `help:"Emit executor logic. Implies --emit-context"`
		Context bool `help:"Emit context constructor / destructor"`
		Replay  bool `help:"Emit replay generation methods"`
	}
	Namespace string        `help:"Dot-delimited root namespace(s)"`
	Symbols   symbols       `help:"Symbol generation method"`
	Optimize  bool          `help:"Optimize generated code"`
	Dump      bool          `help:"Dump LLVM IR to stderr"`
	Search    file.PathList `help:"The set of paths to search for includes"`
}

func (v *compileVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	api, mappings, err := resolve(ctx, v.Search, flags, resolver.Options{})
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
		TargetABI:   abi,
		CaptureABI:  abi,
		Namespaces:  namespaces,
		EmitExec:    v.Emit.Exec,
		EmitContext: v.Emit.Context,
	}

	if v.Emit.Encode {
		settings.Plugins = append(settings.Plugins, encoder.Plugin())
	}
	if v.Emit.Clone {
		settings.Plugins = append(settings.Plugins, cloner.Plugin())
	}
	if v.Emit.Replay {
		settings.Plugins = append(settings.Plugins, replay.Plugin(nil))
	}

	switch v.Symbols {
	case cSym:
		settings.Mangler = c.Mangle
	default:
		settings.Mangler = ia64.Mangle
	}

	prog, err := compiler.Compile([]*semantic.API{api}, mappings, settings)
	if err != nil {
		return err
	}

	if v.Optimize {
		prog.Codegen.Optimize()
	}

	if v.Dump {
		fmt.Fprintln(os.Stderr, prog.Codegen.String())
		return fmt.Errorf("IR dump")
	}

	obj, err := prog.Codegen.Object(v.Optimize)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(v.Output, obj, 0666); err != nil {
		return err
	}

	return nil
}
