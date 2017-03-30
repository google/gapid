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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/stream"
)

type imgfmt struct {
	sif  GLenum // sized internal format (RGB565,        RGBA8,         R16UI,          ...)
	base GLenum // base format           (RGB,           RGBA,          RED,            ...)
	ty   GLenum // pixel type            (UNSIGNED_BYTE, UNSIGNED_BYTE, UNSIGNED_SHORT, ...)
}

func newImgfmt(base, ty GLenum) imgfmt {
	return imgfmt{getSizedInternalFormat(base, ty), base, ty}
}

func newImgfmtFromSIF(sif GLenum) imgfmt {
	base, ty := extractSizedInternalFormat(sif)
	return imgfmt{sif, base, ty}
}

// getSizedInternalFormat returns the sized internal format
// (renderbuffer storage format) for the given base format and component type.
func getSizedInternalFormat(baseFmt, componentType GLenum) GLenum {
	switch baseFmt {
	// ES and desktop disagree how unsized internal formats are represented (floating point in particular),
	// so always explicitly use one of the sized internal formats.
	case GLenum_GL_RED:
		return getSizedInternalFormatFromTypeCount(componentType, 1)
	case GLenum_GL_RG:
		return getSizedInternalFormatFromTypeCount(componentType, 2)
	case GLenum_GL_RGB, GLenum_GL_BGR:
		return getSizedInternalFormatFromTypeCount(componentType, 3)
	case GLenum_GL_RGBA, GLenum_GL_BGRA:
		return getSizedInternalFormatFromTypeCount(componentType, 4)

	// Luminance/Alpha is not supported on desktop so convert it to R/G. (enums defined in EXT_texture_storage)
	case GLenum_GL_LUMINANCE, GLenum_GL_ALPHA:
		return getSizedInternalFormatFromTypeCount(componentType, 1)
	case GLenum_GL_LUMINANCE_ALPHA:
		return getSizedInternalFormatFromTypeCount(componentType, 2)
	case GLenum_GL_ALPHA8_EXT, GLenum_GL_LUMINANCE8_EXT:
		return GLenum_GL_R8
	case GLenum_GL_LUMINANCE8_ALPHA8_EXT:
		return GLenum_GL_RG8
	case GLenum_GL_ALPHA16F_EXT, GLenum_GL_LUMINANCE16F_EXT:
		return GLenum_GL_R16F
	case GLenum_GL_LUMINANCE_ALPHA16F_EXT:
		return GLenum_GL_RG16F
	case GLenum_GL_ALPHA32F_EXT, GLenum_GL_LUMINANCE32F_EXT:
		return GLenum_GL_R32F
	case GLenum_GL_LUMINANCE_ALPHA32F_EXT:
		return GLenum_GL_RG32F

	case GLenum_GL_RGB565: // Not supported in GL 3.2
		return GLenum_GL_RGB8
	case GLenum_GL_RGB10_A2UI: // Not supported in GL 3.2
		return GLenum_GL_RGBA16UI
	case GLenum_GL_STENCIL_INDEX8:
		// TODO: May not be supported on desktop.
	}

	return baseFmt
}

// extractSizedInternalFormat returns the base format and component type for the
// given sized internal format (renderbuffer storage format).
func extractSizedInternalFormat(sif GLenum) (base, ty GLenum) {
	base, _ = subImageFormat(nil, nil, nil, nil, nil, nil, sif)
	ty, _ = subImageType(nil, nil, nil, nil, nil, nil, sif)
	return base, ty
}

var sizedInternalFormats8 = [4]GLenum{GLenum_GL_R8, GLenum_GL_RG8, GLenum_GL_RGB8, GLenum_GL_RGBA8}
var sizedInternalFormats16F = [4]GLenum{GLenum_GL_R16F, GLenum_GL_RG16F, GLenum_GL_RGB16F, GLenum_GL_RGBA16F}
var sizedInternalFormats32F = [4]GLenum{GLenum_GL_R32F, GLenum_GL_RG32F, GLenum_GL_RGB32F, GLenum_GL_RGBA32F}

