// Copyright (C) 2021 Google Inc.
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

package etc

import (
	"bytes"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/os/device"
)

func decodeETCBaseMulModTbl(v uint64) (base, mul int, modTbl [8]int) {
	alphaModTbl := [16][8]int{
		{-3, -6, -9, -15, 2, 5, 8, 14},
		{-3, -7, -10, -13, 2, 6, 9, 12},
		{-2, -5, -8, -13, 1, 4, 7, 12},
		{-2, -4, -6, -13, 1, 3, 5, 12},
		{-3, -6, -8, -12, 2, 5, 7, 11},
		{-3, -7, -9, -11, 2, 6, 8, 10},
		{-4, -7, -8, -11, 3, 6, 7, 10},
		{-3, -5, -8, -11, 2, 4, 7, 10},
		{-2, -6, -8, -10, 1, 5, 7, 9},
		{-2, -5, -8, -10, 1, 4, 7, 9},
		{-2, -4, -8, -10, 1, 3, 7, 9},
		{-2, -5, -7, -10, 1, 4, 6, 9},
		{-3, -4, -7, -10, 2, 3, 6, 9},
		{-1, -2, -3, -10, 0, 1, 2, 9},
		{-4, -6, -8, -9, 3, 5, 7, 8},
		{-3, -5, -7, -9, 2, 4, 6, 8},
	}
	// ┏━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┓
	// ┃         Base          ┃Multiplier ┃Table Index┃
	// ┣━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━┫
	// ┃₆₃│₆₂│₆₁│₆₀│₅₉│₅₈│₅₇│₅₆┃₅₅│₅₄│₅₃│₅₂┃₅₁│₅₀│₄₉│₄₈┃
	// ┖──┴──┴──┴──┴──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┚
	return int(v >> 56),
		int((v >> 52) & 15),
		alphaModTbl[(v>>48)&15]
}

