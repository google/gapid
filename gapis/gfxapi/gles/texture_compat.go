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
	"context"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

var luminanceSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_RED,
	GLenum_GL_GREEN: GLenum_GL_RED,
	GLenum_GL_BLUE:  GLenum_GL_RED,
	GLenum_GL_ALPHA: GLenum_GL_ONE,
	GLenum_GL_ONE:   GLenum_GL_ONE,
	GLenum_GL_ZERO:  GLenum_GL_ZERO,
}
var alphaSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_ZERO,
	GLenum_GL_GREEN: GLenum_GL_ZERO,
	GLenum_GL_BLUE:  GLenum_GL_ZERO,
	GLenum_GL_ALPHA: GLenum_GL_RED,
	GLenum_GL_ONE:   GLenum_GL_ONE,
	GLenum_GL_ZERO:  GLenum_GL_ZERO,
}
var luminanceAlphaSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_RED,
	GLenum_GL_GREEN: GLenum_GL_RED,
	GLenum_GL_BLUE:  GLenum_GL_RED,
	GLenum_GL_ALPHA: GLenum_GL_GREEN,
	GLenum_GL_ONE:   GLenum_GL_ONE,
	GLenum_GL_ZERO:  GLenum_GL_ZERO,
}
var noSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_RED,
	GLenum_GL_GREEN: GLenum_GL_GREEN,
	GLenum_GL_BLUE:  GLenum_GL_BLUE,
	GLenum_GL_ALPHA: GLenum_GL_ALPHA,
	GLenum_GL_ONE:   GLenum_GL_ONE,
	GLenum_GL_ZERO:  GLenum_GL_ZERO,
}

// getLuminanceAlphaSwizzle emulates Luminance/Alpha by mapping the channels to Red/Green.
func getLuminanceAlphaSwizzle(internalformat GLenum) map[GLenum]GLenum {
	switch internalformat {
	case GLenum_GL_LUMINANCE,
		GLenum_GL_LUMINANCE8_EXT,
		GLenum_GL_LUMINANCE16F_EXT,
		GLenum_GL_LUMINANCE32F_EXT:
		return luminanceSwizzle
	case GLenum_GL_ALPHA,
		GLenum_GL_ALPHA8_EXT,
		GLenum_GL_ALPHA16F_EXT,
		GLenum_GL_ALPHA32F_EXT:
		return alphaSwizzle
	case GLenum_GL_LUMINANCE_ALPHA,
		GLenum_GL_LUMINANCE8_ALPHA8_EXT,
		GLenum_GL_LUMINANCE_ALPHA16F_EXT,
		GLenum_GL_LUMINANCE_ALPHA32F_EXT:
		return luminanceAlphaSwizzle
	default:
		return noSwizzle
	}
}

type textureCompat struct {
	f   features
	v   *Version
	ctx context.Context

	// Original user-defined swizzle which would be used without compatibility layer.
	// (GL_TEXTURE_SWIZZLE_{R,G,B,A}, Texture) -> GL_{RED,GREEN,BLUE,ALPHA,ONE,ZERO}
	origSwizzle map[GLenum]map[*Texture]GLenum

	// Compatibility component remapping needed to support luminance/alpha formats.
	// Texture -> (GL_{RED,GREEN,BLUE,ALPHA,ONE,ZERO} -> GL_{RED,GREEN,BLUE,ALPHA,ONE,ZERO})
	compatSwizzle map[*Texture]map[GLenum]GLenum
}

// getSwizzle returns the original user-defined swizzle and the current swizzle from state.
func (tc *textureCompat) getSwizzle(t *Texture, parameter GLenum) (orig, curr GLenum) {
	var init GLenum
	switch parameter {
	case GLenum_GL_TEXTURE_SWIZZLE_R:
		init, curr = GLenum_GL_RED, t.SwizzleR
	case GLenum_GL_TEXTURE_SWIZZLE_G:
		init, curr = GLenum_GL_GREEN, t.SwizzleG
	case GLenum_GL_TEXTURE_SWIZZLE_B:
		init, curr = GLenum_GL_BLUE, t.SwizzleB
	case GLenum_GL_TEXTURE_SWIZZLE_A:
		init, curr = GLenum_GL_ALPHA, t.SwizzleA
	}
	if orig, ok := tc.origSwizzle[parameter][t]; ok {
		return orig, curr
	}
	return init, curr
}