// getSizedInternalFormatFromTypeCount returns internal texture format
// appropriate to store given component type and count.
func getSizedInternalFormatFromTypeCount(componentType GLenum, componentCount uint32) GLenum {
	// TODO: Handle integer formats.
	switch componentType {
	case GLenum_GL_FLOAT:
		return sizedInternalFormats32F[componentCount-1]
	case GLenum_GL_HALF_FLOAT, GLenum_GL_HALF_FLOAT_OES:
		return sizedInternalFormats16F[componentCount-1]
	case GLenum_GL_UNSIGNED_INT_2_10_10_10_REV:
		return GLenum_GL_RGB10_A2
	}
	return sizedInternalFormats8[componentCount-1]
}

// asImage returns the *image.Format for the given imgfmt, or panics if the
// format cannot be matched.
// TODO: We shouldn't be panicing in this package.
// Handle errors gracefully and remove.
func (f imgfmt) asImageOrPanic() *image.Format {
	i, e := f.asImage()
	if e != nil {
		panic(e)
	}
	return i
}

// getStreamChannels converts GL channel enum to stream.Channel array.
func getStreamChannels(glChannels GLenum) (channels []stream.Channel, err error) {
	switch glChannels {
	case GLenum_GL_RED, GLenum_GL_RED_INTEGER:
		return []stream.Channel{stream.Channel_Red}, nil
	case GLenum_GL_RG, GLenum_GL_RG_INTEGER:
		return []stream.Channel{stream.Channel_Red, stream.Channel_Green}, nil
	case GLenum_GL_RGB, GLenum_GL_RGB_INTEGER:
		return []stream.Channel{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue}, nil
	case GLenum_GL_RGBA, GLenum_GL_RGBA_INTEGER:
		return []stream.Channel{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}, nil
	case GLenum_GL_DEPTH_STENCIL:
		return []stream.Channel{stream.Channel_Depth, stream.Channel_Stencil}, nil
	case GLenum_GL_DEPTH, GLenum_GL_DEPTH_COMPONENT:
		return []stream.Channel{stream.Channel_Depth}, nil
	case GLenum_GL_STENCIL, GLenum_GL_STENCIL_INDEX:
		return []stream.Channel{stream.Channel_Stencil}, nil
	case GLenum_GL_ALPHA, GLenum_GL_ALPHA_INTEGER_EXT:
		return []stream.Channel{stream.Channel_Alpha}, nil
	case GLenum_GL_LUMINANCE, GLenum_GL_LUMINANCE_INTEGER_EXT:
		return []stream.Channel{stream.Channel_Luminance}, nil
	case GLenum_GL_LUMINANCE_ALPHA, GLenum_GL_LUMINANCE_ALPHA_INTEGER_EXT:
		return []stream.Channel{stream.Channel_Luminance, stream.Channel_Alpha}, nil
	default:
		return nil, fmt.Errorf("Unsupported channel type: ", glChannels)
	}
}

// sampleAsFloat returns true if the channel's value is returned as float in shader.
func sampleAsFloat(glChannels GLenum, channelIndex int) bool {
	switch glChannels {
	case GLenum_GL_RED_INTEGER, GLenum_GL_RG_INTEGER, GLenum_GL_RGB_INTEGER, GLenum_GL_RGBA_INTEGER,
		GLenum_GL_ALPHA_INTEGER_EXT, GLenum_GL_LUMINANCE_INTEGER_EXT, GLenum_GL_LUMINANCE_ALPHA_INTEGER_EXT,
		GLenum_GL_STENCIL, GLenum_GL_STENCIL_INDEX:
		return false // Integer type.
	case GLenum_GL_DEPTH_STENCIL:
		return channelIndex == 0 // Only depth channel (index 0) is represented by float.
	}
	return true // Float type.
}

