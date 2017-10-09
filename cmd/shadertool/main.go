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

// The shadertool command modifies shader source code.
// For example, it converts GLSL to the desktop dialect.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/gapis/shadertools"
)

var (
	out   = flag.String("out", "", "Directory for the converted shaders")
	check = flag.Bool("check", true, "Verify that the output compiles")
	debug = flag.Bool("debug", false, "Make the shader debuggable")
	asm   = flag.Bool("asm", false, "Print disassembled info")
)

func main() {
	app.Name = "shadertool"
	app.ShortHelp = "Converts GLSL ES shader to the desktop GLSL dialect"
	app.ShortUsage = "<shader file>"
	app.Run(run)
}

func run(ctx context.Context) error {
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return nil
	}

	// Read input
	var wg sync.WaitGroup
	for _, input := range args {
		input := input
		source, err := ioutil.ReadFile(input)
		if err != nil {
			return err
		}

		wg.Add(1)
		crash.Go(func() {
			defer wg.Done()

			// Process the shader
			result, err := convert(string(source), filepath.Ext(input))
			if err != nil {
				fmt.Printf("%v: %v\n", input, err)
				return
			}

			// Write output
			if *out == "" {
				fmt.Print(result)
			} else {
				output := filepath.Join(*out, filepath.Base(input))
				err := ioutil.WriteFile(output, []byte(result), 0666)
				if err != nil {
					fmt.Printf("%v: %v\n", input, err)
					return
				}
			}
		})
	}
	wg.Wait()

	return nil
}

func convert(source, shaderType string) (result string, err error) {
	opts := shadertools.Options{}
	switch shaderType {
	case ".vert":
		opts.ShaderType = shadertools.TypeVertex
	case ".frag":
		opts.ShaderType = shadertools.TypeFragment
	default:
		return "", fmt.Errorf("File extension must be .vert or .frag (seen %v)", shaderType)
	}
	opts.MakeDebuggable = *debug
	opts.CheckAfterChanges = *check
	opts.Disassemble = *asm
	res, err := shadertools.ConvertGlsl(string(source), &opts)
	if err != nil {
		return "", err
	}
	if *asm {
		result += "/* Disassembly:\n" + res.DisassemblyString + "\n*/\n"
		result += "/* Debug info:\n" + shadertools.FormatDebugInfo(res.Info, "  ") + "\n*/\n"
	}
	result += res.SourceCode
	return result, nil
}