func (tc *textureCompat) writeCompatSwizzle(ctx context.Context, t *Texture, parameter GLenum, out transform.Writer) {
	target := t.Kind
	orig, curr := tc.getSwizzle(t, parameter)
	compat := orig
	if compatSwizzle, ok := tc.compatSwizzle[t]; ok {
		compat = compatSwizzle[compat]
	}
	if compat != curr {
		out.MutateAndWrite(ctx, atom.NoID, NewGlTexParameteri(target, parameter, GLint(compat)))
	}
}

// Common handler for all glTex* methods.
// Arguments may be null if the given method does not use them.
func (tc *textureCompat) convertFormat(target GLenum, internalformat, format, componentType *GLenum, out transform.Writer) {
	if tc.v.IsES {
		return
	}

	if internalformat != nil {
		s := out.State()

		switch target {
		case GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_X,
			GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Y, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Y,
			GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Z, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Z:
			target = GLenum_GL_TEXTURE_CUBE_MAP
		}

		// Set swizzles to emulate luminance/alpha formats. We need to do this before we convert the format.
		if t, err := subGetBoundTextureOrErrorInvalidEnum(tc.ctx, nil, nil, s, GetState(s), nil, target); err == nil {
			tc.compatSwizzle[t] = getLuminanceAlphaSwizzle(*internalformat)
			tc.writeCompatSwizzle(tc.ctx, t, GLenum_GL_TEXTURE_SWIZZLE_R, out)
			tc.writeCompatSwizzle(tc.ctx, t, GLenum_GL_TEXTURE_SWIZZLE_G, out)
			tc.writeCompatSwizzle(tc.ctx, t, GLenum_GL_TEXTURE_SWIZZLE_B, out)
			tc.writeCompatSwizzle(tc.ctx, t, GLenum_GL_TEXTURE_SWIZZLE_A, out)
		}

		if componentType != nil {
			*internalformat = getSizedInternalFormat(*internalformat, *componentType)
		} else {
			*internalformat = getSizedInternalFormat(*internalformat, GLenum_GL_UNSIGNED_BYTE)
		}

		// Compressed formats are replaced by RGBA8
		// TODO: What about SRGB?
		if isCompressedFormat(*internalformat) {
			if _, supported := tc.f.compressedTextureFormats[*internalformat]; !supported {
				*internalformat = GLenum_GL_RGBA8
			}
		}
	}

	if format != nil {
		// Luminance/Alpha is not supported on desktop so convert it to R/G.
		switch *format {
		case GLenum_GL_LUMINANCE, GLenum_GL_ALPHA:
			*format = GLenum_GL_RED
		case GLenum_GL_LUMINANCE_ALPHA:
			*format = GLenum_GL_RG
		}
	}

	if componentType != nil {
		// Half-float is a core feature on desktop (with different enum value)
		if *componentType == GLenum_GL_HALF_FLOAT_OES {
			*componentType = GLenum_GL_HALF_FLOAT
		}
	}
}

func (tc *textureCompat) postTexParameter(target, parameter GLenum, out transform.Writer) {
	if tc.v.IsES {
		return
	}

	s := out.State()
	switch parameter {
	case GLenum_GL_TEXTURE_SWIZZLE_R, GLenum_GL_TEXTURE_SWIZZLE_G, GLenum_GL_TEXTURE_SWIZZLE_B, GLenum_GL_TEXTURE_SWIZZLE_A:
		if t, err := subGetBoundTextureOrErrorInvalidEnum(tc.ctx, nil, nil, s, GetState(s), nil, target); err == nil {
			_, curr := tc.getSwizzle(t, parameter)
			// The tex parameter was recently mutated, so set the original swizzle from current state.
			tc.origSwizzle[parameter][t] = curr
			// Combine the original and compat swizzles and write out the commands to set it.
			tc.writeCompatSwizzle(tc.ctx, t, parameter, out)
		}
	case GLenum_GL_TEXTURE_SWIZZLE_RGBA:
		log.E(tc.ctx, "Unexpected GL_TEXTURE_SWIZZLE_RGBA")
	}
}

