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

package gles

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/image/astc"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/api"
)

func (i *Image) getUnsizedFormatAndType() (unsizedFormat, ty GLenum) {
	if i.DataFormat == 0 && i.DataType == 0 {
		return getUnsizedFormatAndType(i.SizedFormat)
	}
	return i.DataFormat, i.DataType
}

func cubemapFaceToLayer(target GLenum) GLint {
	layer, _ := subCubemapFaceToLayer(nil, nil, api.CmdNoID, nil, &api.GlobalState{}, nil, 0, nil, target)
	return layer
}

// getSizedFormatFromTuple returns sized format from unsized format and component type.
func getSizedFormatFromTuple(unsizedFormat, ty GLenum) (sizedFormat GLenum) {
	sf, _ := subGetSizedFormatFromTuple(nil, nil, api.CmdNoID, nil, &api.GlobalState{}, nil, 0, nil, unsizedFormat, ty)
	if sf == GLenum_GL_NONE {
		panic(fmt.Errorf("Unknown unsized format: %v, %v", unsizedFormat, ty))
	}
	return sf
}

// getUnsizedFormatAndType returns unsized format and component type from sized format.
func getUnsizedFormatAndType(sizedFormat GLenum) (unsizedFormat, ty GLenum) {
	info, _ := subGetSizedFormatInfo(nil, nil, api.CmdNoID, nil, &api.GlobalState{}, nil, 0, nil, sizedFormat)
	if info.SizedFormat == GLenum_GL_NONE {
		panic(fmt.Errorf("Unknown sized format: %v", sizedFormat))
	}
	return info.UnsizedFormat, info.DataType
}

// getImageFormat returns the *image.Format for the given format-type tuple.
// The tuple must be in one of the following two forms:
//   (unsizedFormat, ty) - Uncompressed data.
//   (sizedFormat, NONE) - Compressed data.
//   (NONE, NONE) - Uninitialized content.
// Sized uncompressed format (e.g. GL_RGB565) is not a valid input.
func getImageFormat(format, ty GLenum) (*image.Format, error) {
	if format != GLenum_GL_NONE {
		if ty != GLenum_GL_NONE {
			imgfmt, _ := getUncompressedStreamFormat(format, ty)
			if imgfmt != nil {
				return image.NewUncompressed(fmt.Sprintf("%v, %v", format, ty), imgfmt), nil
			}
		} else {
			imgfmt, _ := getCompressedImageFormat(format)
			if imgfmt != nil {
				return imgfmt, nil
			}
		}
	} else {
		return image.NewUncompressed("<uninitialized>", &stream.Format{}), nil
	}
	return nil, fmt.Errorf("Unsupported input format-type pair: (%s, %s)", format, ty)
}

// filterUncompressedImageFormat returns a copy of f with only the components
// that have channels that pass the predicate p.
func filterUncompressedImageFormat(f *image.Format, p func(stream.Channel) bool) *image.Format {
	u := f.GetUncompressed()
	if u == nil {
		panic(fmt.Errorf("Format %v is not uncompressed", f))
	}

	out := proto.Clone(f).(*image.Format)
	filtered := out.GetUncompressed().Format
	filtered.Components = filtered.Components[:0]

	names := []string{}
	for _, c := range u.Format.Components {
		if p(c.Channel) {
			filtered.Components = append(filtered.Components, c)
			names = append(names, c.Channel.String())
		}
	}
	out.Name = fmt.Sprintf("%v from %v", strings.Join(names, ", "), f.Name)
	return out
}

var glChannelToStreamChannel = map[GLenum]stream.Channel{
	GLenum_GL_RED:             stream.Channel_Red,
	GLenum_GL_GREEN:           stream.Channel_Green,
	GLenum_GL_BLUE:            stream.Channel_Blue,
	GLenum_GL_ALPHA:           stream.Channel_Alpha,
	GLenum_GL_LUMINANCE:       stream.Channel_Luminance,
	GLenum_GL_DEPTH_COMPONENT: stream.Channel_Depth,
	GLenum_GL_STENCIL_INDEX:   stream.Channel_Stencil,
}