func decodeETC(src []byte, width, height, depth int, alphaMode image.FmtETC2_AlphaMode) ([]byte, error) {
	dst := make([]byte, width*height*depth*4)

	blockWidth := sint.Max((width+3)/4, 1)
	blockHeight := sint.Max((height+3)/4, 1)

	const (
		R = 0
		G = 1
		B = 2
	)
	c := [4][3]int{}
	codes := [2][4]int{}
	modTbl0 := [2][8][4]int{ // differential mode
		{ // when opaque == 0:
			{0, 8, 0, -8},
			{0, 17, 0, -17},
			{0, 29, 0, -29},
			{0, 42, 0, -42},
			{0, 60, 0, -60},
			{0, 80, 0, -80},
			{0, 106, 0, -106},
			{0, 183, 0, -183},
		}, { // when opaque == 1:
			{2, 8, -2, -8},
			{5, 17, -5, -17},
			{9, 29, -9, -29},
			{13, 42, -13, -42},
			{18, 60, -18, -60},
			{24, 80, -24, -80},
			{33, 106, -33, -106},
			{47, 183, -47, -183},
		},
	}
	modTbl1 := [8]int{3, 6, 11, 16, 23, 32, 41, 64}
	diffTbl := [8]int{0, 1, 2, 3, -4, -3, -2, -1}
	flipTbl := [2][16]int{
		{0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1},
		{0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1},
	}
	alpha := [16]byte{
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	}

	r := endian.Reader(bytes.NewReader(src), device.BigEndian)

	for z := 0; z < depth; z++ {
		dst := dst[z*width*height*4:]
		for by := 0; by < blockHeight; by++ {
			for bx := 0; bx < blockWidth; bx++ {
				if alphaMode == image.FmtETC2_ALPHA_8BIT {
					v64 := r.Uint64()
					base, mul, modTbl := decodeETCBaseMulModTbl(v64)
					for i := uint8(0); i < 16; i++ {
						mod := modTbl[(v64>>(i*3))&7]
						alpha[15-i] = sint.Byte(base + mod*mul)
					}
				}

				v64 := r.Uint64()
				flip := (v64 >> 32) & 1
				diff := (v64 >> 33) & 1
				opaque := 1
				if alphaMode == image.FmtETC2_ALPHA_1BIT {
					opaque = int(diff)
				}

				mode := uint(0)
				for i := uint(0); i < 3; i++ {
					if alphaMode != image.FmtETC2_ALPHA_1BIT && diff == 0 {
						// ┏━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━┳━━┳━━┓
						// ┃    R₀     ┃    R₁     ┃    G₀     ┃    G₁     ┃    B₀     ┃    B₁     ┃   C₀   ┃   C₁   ┃df┃fp┃
						// ┣━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━╋━━┯━━┯━━╋━━╋━━┫
						// ┃₆₃│₆₂│₆₁│₆₀┃₅₉│₅₈│₅₇│₅₆┃₅₅│₅₄│₅₃│₅₂┃₅₁│₅₀│₄₉│₄₈┃₄₇│₄₆│₄₅│₄₄┃₄₃│₄₂│₄₁│₄₀┃₃₉│₃₈│₃₇┃₃₆│₃₅│₃₄┃₃₃┃₃₂┃
						// ┖──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┸──┴──┴──┸──┸──┚
						a := (v64 >> (60 - i*8)) & 15
						b := (v64 >> (56 - i*8)) & 15
						c[0][i] = int((a << 4) | a)
						c[1][i] = int((b << 4) | b)
					} else {
						// ┏━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━┳━━━━━━━━┳━━┳━━┓
						// ┃      R₀      ┃Delta R₀┃      G₀      ┃Delta G₀┃      B₀      ┃Delta B₀┃   C₀   ┃   C₁   ┃df┃fp┃
						// ┣━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━╋━━┯━━┯━━╋━━┯━━┯━━╋━━╋━━┫
						// ┃₆₃│₆₂│₆₁│₆₀│₅₉┃₅₈│₅₇│₅₆┃₅₅│₅₄│₅₃│₅₂│₅₁┃₅₀│₄₉│₄₈┃₄₇│₄₆│₄₅│₄₄│₄₃┃₄₂│₄₁│₄₀┃₃₉│₃₈│₃₇┃₃₆│₃₅│₃₄┃₃₃┃₃₂┃
						// ┖──┴──┴──┴──┴──┸──┴──┴──┸──┴──┴──┴──┴──┸──┴──┴──┸──┴──┴──┴──┴──┸──┴──┴──┸──┴──┴──┸──┴──┴──┸──┸──┚
						a := (v64 >> (59 - i*8)) & 31
						d := (v64 >> (56 - i*8)) & 7
						b := int(a) + diffTbl[d]
						if b < 0 || b > 31 {
							mode = i + 1
							break
						}
						c[0][i] = int((a << 3) | (a >> 2))
						c[1][i] = int((b << 3) | (b >> 2))
					}
				}

				switch mode {
				case 0: // individual & differential mode (ETC1)
					codes[0] = modTbl0[opaque][(v64>>37)&7]
					codes[1] = modTbl0[opaque][(v64>>34)&7]

					blockTbl := flipTbl[flip]

					i := uint(0)
					for x := bx * 4; x < (bx+1)*4; x++ {
						for y := by * 4; y < (by+1)*4; y++ {
							if x < width && y < height {
								block := blockTbl[i]
								k := 4 * (y*width + x)
								idx := ((v64 >> i) & 1) | ((v64 >> (15 + i)) & 2)
								if opaque == 0 && idx == 2 {
									dst[k+2] = 0
									dst[k+1] = 0
									dst[k+0] = 0
									dst[k+3] = 0
								} else {
									base := c[block]
									shift := codes[block][idx]
									dst[k+2] = sint.Byte(base[2] + shift)
									dst[k+1] = sint.Byte(base[1] + shift)
									dst[k+0] = sint.Byte(base[0] + shift)
									dst[k+3] = alpha[i]
								}
							}
							i++
						}
					}
				case 1: // T-mode
					// ┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┓
					// ┃        ┃  R₀ ┏━━┓     ┃    G₀     ┃    B₀     ┃    R₂     ┃    G₂     ┃    B₂     ┃     ┏━━┓  ┃
					// ┣━━┯━━┯━━╋━━┯━━╋━━╋━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━╋━━╋━━┫
					// ┃₆₃│₆₂│₆₁┃₆₀│₅₉┃₅₈┃₅₇│₅₆┃₅₅│₅₄│₅₃│₅₂┃₅₁│₅₀│₄₉│₄₈┃₄₇│₄₆│₄₅│₄₄┃₄₃│₄₂│₄₁│₄₀┃₃₉│₃₈│₃₇│₃₆┃₃₅│₃₄┃₃₃┃₃₂┃
					// ┖──┴──┴──┸──┴──┸──┸──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┸──┸──┚

					// Load colors
					c[0][R] = int(u64.Expand4to8(((v64 >> 57) & 12) | (v64>>56)&3))
					c[0][G] = int(u64.Expand4to8(v64 >> 52))
					c[0][B] = int(u64.Expand4to8(v64 >> 48))
					c[2][R] = int(u64.Expand4to8(v64 >> 44))
					c[2][G] = int(u64.Expand4to8(v64 >> 40))
					c[2][B] = int(u64.Expand4to8(v64 >> 36))

					// Load intensity modifier
					modIdx := ((v64 >> 33) & 6) | ((v64 >> 32) & 1)
					mod := modTbl1[modIdx]
					// Calculate C₁ and c₃
					for i := 0; i < 3; i++ {
						c[1][i] = c[2][i] + mod
						c[3][i] = c[2][i] - mod
					}
					// ┏━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┓
					// ┃  x₀ ┃  x₀ ┃  x₀ ┃  x₀ ┃  x₁ ┃  x₁ ┃  x₁ ┃  x₁ ┃  x₂ ┃  x₂ ┃  x₂ ┃  x₂ ┃  x₃ ┃  x₃ ┃  x₃ ┃  x₃ ┃
					// ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃
					// ┣━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━┫
					// ┃₃₁│₃₀┃₂₉│₂₈┃₂₇│₂₆┃₂₅│₂₄┃₂₃│₂₂┃₂₁│₂₀┃₁₉│₁₈┃₁₇│₁₆┃₁₅│₁₄┃₁₃│₁₂┃₁₁│₁₀┃ ₉│ ₈┃ ₇│ ₆┃ ₅│ ₄┃ ₃│ ₂┃ ₁│ ₀┃
					// ┖──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┚
					// Use 2-bit indices to lookup texel colors
					i := uint(0)
					for x := bx * 4; x < (bx+1)*4; x++ {
						for y := by * 4; y < (by+1)*4; y++ {
							if x < width && y < height {
								k := 4 * (y*width + x)
								idx := ((v64 >> i) & 1) | ((v64 >> (15 + i)) & 2)
								if opaque == 0 && idx == 2 {
									dst[k+0] = 0
									dst[k+1] = 0
									dst[k+2] = 0
									dst[k+3] = 0
								} else {
									dst[k+0] = sint.Byte(c[idx][0])
									dst[k+1] = sint.Byte(c[idx][1])
									dst[k+2] = sint.Byte(c[idx][2])
									dst[k+3] = alpha[i]
								}
							}
							i++
						}
					}
				case 2: // H-mode
					// ┏━━┳━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━┓
					// ┃  ┃    R₀     ┃   G₀   ┏━━━━━━━━┓  ┃  ┏━━┓   B₀   ┃    R₂     ┃    G₂     ┃    B₂     ┃md┏━━┓  ┃
					// ┣━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━╋━━┯━━┯━━╋━━╋━━╋━━╋━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━╋━━╋━━╋━━┫
					// ┃₆₃┃₆₂│₆₁│₆₀│₅₉┃₅₈│₅₇│₅₆┃₅₅│₅₄│₅₃┃₅₂┃₅₁┃₅₀┃₄₉│₄₈│₄₇┃₄₆│₄₅│₄₄│₄₃┃₄₂│₄₁│₄₀│₃₉┃₃₈│₃₇│₃₆│₃₅┃₃₄┃₃₃┃₃₂┃
					// ┖──┸──┴──┴──┴──┸──┴──┴──┸──┴──┴──┸──┸──┸──┸──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┴──┴──┴──┸──┸──┸──┚

					// Load colors
					c[0][R] = int(u64.Expand4to8(v64 >> 59))
					c[0][G] = int(u64.Expand4to8(((v64 >> 55) & 14) | ((v64 >> 52) & 1)))
					c[0][B] = int(u64.Expand4to8(((v64 >> 48) & 8) | ((v64 >> 47) & 7)))
					c[2][R] = int(u64.Expand4to8(v64 >> 43))
					c[2][G] = int(u64.Expand4to8(v64 >> 39))
					c[2][B] = int(u64.Expand4to8(v64 >> 35))

					// Load intensity modifier
					modIdx := ((v64 >> 32) & 4) | ((v64 >> 31) & 2)
					// LSB of modIdx is 1 if:
					if (c[0][R]<<16)+(c[0][G]<<8)+c[0][B] >= (c[2][R]<<16)+(c[2][G]<<8)+c[2][B] {
						modIdx++
					}
					mod := modTbl1[modIdx]
					// Calculate C₁ and c₃
					for i := 0; i < 3; i++ {
						c[0][i], c[1][i] = c[0][i]+mod, c[0][i]-mod
						c[2][i], c[3][i] = c[2][i]+mod, c[2][i]-mod
					}
					// ┏━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┳━━━━━┓
					// ┃  x₀ ┃  x₀ ┃  x₀ ┃  x₀ ┃  x₁ ┃  x₁ ┃  x₁ ┃  x₁ ┃  x₂ ┃  x₂ ┃  x₂ ┃  x₂ ┃  x₃ ┃  x₃ ┃  x₃ ┃  x₃ ┃
					// ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃  y₀ ┃  y₁ ┃  y₂ ┃  y₃ ┃
					// ┣━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━╋━━┯━━┫
					// ┃₃₁│₃₀┃₂₉│₂₈┃₂₇│₂₆┃₂₅│₂₄┃₂₃│₂₂┃₂₁│₂₀┃₁₉│₁₈┃₁₇│₁₆┃₁₅│₁₄┃₁₃│₁₂┃₁₁│₁₀┃ ₉│ ₈┃ ₇│ ₆┃ ₅│ ₄┃ ₃│ ₂┃ ₁│ ₀┃
					// ┖──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┸──┴──┚
					// Use 2-bit indices to lookup texel colors
					i := uint(0)
					for x := bx * 4; x < (bx+1)*4; x++ {
						for y := by * 4; y < (by+1)*4; y++ {
							if x < width && y < height {
								k := 4 * (y*width + x)
								idx := ((v64 >> i) & 1) | ((v64 >> (15 + i)) & 2)
								if opaque == 0 && idx == 2 {
									dst[k+0] = 0
									dst[k+1] = 0
									dst[k+2] = 0
									dst[k+3] = 0
								} else {
									dst[k+0] = sint.Byte(c[idx][0])
									dst[k+1] = sint.Byte(c[idx][1])
									dst[k+2] = sint.Byte(c[idx][2])
									dst[k+3] = alpha[i]
								}
							}
							i++
						}
					}
				case 3: // planar-mode
					// ┏━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┓
					// ┃  ┃    R₀           ┃  ┏━━┓           G₀    ┃  ┏━━━━━━━━┓  B₀ ┏━━┓        ┃    R₁        ┏━━┓  ┃
					// ┣━━╋━━┯━━┯━━┯━━┯━━┯━━╋━━╋━━╋━━┯━━┯━━┯━━┯━━┯━━╋━━╋━━┯━━┯━━╋━━┯━━╋━━╋━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━╋━━╋━━┫
					// ┃₆₃┃₆₂│₆₁│₆₀│₅₉│₅₈│₅₇┃₅₆┃₅₅┃₅₄│₅₃│₅₂│₅₁│₅₀│₄₉┃₄₈┃₄₇│₄₆│₄₅┃₄₄│₄₃┃₄₂┃₄₁│₄₀│₃₉┃₃₈│₃₇│₃₆│₃₅│₃₄┃₃₃┃₃₂┃
					// ┖──┸──┴──┴──┴──┴──┴──┸──┸──┸──┴──┴──┴──┴──┴──┸──┸──┴──┴──┸──┴──┸──┸──┴──┴──┸──┴──┴──┴──┴──┸──┸──┚
					//
					// ┏━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┓
					// ┃         G₁         ┃        B₁       ┃        R₂       ┃         G₂         ┃       B₂        ┃
					// ┣━━┯━━┯━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━┯━━┯━━╋━━┯━━┯━━┯━━┯━━┯━━┫
					// ┃₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅┃₂₄│₂₃│₂₂│₂₁│₂₀│₁₉┃₁₈│₁₇│₁₆│₁₅│₁₄│₁₃┃₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
					// ┖──┴──┴──┴──┴──┴──┴──┸──┴──┴──┴──┴──┴──┸──┴──┴──┴──┴──┴──┸──┴──┴──┴──┴──┴──┴──┸──┴──┴──┴──┴──┴──┚

					// Load colors
					c[0][R] = int(u64.Expand6to8(v64 >> 57))
					c[0][G] = int(u64.Expand7to8(((v64 >> 50) & 64) | ((v64 >> 49) & 63)))
					c[0][B] = int(u64.Expand6to8(((v64 >> 43) & 32) | ((v64 >> 40) & 24) | ((v64 >> 39) & 7)))

					c[1][R] = int(u64.Expand6to8(((v64 >> 33) & 62) | ((v64 >> 32) & 1)))
					c[1][G] = int(u64.Expand7to8(v64 >> 25))
					c[1][B] = int(u64.Expand6to8(v64 >> 19))

					c[2][R] = int(u64.Expand6to8(v64 >> 13))
					c[2][G] = int(u64.Expand7to8(v64 >> 6))
					c[2][B] = int(u64.Expand6to8(v64))

					i := 0
					for dx := 0; dx < 4; dx++ {
						x := bx*4 + dx
						for dy := 0; dy < 4; dy++ {
							y := by*4 + dy
							if x < width && y < height {
								k := 4 * (y*width + x)
								dst[k+0] = sint.Byte((dx*(c[1][R]-c[0][R]) + dy*(c[2][R]-c[0][R]) + 4*c[0][R] + 2) >> 2)
								dst[k+1] = sint.Byte((dx*(c[1][G]-c[0][G]) + dy*(c[2][G]-c[0][G]) + 4*c[0][G] + 2) >> 2)
								dst[k+2] = sint.Byte((dx*(c[1][B]-c[0][B]) + dy*(c[2][B]-c[0][B]) + 4*c[0][B] + 2) >> 2)
								dst[k+3] = alpha[i]
							}
							i++
						}
					}
				}
			}
		}
	}

	return dst, nil
}