// getUncompressedStreamFormat returns the decoding format which can be used to read single pixel.
func getUncompressedStreamFormat(glChannels, glDataType GLenum) (format *stream.Format, err error) {
	channels, err := getStreamChannels(glChannels)
	if err != nil {
		return nil, err
	}

	// Helper method to build the format.
	format = &stream.Format{}
	addComponent := func(channelIndex int, datatype *stream.DataType) {
		channel := stream.Channel_Undefined // Padding field
		if 0 <= channelIndex && channelIndex < len(channels) {
			channel = channels[channelIndex]
		}
		sampling := stream.Linear
		if datatype.IsInteger() && sampleAsFloat(glChannels, channelIndex) {
			sampling = stream.LinearNormalized // Convert int to float
		}
		format.Components = append(format.Components, &stream.Component{datatype, sampling, channel})
	}

	// Read the components in increasing memory order (assuming little-endian architecture).
	// Note that the GL names are based on big-endian, so the order is generally backwards.
	switch glDataType {
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
	// TODO: This requires some extra work for the shared exponent.
	// case GLenum_GL_UNSIGNED_INT_5_9_9_9_REV:
	// 	addComponent(0, &stream.U9)
	// 	addComponent(1, &stream.U9)
	// 	addComponent(2, &stream.U9)
	// 	addComponent(3, &stream.U5)
	case GLenum_GL_FLOAT_32_UNSIGNED_INT_24_8_REV:
		addComponent(0, &stream.F32)
		addComponent(1, &stream.U8)
		addComponent(-1, &stream.U24)
	default:
		return nil, fmt.Errorf("Unsupported data type: ", glDataType)
	}
	return format, nil
}