// getUncompressedStreamFormat returns the decoding format which can be used to read single pixel.
func getUncompressedStreamFormat(unsizedFormat, ty GLenum) (format *stream.Format, err error) {
	info, _ := subGetUnsizedFormatInfo(nil, nil, api.CmdNoID, nil, &api.GlobalState{}, nil, 0, nil, unsizedFormat)
	if info.Count == 0 {
		return nil, fmt.Errorf("Unknown unsized format: %v", unsizedFormat)
	}
	glChannels := []GLenum{info.Channel0, info.Channel1, info.Channel2, info.Channel3}
	channels := make(stream.Channels, info.Count)
	for i := range channels {
		channel, ok := glChannelToStreamChannel[glChannels[i]]
		if !ok {
			return nil, fmt.Errorf("Unknown GL channel: %v", glChannels[i])
		}
		channels[i] = channel
	}

	// Helper method to build the format.
	format = &stream.Format{}
	addComponent := func(channelIndex int, datatype *stream.DataType) {
		channel := stream.Channel_Undefined // Padding field
		if 0 <= channelIndex && channelIndex < len(channels) {
			channel = channels[channelIndex]
		}
		sampling := stream.Linear
		var sampleAsFloat bool
		if channel == stream.Channel_Depth {
			sampleAsFloat = true
		} else if channel == stream.Channel_Stencil {
			sampleAsFloat = false
		} else /* colour */ {
			sampleAsFloat = !info.Integer
		}
		if datatype.IsInteger() && sampleAsFloat {
			sampling = stream.LinearNormalized // Convert int to float
		}
		format.Components = append(format.Components, &stream.Component{datatype, sampling, channel})
	}

	// Read the components in increasing memory order (assuming little-endian architecture).
	// Note that the GL names are based on big-endian, so the order is generally backwards.
	switch ty {
	case GLenum_GL_UNSIGNED_BYTE:
		for i := range channels {
			addComponent(i, &stream.U8)
		}
	case GLenum_GL_BYTE:
		for i := range channels {
			addComponent(i, &stream.S8)
		}
	case GLenum_GL_UNSIGNED_SHORT:
		for i := range channels {
			addComponent(i, &stream.U16)
		}
	case GLenum_GL_SHORT:
		for i := range channels {
			addComponent(i, &stream.S16)
		}
	case GLenum_GL_UNSIGNED_INT:
		for i := range channels {
			addComponent(i, &stream.U32)
		}
	case GLenum_GL_INT:
		for i := range channels {
			addComponent(i, &stream.S32)
		}
	case GLenum_GL_HALF_FLOAT, GLenum_GL_HALF_FLOAT_OES:
		for i := range channels {
			addComponent(i, &stream.F16)
		}
	case GLenum_GL_FLOAT:
		for i := range channels {
			addComponent(i, &stream.F32)
		}
	case GLenum_GL_UNSIGNED_SHORT_5_6_5:
		addComponent(2, &stream.U5)
		addComponent(1, &stream.U6)
		addComponent(0, &stream.U5)
	case GLenum_GL_UNSIGNED_SHORT_4_4_4_4:
		addComponent(3, &stream.U4)
		addComponent(2, &stream.U4)
		addComponent(1, &stream.U4)
		addComponent(0, &stream.U4)
	case GLenum_GL_UNSIGNED_SHORT_5_5_5_1:
		addComponent(3, &stream.U1)
		addComponent(2, &stream.U5)
		addComponent(1, &stream.U5)
		addComponent(0, &stream.U5)
	case GLenum_GL_UNSIGNED_INT_2_10_10_10_REV:
		addComponent(0, &stream.U10)
		addComponent(1, &stream.U10)
		addComponent(2, &stream.U10)
		addComponent(3, &stream.U2)
	case GLenum_GL_UNSIGNED_INT_24_8:
		addComponent(1, &stream.U8)
		addComponent(0, &stream.U24)
	case GLenum_GL_UNSIGNED_INT_10F_11F_11F_REV:
		addComponent(0, &stream.F11)
		addComponent(1, &stream.F11)
		addComponent(2, &stream.F10)
	case GLenum_GL_UNSIGNED_INT_5_9_9_9_REV:
		return fmts.RGBE_U9U9U9U5, nil
	case GLenum_GL_FLOAT_32_UNSIGNED_INT_24_8_REV:
		addComponent(0, &stream.F32)
		addComponent(1, &stream.U8)
		addComponent(-1, &stream.U24)
	default:
		return nil, fmt.Errorf("Unsupported data type: %v", ty)
	}
	return format, nil
}