func decodeETCU11(src []byte, width, height, depth int, channels int) ([]byte, error) {
	dst := make([]byte, width*height*depth*channels*2)
	blockWidth := sint.Max((width+3)/4, 1)
	blockHeight := sint.Max((height+3)/4, 1)
	r := endian.Reader(bytes.NewReader(src), device.BigEndian)

	for z := 0; z < depth; z++ {
		dst := dst[z*width*height*channels*2:]
		for by := 0; by < blockHeight; by++ {
			for bx := 0; bx < blockWidth; bx++ {
				for c := 0; c < channels; c++ {
					v64 := r.Uint64()
					base, mul, modTbl := decodeETCBaseMulModTbl(v64)
					if mul != 0 {
						mul *= 8
					} else {
						mul = 1
					}
					i := uint(15)
					for x := bx * 4; x < bx*4+4; x++ {
						for y := by * 4; y < by*4+4; y++ {
							if x < width && y < height {
								mod := modTbl[(v64>>(i*3))&7]
								u11 := uint(sint.Clamp(base*8+4+mod*mul, 0, 2047))
								u16 := (u11 << 5) | (u11 >> 5)
								k := 2*channels*(y*width+x) + c*2
								dst[k+0] = byte(u16)
								dst[k+1] = byte(u16 >> 8)
							}
							i--
						}
					}
				}
			}
		}
	}

	return dst, nil
}