// decompressTexImage2D writes a glTexImage2D using the decompressed data for
// the given glCompressedTexImage2D.
func decompressTexImage2D(ctx context.Context, i atom.ID, a *GlCompressedTexImage2D, s *gfxapi.State, out transform.Writer) error {
	ctx = log.Enter(ctx, "decompressTexImage2D")
	c := GetContext(s)

	data := a.Data
	if pb := c.BoundBuffers.PixelUnpackBuffer; pb != 0 {
		base := a.Data.Address
		data = TexturePointer(c.Instances.Buffers[pb].Data.Index(base, s))
		out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, 0))
		defer out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, pb))
	} else {
		a.Extras().Observations().ApplyReads(s.Memory[memory.ApplicationPool])
	}

	src := image.Info2D{
		Data:   image.NewID(data.Slice(0, uint64(a.ImageSize), s).ResourceID(ctx, s)),
		Width:  uint32(a.Width),
		Height: uint32(a.Height),
		Format: newImgfmt(a.Format, 0).asImageOrPanic(),
	}
	dst, err := src.ConvertTo(ctx, image.RGBA_U8_NORM)
	if err != nil {
		return err
	}

	dstSize := a.Width * a.Height * 4

	tmp := atom.Must(atom.Alloc(ctx, s, uint64(dstSize)))
	out.MutateAndWrite(ctx, i, NewGlTexImage2D(
		a.Target,
		a.Level,
		GLint(GLenum_GL_RGBA8),
		a.Width,
		a.Height,
		a.Border,
		GLenum_GL_RGBA,
		GLenum_GL_UNSIGNED_BYTE,
		tmp.Ptr(),
	).AddRead(tmp.Range(), dst.Data.ID()))
	tmp.Free()

	return nil
}

// decompressTexSubImage2D writes a glTexSubImage2D using the decompressed data for
// the given glCompressedTexSubImage2D.
func decompressTexSubImage2D(ctx context.Context, i atom.ID, a *GlCompressedTexSubImage2D, s *gfxapi.State, out transform.Writer) error {
	ctx = log.Enter(ctx, "decompressTexSubImage2D")
	c := GetContext(s)

	data := a.Data
	if pb := c.BoundBuffers.PixelUnpackBuffer; pb != 0 {
		base := a.Data.Address
		data = TexturePointer(c.Instances.Buffers[pb].Data.Index(base, s))
		out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, 0))
		defer out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, pb))
	} else {
		a.Extras().Observations().ApplyReads(s.Memory[memory.ApplicationPool])
	}

	src := image.Info2D{
		Data:   image.NewID(data.Slice(0, uint64(a.ImageSize), s).ResourceID(ctx, s)),
		Width:  uint32(a.Width),
		Height: uint32(a.Height),
		Format: newImgfmtFromSIF(a.Format).asImageOrPanic(),
	}
	dst, err := src.ConvertTo(ctx, image.RGBA_U8_NORM)
	if err != nil {
		return err
	}

	dstSize := a.Width * a.Height * 4

	tmp := atom.Must(atom.Alloc(ctx, s, uint64(dstSize)))
	out.MutateAndWrite(ctx, i, NewGlTexSubImage2D(
		a.Target,
		a.Level,
		a.Xoffset,
		a.Yoffset,
		a.Width,
		a.Height,
		GLenum_GL_RGBA,
		GLenum_GL_UNSIGNED_BYTE,
		tmp.Ptr(),
	).AddRead(tmp.Range(), dst.Data.ID()))
	tmp.Free()

	return nil
}

// getSupportedCompressedTextureFormats returns the set of supported compressed
// texture formats for a given extension list.
func getSupportedCompressedTextureFormats(extensions extensions) map[GLenum]struct{} {
	supported := map[GLenum]struct{}{}
	for extension := range extensions {
		for _, format := range getExtensionTextureFormats(extension) {
			supported[format] = struct{}{}
		}
	}
	return supported
}