// asImage returns the *image.Format for the given imgfmt.
func (f imgfmt) asImage() (*image.Format, error) {
	ty := f.ty
	switch f.base {
	case GLenum_GL_RGB:
		switch ty {
		case GLenum_GL_ATC_RGB_AMD:
			return image.NewATC_RGB_AMD("GL_ATC_RGB_AMD"), nil
		case GLenum_GL_ETC1_RGB8_OES:
			return image.NewETC1_RGB8("GL_ETC1_RGB8_OES"), nil
		case GLenum_GL_COMPRESSED_RGB8_ETC2:
			return image.NewETC2_RGB8("GL_COMPRESSED_RGB8_ETC2"), nil
		case GLenum_GL_COMPRESSED_RGB_S3TC_DXT1_EXT:
			return image.NewS3_DXT1_RGB("GL_COMPRESSED_RGB_S3TC_DXT1_EXT"), nil
		}
	case GLenum_GL_RGBA:
		switch ty {
		case GLenum_GL_ATC_RGBA_EXPLICIT_ALPHA_AMD:
			return image.NewATC_RGBA_EXPLICIT_ALPHA_AMD("GL_ATC_RGBA_EXPLICIT_ALPHA_AMD"), nil
		case GLenum_GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD:
			return image.NewATC_RGBA_INTERPOLATED_ALPHA_AMD("GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD"), nil
		case GLenum_GL_COMPRESSED_RGBA8_ETC2_EAC:
			return image.NewETC2_RGBA8_EAC("GL_COMPRESSED_RGBA8_ETC2_EAC"), nil
		case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT1_EXT:
			return image.NewS3_DXT1_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT1_EXT"), nil
		case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT3_EXT:
			return image.NewS3_DXT3_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT3_EXT"), nil
		case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT5_EXT:
			return image.NewS3_DXT5_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT5_EXT"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_4x4_KHR:
			return image.NewASTC_RGBA_4x4("GLenum_COMPRESSED_RGBA_ASTC_4x4_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_5x4_KHR:
			return image.NewASTC_RGBA_5x4("GLenum_COMPRESSED_RGBA_ASTC_5x4_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_5x5_KHR:
			return image.NewASTC_RGBA_5x5("GLenum_COMPRESSED_RGBA_ASTC_5x5_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_6x5_KHR:
			return image.NewASTC_RGBA_6x5("GLenum_COMPRESSED_RGBA_ASTC_6x5_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_6x6_KHR:
			return image.NewASTC_RGBA_6x6("GLenum_COMPRESSED_RGBA_ASTC_6x6_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_8x5_KHR:
			return image.NewASTC_RGBA_8x5("GLenum_COMPRESSED_RGBA_ASTC_8x5_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_8x6_KHR:
			return image.NewASTC_RGBA_8x6("GLenum_COMPRESSED_RGBA_ASTC_8x6_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_8x8_KHR:
			return image.NewASTC_RGBA_8x8("GLenum_COMPRESSED_RGBA_ASTC_8x8_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_10x5_KHR:
			return image.NewASTC_RGBA_10x5("GLenum_COMPRESSED_RGBA_ASTC_10x5_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_10x6_KHR:
			return image.NewASTC_RGBA_10x6("GLenum_COMPRESSED_RGBA_ASTC_10x6_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_10x8_KHR:
			return image.NewASTC_RGBA_10x8("GLenum_COMPRESSED_RGBA_ASTC_10x8_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_10x10_KHR:
			return image.NewASTC_RGBA_10x10("GLenum_COMPRESSED_RGBA_ASTC_10x10_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_12x10_KHR:
			return image.NewASTC_RGBA_12x10("GLenum_COMPRESSED_RGBA_ASTC_12x10_KHR"), nil
		case GLenum_GL_COMPRESSED_RGBA_ASTC_12x12_KHR:
			return image.NewASTC_RGBA_12x12("GLenum_COMPRESSED_RGBA_ASTC_12x12_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:
			return image.NewASTC_SRGB8_ALPHA8_4x4("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:
			return image.NewASTC_SRGB8_ALPHA8_5x4("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:
			return image.NewASTC_SRGB8_ALPHA8_5x5("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:
			return image.NewASTC_SRGB8_ALPHA8_6x5("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:
			return image.NewASTC_SRGB8_ALPHA8_6x6("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:
			return image.NewASTC_SRGB8_ALPHA8_8x5("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:
			return image.NewASTC_SRGB8_ALPHA8_8x6("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:
			return image.NewASTC_SRGB8_ALPHA8_8x8("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:
			return image.NewASTC_SRGB8_ALPHA8_10x5("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:
			return image.NewASTC_SRGB8_ALPHA8_10x6("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:
			return image.NewASTC_SRGB8_ALPHA8_10x8("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:
			return image.NewASTC_SRGB8_ALPHA8_10x10("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:
			return image.NewASTC_SRGB8_ALPHA8_12x10("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR"), nil
		case GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:
			return image.NewASTC_SRGB8_ALPHA8_12x12("GLenum_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR"), nil
		}
	case GLenum_GL_ATC_RGB_AMD:
		return image.NewATC_RGB_AMD("GL_ATC_RGB_AMD"), nil
	case GLenum_GL_ATC_RGBA_EXPLICIT_ALPHA_AMD:
		return image.NewATC_RGBA_EXPLICIT_ALPHA_AMD("GL_ATC_RGBA_EXPLICIT_ALPHA_AMD"), nil
	case GLenum_GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD:
		return image.NewATC_RGBA_INTERPOLATED_ALPHA_AMD("GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD"), nil
	case GLenum_GL_ETC1_RGB8_OES:
		return image.NewETC1_RGB8("GL_ETC1_RGB8_OES"), nil
	case GLenum_GL_COMPRESSED_RGB8_ETC2:
		return image.NewETC2_RGB8("GL_COMPRESSED_RGB8_ETC2"), nil
	case GLenum_GL_COMPRESSED_RGBA8_ETC2_EAC:
		return image.NewETC2_RGBA8_EAC("GL_COMPRESSED_RGBA8_ETC2_EAC"), nil
	case GLenum_GL_COMPRESSED_RGB_S3TC_DXT1_EXT:
		return image.NewS3_DXT1_RGB("GL_COMPRESSED_RGB_S3TC_DXT1_EXT"), nil
	case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT1_EXT:
		return image.NewS3_DXT1_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT1_EXT"), nil
	case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT3_EXT:
		return image.NewS3_DXT3_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT3_EXT"), nil
	case GLenum_GL_COMPRESSED_RGBA_S3TC_DXT5_EXT:
		return image.NewS3_DXT5_RGBA("GL_COMPRESSED_RGBA_S3TC_DXT5_EXT"), nil
	}

	if format, err := getUncompressedStreamFormat(f.base, f.ty); err == nil {
		return image.NewUncompressed(fmt.Sprint(format), format), nil
	}

	return nil, fmt.Errorf("Unsupported input format-type pair: (%s, %s)", f.base, ty)
}
