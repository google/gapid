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

package fmts_test

import (
	"testing"

	"fmt"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/core/stream/fmts"
)

func TestFormatNames(t *testing.T) {
	assert := assert.To(t)

	for n, f := range map[string]*stream.Format{
		"A_U16_NORM":            fmts.A_U16_NORM,
		"A_U8_NORM":             fmts.A_U8_NORM,
		"BGR_U5U6U5_NORM":       fmts.BGR_U5U6U5_NORM,
		"BGRA_U8_NORM":          fmts.BGRA_U8_NORM,
		"D_F32":                 fmts.D_F32,
		"D_U16_NORM":            fmts.D_U16_NORM,
		"DS_F32U8":              fmts.DS_F32U8,
		"DS_NU16U8":             fmts.DS_NU16U8,
		"DS_NU24U8":             fmts.DS_NU24U8,
		"Gray_U16_NORM":         fmts.Gray_U16_NORM,
		"Gray_U8_NORM":          fmts.Gray_U8_NORM,
		"L_F32":                 fmts.L_F32,
		"L_U8_NORM":             fmts.L_U8_NORM,
		"LA_U8_NORM":            fmts.LA_U8_NORM,
		"R_F16":                 fmts.R_F16,
		"R_F32":                 fmts.R_F32,
		"R_U16_NORM":            fmts.R_U16_NORM,
		"R_U8_NORM":             fmts.R_U8_NORM,
		"R_U8":                  fmts.R_U8,
		"RG_F16":                fmts.RG_F16,
		"RG_F32":                fmts.RG_F32,
		"RG_U8_NORM":            fmts.RG_U8_NORM,
		"RG_U8":                 fmts.RG_U8,
		"RG_U16_NORM":           fmts.RG_U16_NORM,
		"RGB_F16":               fmts.RGB_F16,
		"RGB_F32":               fmts.RGB_F32,
		"RGB_U1_NORM":           fmts.RGB_U1_NORM,
		"RGB_U1":                fmts.RGB_U1,
		"RGB_U4_NORM":           fmts.RGB_U4_NORM,
		"RGB_U4":                fmts.RGB_U4,
		"RGB_U5_NORM":           fmts.RGB_U5_NORM,
		"RGB_U5U6U5_NORM":       fmts.RGB_U5U6U5_NORM,
		"RGB_U8_NORM":           fmts.RGB_U8_NORM,
		"RGB_U8":                fmts.RGB_U8,
		"RGBA_F16":              fmts.RGBA_F16,
		"RGBA_F32":              fmts.RGBA_F32,
		"RGBA_U10U10U10U2_NORM": fmts.RGBA_U10U10U10U2_NORM,
		"RGBA_U4_NORM":          fmts.RGBA_U4_NORM,
		"RGBA_U5U5U5U1_NORM":    fmts.RGBA_U5U5U5U1_NORM,
		"RGBA_U8_NORM":          fmts.RGBA_U8_NORM,
		"RGBA_U8":               fmts.RGBA_U8,
		"SD_U8F32":              fmts.SD_U8F32,
		"SD_U8NU16":             fmts.SD_U8NU16,
		"SD_U8NU24":             fmts.SD_U8NU24,
		"XY_F32":                fmts.XY_F32,
		"XYZ_F32":               fmts.XYZ_F32,
		"XYZ_F64":               fmts.XYZ_F64,
		"XYZ_S16_NORM":          fmts.XYZ_S16_NORM,
		"XYZ_S16":               fmts.XYZ_S16,
		"XYZ_S8_NORM":           fmts.XYZ_S8_NORM,
		"XYZ_S8":                fmts.XYZ_S8,
		"XYZW_F32":              fmts.XYZW_F32,
		"XYZW_F64":              fmts.XYZW_F64,
		"XYZW_S10S10S10S2_NORM": fmts.XYZW_S10S10S10S2_NORM,
		"XYZW_S10S10S10S2":      fmts.XYZW_S10S10S10S2,
		"XYZW_S16_NORM":         fmts.XYZW_S16_NORM,
		"XYZW_S16":              fmts.XYZW_S16,
		"XYZW_S32_NORM":         fmts.XYZW_S32_NORM,
		"XYZW_S32":              fmts.XYZW_S32,
		"XYZW_S8_NORM":          fmts.XYZW_S8_NORM,
		"XYZW_S8":               fmts.XYZW_S8,
		"XYZW_U10U10U10U2_NORM": fmts.XYZW_U10U10U10U2_NORM,
		"XYZW_U10U10U10U2":      fmts.XYZW_U10U10U10U2,
		"XYZW_U16_NORM":         fmts.XYZW_U16_NORM,
		"XYZW_U16":              fmts.XYZW_U16,
		"XYZW_U32_NORM":         fmts.XYZW_U32_NORM,
		"XYZW_U32":              fmts.XYZW_U32,
		"XYZW_U8_NORM":          fmts.XYZW_U8_NORM,
		"XYZW_U8":               fmts.XYZW_U8,
	} {
		if assert.For("name").ThatString(n).Equals(fmt.Sprint(f)) {
			for _, c := range f.Components {
				assert.For("%v DataType", f).That(c.DataType).IsNotNil()
				assert.For("%v Sampling", f).That(c.Sampling).IsNotNil()
			}
		}
	}
}
