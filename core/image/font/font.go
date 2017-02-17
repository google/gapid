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

package font

import (
	"image"
	"image/color"
	"image/draw"
)

// DrawString draws s to the image i starting at point p with the color c.
// Text flows left to right.
func DrawString(s string, i draw.Image, p image.Point, c color.Color) {
	// Reused glyph image
	glyph := &image.Alpha{
		Stride: glyphWidth,
		Rect:   image.Rect(0, 0, glyphWidth, glyphHeight),
	}

	// Reused color image
	color := &image.Uniform{c}

	x, y := p.X, p.Y
	for _, rune := range s {
		if rune == '\n' {
			x, y = p.X, y+glyphHeight
			continue
		}
		data, found := glyphs[rune]
		if !found {
			data = glyphs['?']
		}
		glyph.Pix = data
		dstRect := image.Rect(x, y, x+glyphWidth, y+glyphHeight)
		draw.DrawMask(i, dstRect, color, image.ZP, glyph, image.ZP, draw.Over)
		x += glyphWidth
	}
}
