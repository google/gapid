// Copyright (C) 2020 Google Inc.
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

// img2h is a utility program that creates C++ headers containing image data.
// It is usefull to embed texture data within a binary to simplify loading.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"

	// Import to register image formats.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var (
	out = flag.String("out", "-", "Output file, '-' for stdout")
)

func main() {
	app.ShortHelp = "img2h generates C++ headers containing image data"
	app.Name = "img2h"
	app.Run(run)
}

func run(ctx context.Context) error {
	if flag.NArg() != 1 {
		app.Usage(ctx, "Exactly one image file expected, got %d", flag.NArg())
		return nil
	}

	in, err := os.Open(flag.Arg(0))
	if err != nil {
		return log.Errf(ctx, err, "Failed to open %s", flag.Arg(0))
	}
	defer in.Close()

	img, _, err := image.Decode(in)
	if err != nil {
		return log.Errf(ctx, err, "Failed to decode image")
	}

	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	model := color.RGBAModel

	o := os.Stdout
	if *out != "-" {
		o, err = os.Create(*out)
		if err != nil {
			return log.Errf(ctx, err, "Failed to create %s", *out)
		}
		defer o.Close()
	}

	fmt.Fprintf(o, "const struct {\n")
	fmt.Fprintf(o, "  VkFormat format;\n")
	fmt.Fprintf(o, "  size_t width;\n")
	fmt.Fprintf(o, "  size_t height;\n")
	fmt.Fprintf(o, "  struct {\n")
	fmt.Fprintf(o, "    uint8_t r;\n")
	fmt.Fprintf(o, "    uint8_t g;\n")
	fmt.Fprintf(o, "    uint8_t b;\n")
	fmt.Fprintf(o, "    uint8_t a;\n")
	fmt.Fprintf(o, "  } data[%d];\n", width*height)
	fmt.Fprintf(o, "} texture = {\n")
	fmt.Fprintf(o, "  VK_FORMAT_R8G8B8A8_UNORM,\n")
	fmt.Fprintf(o, "  %d,\n", width)
	fmt.Fprintf(o, "  %d,\n", height)
	fmt.Fprintf(o, "  {\n")

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := model.Convert(img.At(x, y)).(color.RGBA)
			fmt.Fprintf(o, "{%d, %d, %d, %d}, ", p.R, p.G, p.B, 255)
		}
		fmt.Fprintf(o, "\n")
	}

	fmt.Fprintf(o, "  }\n")
	fmt.Fprintf(o, "};\n")
	return nil
}