// getExtensionTextureFormats returns the list of compressed texture formats
// enabled by a given extension
func getExtensionTextureFormats(extension string) []GLenum {
	switch extension {
	case "GL_AMD_compressed_ATC_texture":
		return []GLenum{
			GLenum_GL_ATC_RGB_AMD,
			GLenum_GL_ATC_RGBA_EXPLICIT_ALPHA_AMD,
			GLenum_GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD,
		}
	case "GL_OES_compressed_ETC1_RGB8_texture":
		return []GLenum{
			GLenum_GL_ETC1_RGB8_OES,
		}
	case "GL_EXT_texture_compression_dxt1":
		return []GLenum{
			GLenum_GL_COMPRESSED_RGB_S3TC_DXT1_EXT,
			GLenum_GL_COMPRESSED_RGBA_S3TC_DXT1_EXT,
		}
	case "GL_EXT_texture_compression_s3tc", "GL_NV_texture_compression_s3tc":
		return []GLenum{
			GLenum_GL_COMPRESSED_RGB_S3TC_DXT1_EXT,
			GLenum_GL_COMPRESSED_RGBA_S3TC_DXT1_EXT,
			GLenum_GL_COMPRESSED_RGBA_S3TC_DXT3_EXT,
			GLenum_GL_COMPRESSED_RGBA_S3TC_DXT5_EXT,
		}
	case "GL_KHR_texture_compression_astc_ldr":
		return []GLenum{
			GLenum_GL_COMPRESSED_RGBA_ASTC_4x4_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_5x4_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_5x5_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_6x5_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_6x6_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_8x5_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_8x6_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_8x8_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_10x5_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_10x6_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_10x8_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_10x10_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_12x10_KHR,
			GLenum_GL_COMPRESSED_RGBA_ASTC_12x12_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR,
			GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR,
		}
	case "GL_EXT_texture_compression_latc", "GL_NV_texture_compression_latc":
		return []GLenum{
			GLenum_GL_COMPRESSED_LUMINANCE_LATC1_EXT,
			GLenum_GL_COMPRESSED_SIGNED_LUMINANCE_LATC1_EXT,
			GLenum_GL_COMPRESSED_LUMINANCE_ALPHA_LATC2_EXT,
			GLenum_GL_COMPRESSED_SIGNED_LUMINANCE_ALPHA_LATC2_EXT,
		}
	default:
		return []GLenum{}
	}
}

func isCompressedFormat(internalformat GLenum) bool {
	switch internalformat {
	case
		GLenum_GL_ATC_RGBA_EXPLICIT_ALPHA_AMD,
		GLenum_GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD,
		GLenum_GL_ATC_RGB_AMD,
		GLenum_GL_COMPRESSED_LUMINANCE_ALPHA_LATC2_EXT,
		GLenum_GL_COMPRESSED_LUMINANCE_LATC1_EXT,
		GLenum_GL_COMPRESSED_RG11_EAC,
		GLenum_GL_COMPRESSED_RGB8_ETC2,
		GLenum_GL_COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2,
		GLenum_GL_COMPRESSED_RGBA8_ETC2_EAC,
		GLenum_GL_COMPRESSED_RGBA_ASTC_10x10_KHR,
		GLenum_GL_COMPRESSED_RGBA_ASTC_10x5,
		GLenum_GL_COMPRESSED_RGBA_ASTC_10x6,
		GLenum_GL_COMPRESSED_RGBA_ASTC_10x8,
		GLenum_GL_COMPRESSED_RGBA_ASTC_12x10,
		GLenum_GL_COMPRESSED_RGBA_ASTC_12x12,
		GLenum_GL_COMPRESSED_RGBA_ASTC_4x4,
		GLenum_GL_COMPRESSED_RGBA_ASTC_5x4,
		GLenum_GL_COMPRESSED_RGBA_ASTC_5x5,
		GLenum_GL_COMPRESSED_RGBA_ASTC_6x5,
		GLenum_GL_COMPRESSED_RGBA_ASTC_6x6,
		GLenum_GL_COMPRESSED_RGBA_ASTC_8x5,
		GLenum_GL_COMPRESSED_RGBA_ASTC_8x6,
		GLenum_GL_COMPRESSED_RGBA_ASTC_8x8,
		GLenum_GL_COMPRESSED_RGBA_S3TC_DXT1_EXT,
		GLenum_GL_COMPRESSED_RGBA_S3TC_DXT3_EXT,
		GLenum_GL_COMPRESSED_RGBA_S3TC_DXT5_EXT,
		GLenum_GL_COMPRESSED_RGB_S3TC_DXT1_EXT,
		GLenum_GL_COMPRESSED_SIGNED_LUMINANCE_ALPHA_LATC2_EXT,
		GLenum_GL_COMPRESSED_SIGNED_LUMINANCE_LATC1_EXT,
		GLenum_GL_COMPRESSED_SIGNED_R11_EAC,
		GLenum_GL_COMPRESSED_SIGNED_RG11_EAC,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8,
		GLenum_GL_COMPRESSED_SRGB8_ALPHA8_ETC2_EAC,
		GLenum_GL_COMPRESSED_SRGB8_ETC2,
		GLenum_GL_COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2,
		GLenum_GL_ETC1_RGB8_OES:
		return true
	}
	return false
}
