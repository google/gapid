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

func rgbaAvg(a, b rgbaF32) rgbaF32 {
	return rgbaF32{(a.r + b.r) * 0.5, (a.g + b.g) * 0.5, (a.b + b.b) * 0.5, (a.a + b.a) * 0.5}
}

func rgbaLerp(a, b rgbaF32, f float32) rgbaF32 {
	return rgbaF32{a.r + (b.r-a.r)*f, a.g + (b.g-a.g)*f, a.b + (b.b-a.b)*f, a.a + (b.a-a.a)*f}
}

// resizeRGBA_F32 returns a RGBA_F32 image resized from srcW x srcH to dstW x dstH.
// The algorithm uses pixel-pair averaging to down-sample (if required) the
// image to no greater than twice the width or height than the target
// dimensions, then uses a bilinear interpolator to calculate the final image
// at the requested size.
func resizeRGBA_F32(data []byte, srcW, srcH, srcD, dstW, dstH, dstD int) ([]byte, error) {
	if err := checkSize(data, RGBA_F32.format(), srcW, srcH, srcD); err != nil {
		return nil, err
	}
	if srcW <= 0 || srcH <= 0 || srcD <= 0 {
		return nil, fmt.Errorf("Invalid source size for Resize: %dx%dx%d", srcW, srcH, srcD)
	}
	if dstW <= 0 || dstH <= 0 || dstD <= 0 {
		return nil, fmt.Errorf("Invalid target size for Resize: %dx%dx%d", dstW, dstH, dstD)
	}
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)
	bufTexels := sint.Max(srcW*srcH*srcD, dstW*dstH*dstD)
	bufA, bufB := make([]rgbaF32, bufTexels), make([]rgbaF32, bufTexels)
	for i := range bufA {
		bufA[i] = rgbaF32{r.Float32(), r.Float32(), r.Float32(), r.Float32()}
	}

	dst, src := bufB, bufA

	samples := func(val, max int, scale float64) (int, int, float32) {
		f := float64(val) * scale
		i := int(f)
		return i, sint.Min(i+1, max-1), float32(f - float64(i))
	}

	for dstD*2 <= srcD { // Depth 2x downsample
		i, newD := 0, srcD/2
		for z := 0; z < newD; z++ {
			srcA, srcB := src[srcW*srcH*z*2:], src[srcW*srcH*(z*2+1):]
			for y := 0; y < srcH; y++ {
				srcA, srcB := srcA[srcW*y:], srcB[srcW*y:]
				for x := 0; x < srcW; x++ {
					dst[i] = rgbaAvg(srcA[x], srcB[x])
					i++
				}
			}
		}
		dst, src, srcD = src, dst, newD
	}

	if srcD != dstD { // Depth bi-linear downsample
		i, s := 0, float64(sint.Max(srcD-1, 0))/float64(sint.Max(dstD-1, 1))
		for z := 0; z < dstD; z++ {
			iA, iB, f := samples(z, srcD, s)
			srcA, srcB := src[srcW*srcH*iA:], src[srcW*srcH*iB:]
			for y := 0; y < srcH; y++ {
				srcA, srcB := srcA[srcW*y:], srcB[srcW*y:]
				for x := 0; x < srcW; x++ {
					dst[i] = rgbaLerp(srcA[x], srcB[x], f)
					i++
				}
			}
		}
		dst, src, srcD = src, dst, dstD
	}

	for dstH*2 <= srcH { // Vertical 2x downsample
		i, newH := 0, srcH/2
		for z := 0; z < srcD; z++ {
			src := src[srcW*srcH*z:]
			for y := 0; y < newH; y++ {
				srcA, srcB := src[srcW*y*2:], src[srcW*(y*2+1):]
				for x := 0; x < srcW; x++ {
					dst[i] = rgbaAvg(srcA[x], srcB[x])
					i++
				}
			}
		}
		dst, src, srcH = src, dst, newH
	}

	if srcH != dstH { // Vertical bi-linear downsample
		i, s := 0, float64(sint.Max(srcH-1, 0))/float64(sint.Max(dstH-1, 1))
		for z := 0; z < srcD; z++ {
			src := src[srcW*srcH*z:]
			for y := 0; y < dstH; y++ {
				iA, iB, f := samples(y, srcH, s)
				srcA, srcB := src[srcW*iA:], src[srcW*iB:]
				for x := 0; x < srcW; x++ {
					dst[i] = rgbaLerp(srcA[x], srcB[x], f)
					i++
				}
			}
		}
		dst, src, srcH = src, dst, dstH
	}

	for dstW*2 <= srcW { // Horizontal 2x downsample
		i, newW := 0, srcW/2
		for z := 0; z < srcD; z++ {
			src := src[srcW*srcH*z:]
			for y := 0; y < srcH; y++ {
				src := src[srcW*y:]
				for x := 0; x < srcW/2; x++ {
					dst[i] = rgbaAvg(src[x*2], src[x*2+1])
					i++
				}
			}
		}
		dst, src, srcW = src, dst, newW
	}

	if srcW != dstW { // Horizontal bi-linear downsample
		i, s := 0, float64(sint.Max(srcW-1, 0))/float64(sint.Max(dstW-1, 1))
		for z := 0; z < srcD; z++ {
			src := src[srcW*srcH*z:]
			for y := 0; y < srcH; y++ {
				src := src[srcW*y:]
				for x := 0; x < dstW; x++ {
					iA, iB, f := samples(x, srcW, s)
					dst[i] = rgbaLerp(src[iA], src[iB], f)
					i++
				}
			}
		}
		dst, src, srcW = src, dst, dstW
	}

	out := make([]byte, dstW*dstH*dstD*4*4)
	w := endian.Writer(bytes.NewBuffer(out[:0]), device.LittleEndian)
	for i, c := 0, dstW*dstH*dstD; i < c; i++ {
		w.Float32(src[i].r)
		w.Float32(src[i].g)
		w.Float32(src[i].b)
		w.Float32(src[i].a)
	}

	return out, nil
}