func decodeETCS11(src []byte, width, height, depth int, channels int) ([]byte, error) {
	dst := make([]byte, width*height*depth*channels*2)
	blockWidth := sint.Max((width+3)/4, 1)
	blockHeight := sint.Max((height+3)/4, 1)
	r := endian.Reader(bytes.NewReader(src), device.BigEndian)
	for z := 0; z < depth; z++ {
		dst := dst[z*width*height*channels*2:]
		for by := 0; by < blockHeight; by++ {
			for bx := 0; bx < blockWidth; bx++ {
				for c := 0; c < channels; c++ {
					v64 := r.Uint64()
					base, mul, modTbl := decodeETCBaseMulModTbl(v64)
					base = int(int8(base))
					if mul != 0 {
						mul *= 8
					} else {
						mul = 1
					}
					i := uint(15)
					for x := bx * 4; x < bx*4+4; x++ {
						for y := by * 4; y < by*4+4; y++ {
							if x < width && y < height {
								mod := modTbl[(v64>>(i*3))&7]
								s11 := sint.Clamp(base*8+mod*mul, -1023, 1023)
								var s16 int
								if s11 >= 0 {
									s16 = (s11 << 5) | (s11 >> 5)
								} else {
									s16 = -((-s11 << 5) | (-s11 >> 5))
								}
								k := 2*channels*(y*width+x) + c*2
								dst[k+0] = byte(s16)
								dst[k+1] = byte(s16 >> 8)
							}
							i--
						}
					}
				}
			}
		}
	}

	return dst, nil
}
