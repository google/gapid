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
	"flag"
	"fmt"
	"image/png"
	"os"
)

var (
	pngPath = flag.String("png", "glyphs.png", "The source glyph png")
)

type glyph struct {
	rune rune
	data []byte
}

func run() error {
	flag.Parse()

	file, err := os.Open(*pngPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return err
	}

	glyphCountX := 16
	glyphCountY := 6

	glyphW := img.Bounds().Dx() / glyphCountX
	glyphH := img.Bounds().Dy() / glyphCountY
	glyphs := []glyph{}

	for r := ' '; r < '~'; r++ {
		g := glyph{rune: r, data: make([]byte, 0, glyphW*glyphH)}
		baseX := glyphW * (len(glyphs) % glyphCountX)
		baseY := glyphH * (len(glyphs) / glyphCountX)
		for y := 0; y < glyphH; y++ {
			for x := 0; x < glyphW; x++ {
				_, _, _, a := img.At(baseX+x, baseY+y).RGBA()
				g.data = append(g.data, byte(a>>8))
			}
		}
		glyphs = append(glyphs, g)
	}

	fmt.Print("var glyphs = map[rune][]byte {\n")
	for _, g := range glyphs {
		fmt.Printf("	'%c': []byte {\n", g.rune)
		i := 0
		for y := 0; y < glyphH; y++ {
			fmt.Printf("		")
			for x := 0; x < glyphW; x++ {
				fmt.Printf("0x%.2x, ", g.data[i])
				i++
			}
			fmt.Printf("\n")
		}
		fmt.Printf("	},\n")
	}
	fmt.Printf("}\n")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
