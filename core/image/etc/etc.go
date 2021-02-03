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
	"github.com/google/gapid/core/image"
)

var (
	// ETC2
	ETC2_RGB_U8_NORM         = image.NewETC2_RGB_U8_NORM("ETC2_RGB_U8_NORM")
	ETC2_RGBA_U8_NORM        = image.NewETC2_RGBA_U8_NORM("ETC2_RGBA_U8_NORM")
	ETC2_RGBA_U8U8U8U1_NORM  = image.NewETC2_RGBA_U8U8U8U1_NORM("ETC2_RGBA_U8U8U8U1_NORM")
	ETC2_SRGB_U8_NORM        = image.NewETC2_SRGB_U8_NORM("ETC2_SRGB_U8_NORM")
	ETC2_SRGBA_U8_NORM       = image.NewETC2_SRGBA_U8_NORM("ETC2_SRGBA_U8_NORM")
	ETC2_SRGBA_U8U8U8U1_NORM = image.NewETC2_SRGBA_U8U8U8U1_NORM("ETC2_SRGBA_U8U8U8U1_NORM")

	// EAC
	ETC2_R_U11_NORM  = image.NewETC2_R_U11_NORM("ETC2_R_U11_NORM")
	ETC2_RG_U11_NORM = image.NewETC2_RG_U11_NORM("ETC2_RG_U11_NORM")
	ETC2_R_S11_NORM  = image.NewETC2_R_S11_NORM("ETC2_R_S11_NORM")
	ETC2_RG_S11_NORM = image.NewETC2_RG_S11_NORM("ETC2_RG_S11_NORM")

	// ETC 1
	ETC1_RGB_U8_NORM = image.NewETC1_RGB_U8_NORM("ETC1_RGB_U8_NORM")
)

type converterLayout struct {
	plain      *image.Format
	compressed *image.Format
}

type etcLayout struct {
	converterLayout
	alphaMode etcAlphaMode
}

type eacLayout struct {
	converterLayout
	channels int
}

func init() {
	//ETC2 Formats
	etc2SupportMap := []etcLayout{
		{converterLayout{image.RGBA_U8_NORM, ETC2_RGB_U8_NORM}, etcAlphaNone},
		{converterLayout{image.RGBA_U8_NORM, ETC2_RGBA_U8_NORM}, etcAlpha8Bit},
		{converterLayout{image.RGBA_U8_NORM, ETC2_RGBA_U8U8U8U1_NORM}, etcAlpha1Bit},
		{converterLayout{image.SRGBA_U8_NORM, ETC2_SRGB_U8_NORM}, etcAlphaNone},
		{converterLayout{image.SRGBA_U8_NORM, ETC2_SRGBA_U8_NORM}, etcAlpha8Bit},
		{converterLayout{image.SRGBA_U8_NORM, ETC2_SRGBA_U8U8U8U1_NORM}, etcAlpha1Bit},
	}

	for _, conversion := range etc2SupportMap {
		// Intentional local copy
		conv := conversion
		image.RegisterConverter(conv.compressed, conv.plain, func(src []byte, w, h, d int) ([]byte, error) {
			return decodeETC(src, w, h, d, conv.alphaMode)
		})
	}

	// EAC formats
	etcU11SupportMap := []eacLayout{
		{converterLayout{image.R_U16_NORM, ETC2_R_U11_NORM}, 1},
		{converterLayout{image.RG_U16_NORM, ETC2_RG_U11_NORM}, 2},
	}

	for _, conversion := range etcU11SupportMap {
		// Intentional local copy
		conv := conversion
		image.RegisterConverter(conv.compressed, conv.plain, func(src []byte, w, h, d int) ([]byte, error) {
			return decodeETCU11(src, w, h, d, conv.channels)
		})
	}

	etcS11SupportMap := []eacLayout{
		{converterLayout{image.R_S16_NORM, ETC2_R_S11_NORM}, 1},
		{converterLayout{image.RG_S16_NORM, ETC2_RG_S11_NORM}, 2},
	}

	for _, conversion := range etcS11SupportMap {
		// Intentional local copy
		conv := conversion
		image.RegisterConverter(conv.compressed, conv.plain, func(src []byte, w, h, d int) ([]byte, error) {
			return decodeETCS11(src, w, h, d, conv.channels)
		})
	}

	// This is for converting via intermediate format.
	// TODO (melihyalcin): We should check if we really need all the conversion formats
	// especially from SRGB(A) formats to RGB(A) formats
	streamETCSupportMap := []converterLayout{
		{image.RGBA_U8_NORM, ETC2_RGB_U8_NORM},
		{image.RGBA_U8_NORM, ETC2_RGBA_U8_NORM},
		{image.RGBA_U8_NORM, ETC2_RGBA_U8U8U8U1_NORM},
		{image.SRGBA_U8_NORM, ETC2_SRGB_U8_NORM},
		{image.SRGBA_U8_NORM, ETC2_SRGBA_U8_NORM},
		{image.SRGBA_U8_NORM, ETC2_SRGBA_U8U8U8U1_NORM},
		{image.R_U16_NORM, ETC2_R_U11_NORM},
		{image.RG_U16_NORM, ETC2_RG_U11_NORM},
		{image.R_S16_NORM, ETC2_R_S11_NORM},
		{image.RG_S16_NORM, ETC2_RG_S11_NORM},
	}

	for _, conversion := range streamETCSupportMap {
		// Intentional local copy
		conv := conversion
		if !image.Registered(conv.compressed, image.RGB_U8_NORM) {
			image.RegisterConverter(conv.compressed, image.RGB_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
				rgb, err := image.Convert(src, w, h, d, conv.compressed, conv.plain)
				if err != nil {
					return nil, err
				}
				return image.Convert(rgb, w, h, d, conv.plain, image.RGB_U8_NORM)
			})
		}
		if !image.Registered(conv.compressed, image.RGBA_U8_NORM) {
			image.RegisterConverter(conv.compressed, image.RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
				rgba, err := image.Convert(src, w, h, d, conv.compressed, conv.plain)
				if err != nil {
					return nil, err
				}
				return image.Convert(rgba, w, h, d, conv.plain, image.RGBA_U8_NORM)
			})
		}
	}

	// ETC1 formats
	image.RegisterConverter(ETC1_RGB_U8_NORM, image.RGB_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return image.Convert(src, w, h, d, ETC2_RGB_U8_NORM, image.RGB_U8_NORM)
	})
	image.RegisterConverter(ETC1_RGB_U8_NORM, image.RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return image.Convert(src, w, h, d, ETC2_RGB_U8_NORM, image.RGBA_U8_NORM)
	})
}
