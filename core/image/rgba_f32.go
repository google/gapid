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

package image

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
)

type rgbaF32 struct {
	r, g, b, a float32
}

// resizeRGBA_F32 returns a RGBA_F32 image resized from srcW x srcH to dstW x dstH.
// The algorithm uses pixel-pair averaging to down-sample (if required) the
// image to no greater than twice the width or height than the target
// dimensions, then uses a bilinear interpolator to calculate the final image
// at the requested size.
func resizeRGBA_F32(data []byte, srcW, srcH, dstW, dstH int) ([]byte, error) {
	if err := checkSize(data, RGBA_F32.format(), srcW, srcH); err != nil {
		return nil, err
	}
	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("Invalid source size for Resize: %dx%d", srcW, srcH)
	}
	if dstW <= 0 || dstH <= 0 {
		return nil, fmt.Errorf("Invalid target size for Resize: %dx%d", dstW, dstH)
	}
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)
	bufA, bufB := make([]rgbaF32, srcW*srcH), make([]rgbaF32, srcW*srcH)
	for i := range bufA {
		bufA[i] = rgbaF32{r.Float32(), r.Float32(), r.Float32(), r.Float32()}
	}

	dst, src := bufB, bufA

	for dstW*2 <= srcW { // Horizontal 2x downsample
		newW := srcW / 2
		for y := 0; y < srcH; y++ {
			i := newW * y
			a, b := srcW*y, srcW*y+1
			for x := 0; x < srcW/2; x++ {
				dst[i].r = (src[a].r + src[b].r) * 0.5
				dst[i].g = (src[a].g + src[b].g) * 0.5
				dst[i].b = (src[a].b + src[b].b) * 0.5
				dst[i].a = (src[a].a + src[b].a) * 0.5
				i, a, b = i+1, a+2, b+2
			}
		}
		dst, src, srcW = src, dst, newW
	}

	for dstH*2 <= srcH { // Vertical 2x downsample
		newH := srcH / 2
		for y := 0; y < newH; y++ {
			i := srcW * y
			a, b := i*2, i*2+srcW
			for x := 0; x < srcW; x++ {
				dst[i].r = (src[a].r + src[b].r) * 0.5
				dst[i].g = (src[a].g + src[b].g) * 0.5
				dst[i].b = (src[a].b + src[b].b) * 0.5
				dst[i].a = (src[a].a + src[b].a) * 0.5
				i, a, b = i+1, a+1, b+1
			}
		}
		dst, src, srcH = src, dst, newH
	}

	out := make([]byte, dstW*dstH*4*4)
	w := endian.Writer(bytes.NewBuffer(out[:0]), device.LittleEndian)
	if srcW == dstW && srcH == dstH {
		for i, c := 0, dstW*dstH; i < c; i++ {
			w.Float32(src[i].r)
			w.Float32(src[i].g)
			w.Float32(src[i].b)
			w.Float32(src[i].a)
		}
	} else {
		// bi-linear filtering
		sx := float64(sint.Max(srcW-1, 0)) / float64(sint.Max(dstW-1, 1))
		sy := float64(sint.Max(srcH-1, 0)) / float64(sint.Max(dstH-1, 1))
		for y := 0; y < dstH; y++ {
			fy := float64(y) * sy
			iy := int(fy)
			dy, y0, y1 := float32(fy-float64(iy)), iy, sint.Min(iy+1, srcH-1)
			for x := 0; x < dstW; x++ {
				fx := float64(x) * sx
				ix := int(fx)
				dx, x0, x1 := float32(fx-float64(ix)), ix, sint.Min(ix+1, srcW-1)

				a, b := src[x0+y0*srcW], src[x1+y0*srcW]
				c, d := src[x0+y1*srcW], src[x1+y1*srcW]

				p := rgbaF32{a.r + (b.r-a.r)*dx, a.g + (b.g-a.g)*dx, a.b + (b.b-a.b)*dx, a.a + (b.a-a.a)*dx}
				q := rgbaF32{c.r + (d.r-c.r)*dx, c.g + (d.g-c.g)*dx, c.b + (d.b-c.b)*dx, c.a + (d.a-c.a)*dx}
				r := rgbaF32{p.r + (q.r-p.r)*dy, p.g + (q.g-p.g)*dy, p.b + (q.b-p.b)*dy, p.a + (q.a-p.a)*dy}

				w.Float32(r.r)
				w.Float32(r.g)
				w.Float32(r.b)
				w.Float32(r.a)
			}
		}
	}
	return out, nil
}
