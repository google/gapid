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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/gapis/shadertools"
)

var (
	output     = flag.String("out", "", "Destination for the converted shader")
	debuggable = flag.Bool("debuggable", false, "Make the shader debuggable")
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
	}

	// Read input
	input := args[0]
	source, err := ioutil.ReadFile(input)
	if err != nil {
		return err
	}

	// Process the shader
	result, err := convert(string(source), filepath.Ext(input))
	if err != nil {
		return err
	}

	// Write output
	if *output == "" {
		fmt.Print(result)
	} else {
		err := ioutil.WriteFile(*output, []byte(result), 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func convert(source, shaderType string) (result string, err error) {
	opts := shadertools.Option{}
	switch shaderType {
	case ".vert":
		opts.IsVertexShader = true
	case ".frag":
		opts.IsFragmentShader = true
	default:
		return "", fmt.Errorf("File extension must be .vert or .frag (seen %v)", shaderType)
	}
	opts.MakeDebuggable = *debuggable
	res := shadertools.ConvertGlsl(string(source), &opts)
	if !res.Ok {
		return "", fmt.Errorf("Failed to translate GLSL:\n%s\nSource:%s\n", res.Message, source)
	}
	return res.SourceCode, nil
}