// getCompressedImageFormat returns *image.Format for the given compressed format.
func getCompressedImageFormat(format GLenum) (*image.Format, error) {
	switch format {
	// ETC1
	case GLenum_GL_ETC1_RGB8_OES:
		return image.NewETC1_RGB_U8_NORM("GL_ETC1_RGB8_OES"), nil

	// ASTC
	case GLenum_GL_COMPRESSED_RGBA_ASTC_4x4_KHR:
		return astc.NewRGBA_4x4("GL_COMPRESSED_RGBA_ASTC_4x4_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_5x4_KHR:
		return astc.NewRGBA_5x4("GL_COMPRESSED_RGBA_ASTC_5x4_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_5x5_KHR:
		return astc.NewRGBA_5x5("GL_COMPRESSED_RGBA_ASTC_5x5_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_6x5_KHR:
		return astc.NewRGBA_6x5("GL_COMPRESSED_RGBA_ASTC_6x5_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_6x6_KHR:
		return astc.NewRGBA_6x6("GL_COMPRESSED_RGBA_ASTC_6x6_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_8x5_KHR:
		return astc.NewRGBA_8x5("GL_COMPRESSED_RGBA_ASTC_8x5_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_8x6_KHR:
		return astc.NewRGBA_8x6("GL_COMPRESSED_RGBA_ASTC_8x6_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_8x8_KHR:
		return astc.NewRGBA_8x8("GL_COMPRESSED_RGBA_ASTC_8x8_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_10x5_KHR:
		return astc.NewRGBA_10x5("GL_COMPRESSED_RGBA_ASTC_10x5_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_10x6_KHR:
		return astc.NewRGBA_10x6("GL_COMPRESSED_RGBA_ASTC_10x6_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_10x8_KHR:
		return astc.NewRGBA_10x8("GL_COMPRESSED_RGBA_ASTC_10x8_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_10x10_KHR:
		return astc.NewRGBA_10x10("GL_COMPRESSED_RGBA_ASTC_10x10_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_12x10_KHR:
		return astc.NewRGBA_12x10("GL_COMPRESSED_RGBA_ASTC_12x10_KHR"), nil
	case GLenum_GL_COMPRESSED_RGBA_ASTC_12x12_KHR:
		return astc.NewRGBA_12x12("GL_COMPRESSED_RGBA_ASTC_12x12_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:
		return astc.NewSRGB8_ALPHA8_4x4("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:
		return astc.NewSRGB8_ALPHA8_5x4("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:
		return astc.NewSRGB8_ALPHA8_5x5("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:
		return astc.NewSRGB8_ALPHA8_6x5("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:
		return astc.NewSRGB8_ALPHA8_6x6("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:
		return astc.NewSRGB8_ALPHA8_8x5("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:
		return astc.NewSRGB8_ALPHA8_8x6("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:
		return astc.NewSRGB8_ALPHA8_8x8("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:
		return astc.NewSRGB8_ALPHA8_10x5("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:
		return astc.NewSRGB8_ALPHA8_10x6("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:
		return astc.NewSRGB8_ALPHA8_10x8("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:
		return astc.NewSRGB8_ALPHA8_10x10("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:
		return astc.NewSRGB8_ALPHA8_12x10("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:
		return astc.NewSRGB8_ALPHA8_12x12("GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR"), nil

	// ATC
	case GLenum_GL_ATC_RGB_AMD:
		return image.NewATC_RGB_AMD("GL_ATC_RGB_AMD"), nil
	case GLenum_GL_ATC_RGBA_EXPLICIT_ALPHA_AMD:
		return image.NewATC_RGBA_EXPLICIT_ALPHA_AMD("GL_ATC_RGBA_EXPLICIT_ALPHA_AMD"), nil
	case GLenum_GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD:
		return image.NewATC_RGBA_INTERPOLATED_ALPHA_AMD("GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD"), nil

	// ETC
	case GLenum_GL_COMPRESSED_R11_EAC:
		return image.NewETC2_R_U11_NORM("GL_COMPRESSED_R11_EAC"), nil
	case GLenum_GL_COMPRESSED_SIGNED_R11_EAC:
		return image.NewETC2_R_S11_NORM("GL_COMPRESSED_SIGNED_R11_EAC"), nil
	case GLenum_GL_COMPRESSED_RG11_EAC:
		return image.NewETC2_RG_U11_NORM("GL_COMPRESSED_RG11_EAC"), nil
	case GLenum_GL_COMPRESSED_SIGNED_RG11_EAC:
		return image.NewETC2_RG_S11_NORM("GL_COMPRESSED_SIGNED_RG11_EAC"), nil
	case GLenum_GL_COMPRESSED_RGB8_ETC2:
		return image.NewETC2_RGB_U8_NORM("GL_COMPRESSED_RGB8_ETC2"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ETC2:
		return image.NewETC2_SRGB_U8_NORM("GL_COMPRESSED_SRGB8_ETC2"), nil
	case GLenum_GL_COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2:
		return image.NewETC2_RGBA_U8U8U8U1_NORM("GL_COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2"), nil
	case GLenum_GL_COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2:
		return image.NewETC2_SRGBA_U8U8U8U1_NORM("GL_COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2"), nil
	case GLenum_GL_COMPRESSED_RGBA8_ETC2_EAC:
		return image.NewETC2_RGBA_U8_NORM("GL_COMPRESSED_RGBA8_ETC2_EAC"), nil
	case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:
		return image.NewETC2_SRGBA_U8_NORM("GL_COMPRESSED_SRGB8_ALPHA8_ETC2_EAC"), nil

	// S3TC
	case GLenum_GL_COMPRESSED_RGB_S3TC_DXT1_EXT:
		return image.NewS3_DXT1_RGB("GL_COMPRESSED_RGB_S3TC_DXT1_EXT"), nil
	case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT1_EXT:
		return image.NewS3_DXT1_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT1_EXT"), nil
	case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT3_EXT:
		return image.NewS3_DXT3_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT3_EXT"), nil
	case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT5_EXT:
		return image.NewS3_DXT5_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT5_EXT"), nil
	}

	return nil, fmt.Errorf("Unsupported compressed format: %s", format)
}
