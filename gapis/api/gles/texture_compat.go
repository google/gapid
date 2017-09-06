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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

var luminanceSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_RED,
	GLenum_GL_GREEN: GLenum_GL_RED,
	GLenum_GL_BLUE:  GLenum_GL_RED,
	GLenum_GL_ALPHA: GLenum_GL_ONE,
}
var alphaSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_ZERO,
	GLenum_GL_GREEN: GLenum_GL_ZERO,
	GLenum_GL_BLUE:  GLenum_GL_ZERO,
	GLenum_GL_ALPHA: GLenum_GL_RED,
}
var luminanceAlphaSwizzle = map[GLenum]GLenum{
	GLenum_GL_RED:   GLenum_GL_RED,
	GLenum_GL_GREEN: GLenum_GL_RED,
	GLenum_GL_BLUE:  GLenum_GL_RED,
	GLenum_GL_ALPHA: GLenum_GL_GREEN,
}
var luminanceAlphaCompat = map[GLenum]struct {
	rgFormat      GLenum
	compatSwizzle map[GLenum]GLenum
}{
	GLenum_GL_LUMINANCE:              {GLenum_GL_RED, luminanceSwizzle},
	GLenum_GL_LUMINANCE8_EXT:         {GLenum_GL_R8, luminanceSwizzle},
	GLenum_GL_LUMINANCE16F_EXT:       {GLenum_GL_R16F, luminanceSwizzle},
	GLenum_GL_LUMINANCE32F_EXT:       {GLenum_GL_R32F, luminanceSwizzle},
	GLenum_GL_ALPHA:                  {GLenum_GL_RED, alphaSwizzle},
	GLenum_GL_ALPHA8_EXT:             {GLenum_GL_R8, alphaSwizzle},
	GLenum_GL_ALPHA16F_EXT:           {GLenum_GL_R16F, alphaSwizzle},
	GLenum_GL_ALPHA32F_EXT:           {GLenum_GL_R32F, alphaSwizzle},
	GLenum_GL_LUMINANCE_ALPHA:        {GLenum_GL_RG, luminanceAlphaSwizzle},
	GLenum_GL_LUMINANCE8_ALPHA8_EXT:  {GLenum_GL_RG8, luminanceAlphaSwizzle},
	GLenum_GL_LUMINANCE_ALPHA16F_EXT: {GLenum_GL_RG16F, luminanceAlphaSwizzle},
	GLenum_GL_LUMINANCE_ALPHA32F_EXT: {GLenum_GL_RG32F, luminanceAlphaSwizzle},
}

type textureCompat struct {
	f features
	v *Version

	// Original user-defined swizzle which would be used without compatibility layer.
	// (GL_TEXTURE_SWIZZLE_{R,G,B,A}, Texture) -> GL_{RED,GREEN,BLUE,ALPHA,ONE,ZERO}
	origSwizzle map[GLenum]map[*Texture]GLenum

	// Compatibility component remapping needed to support luminance/alpha formats.
	// Texture -> (GL_{RED,GREEN,BLUE,ALPHA} -> GL_{RED,GREEN,ONE,ZERO})
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

func (tc *textureCompat) writeCompatSwizzle(ctx context.Context, cb CommandBuilder, t *Texture, parameter GLenum, out transform.Writer, id api.CmdID) {
	target := t.Kind
	orig, curr := tc.getSwizzle(t, parameter)
	compat := orig
	if compatSwizzle, ok := tc.compatSwizzle[t]; ok {
		if v, ok := compatSwizzle[compat]; ok {
			compat = v
		}
	}
	if compat != curr {
		out.MutateAndWrite(ctx, id.Derived(), cb.GlTexParameteri(target, parameter, GLint(compat)))
	}
}

// Common handler for all glTex* methods.
// Arguments may be null if the given method does not use them.
func (tc *textureCompat) convertFormat(
	ctx context.Context,
	target GLenum,
	internalformat, format, componentType *GLenum,
	out transform.Writer,
	id api.CmdID,
	cmd api.Cmd) {

	if tc.v.IsES {
		return
	}

	// ES and desktop disagree how unsized internal formats are represented
	// (floats in particular), so always explicitly use one of the sized formats.
	if internalformat != nil && format != nil && componentType != nil && *internalformat == *format {
		*internalformat = getSizedFormatFromTuple(*internalformat, *componentType)
	}

	if internalformat != nil {
		s := out.State()

		switch target {
		case GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_X,
			GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Y, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Y,
			GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Z, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Z:
			target = GLenum_GL_TEXTURE_CUBE_MAP
		}

		// Luminance/Alpha is not supported on desktop so convert it to R/G.
		if t, err := subGetBoundTextureOrErrorInvalidEnum(ctx, nil, api.CmdNoID, nil, s, GetState(s), cmd.Thread(), nil, target); err == nil {
			if laCompat, ok := luminanceAlphaCompat[*internalformat]; ok {
				*internalformat = laCompat.rgFormat
				tc.compatSwizzle[t] = laCompat.compatSwizzle
			} else {
				// Remove the compat mapping and reset swizzles to the original values below.
				delete(tc.compatSwizzle, t)
			}
			cb := CommandBuilder{Thread: cmd.Thread()}
			tc.writeCompatSwizzle(ctx, cb, t, GLenum_GL_TEXTURE_SWIZZLE_R, out, id)
			tc.writeCompatSwizzle(ctx, cb, t, GLenum_GL_TEXTURE_SWIZZLE_G, out, id)
			tc.writeCompatSwizzle(ctx, cb, t, GLenum_GL_TEXTURE_SWIZZLE_B, out, id)
			tc.writeCompatSwizzle(ctx, cb, t, GLenum_GL_TEXTURE_SWIZZLE_A, out, id)
		}

		switch *internalformat {
		case GLenum_GL_BGRA8_EXT: // Not supported in GL 3.2
			// The GPU order of channels is transparent to us, so we can just use RGBA instead.
			*internalformat = GLenum_GL_RGBA8
		case GLenum_GL_RGB565: // Not supported in GL 3.2
			*internalformat = GLenum_GL_RGB8
		case GLenum_GL_RGB10_A2UI: // Not supported in GL 3.2
			*internalformat = GLenum_GL_RGBA16UI
		case GLenum_GL_STENCIL_INDEX8: // TODO: not supported on desktop.
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

func (tc *textureCompat) postTexParameter(ctx context.Context, target, parameter GLenum, out transform.Writer, id api.CmdID, cmd api.Cmd) {
	if tc.v.IsES {
		return
	}

	s := out.State()
	switch parameter {
	case GLenum_GL_TEXTURE_SWIZZLE_R, GLenum_GL_TEXTURE_SWIZZLE_G, GLenum_GL_TEXTURE_SWIZZLE_B, GLenum_GL_TEXTURE_SWIZZLE_A:
		if t, err := subGetBoundTextureOrErrorInvalidEnum(ctx, nil, api.CmdNoID, nil, s, GetState(s), cmd.Thread(), nil, target); err == nil {
			_, curr := tc.getSwizzle(t, parameter)
			// The tex parameter was recently mutated, so set the original swizzle from current state.
			tc.origSwizzle[parameter][t] = curr
			// Combine the original and compat swizzles and write out the commands to set it.
			cb := CommandBuilder{Thread: cmd.Thread()}
			tc.writeCompatSwizzle(ctx, cb, t, parameter, out, id)
		}
	case GLenum_GL_TEXTURE_SWIZZLE_RGBA:
		log.E(ctx, "Unexpected GL_TEXTURE_SWIZZLE_RGBA")
	}
}

// decompressTexImage2D writes a glTexImage2D using the decompressed data for
// the given glCompressedTexImage2D.
func decompressTexImage2D(ctx context.Context, i api.CmdID, a *GlCompressedTexImage2D, s *api.GlobalState, out transform.Writer) error {
	ctx = log.Enter(ctx, "decompressTexImage2D")
	dID := i.Derived()
	c := GetContext(s, a.thread)
	cb := CommandBuilder{Thread: a.thread}
	data := a.Data
	if pb := c.Bound.PixelUnpackBuffer; pb != nil {
		base := a.Data.addr
		data = NewTexturePointer(pb.Data.Index(base, s.MemoryLayout))
		out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, 0))
		defer out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, pb.ID))
	} else {
		a.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
	}

	format, err := getCompressedImageFormat(a.Internalformat)
	if err != nil {
		return err
	}

	src := image.Info{
		Bytes:  image.NewID(data.Slice(0, uint64(a.ImageSize), s.MemoryLayout).ResourceID(ctx, s)),
		Width:  uint32(a.Width),
		Height: uint32(a.Height),
		Depth:  1,
		Format: format,
	}
	dst, err := src.Convert(ctx, image.RGBA_U8_NORM)
	if err != nil {
		return err
	}

	dstSize := a.Width * a.Height * 4

	tmp := s.AllocOrPanic(ctx, uint64(dstSize))
	out.MutateAndWrite(ctx, i, cb.GlTexImage2D(
		a.Target,
		a.Level,
		GLint(GLenum_GL_RGBA8),
		a.Width,
		a.Height,
		a.Border,
		GLenum_GL_RGBA,
		GLenum_GL_UNSIGNED_BYTE,
		tmp.Ptr(),
	).AddRead(tmp.Range(), dst.Bytes.ID()))
	tmp.Free()

	return nil
}

// decompressTexSubImage2D writes a glTexSubImage2D using the decompressed data for
// the given glCompressedTexSubImage2D.
func decompressTexSubImage2D(ctx context.Context, i api.CmdID, a *GlCompressedTexSubImage2D, s *api.GlobalState, out transform.Writer) error {
	ctx = log.Enter(ctx, "decompressTexSubImage2D")
	dID := i.Derived()
	c := GetContext(s, a.thread)
	cb := CommandBuilder{Thread: a.thread}
	data := a.Data
	if pb := c.Bound.PixelUnpackBuffer; pb != nil {
		base := a.Data.addr
		data = TexturePointer(pb.Data.Index(base, s.MemoryLayout))
		out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, 0))
		defer out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, pb.ID))
	} else {
		a.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
	}

	format, err := getCompressedImageFormat(a.Internalformat)
	if err != nil {
		return err
	}

	src := image.Info{
		Bytes:  image.NewID(data.Slice(0, uint64(a.ImageSize), s.MemoryLayout).ResourceID(ctx, s)),
		Width:  uint32(a.Width),
		Height: uint32(a.Height),
		Depth:  1,
		Format: format,
	}
	dst, err := src.Convert(ctx, image.RGBA_U8_NORM)
	if err != nil {
		return err
	}

	dstSize := a.Width * a.Height * 4

	tmp := s.AllocOrPanic(ctx, uint64(dstSize))
	out.MutateAndWrite(ctx, i, cb.GlTexSubImage2D(
		a.Target,
		a.Level,
		a.Xoffset,
		a.Yoffset,
		a.Width,
		a.Height,
		GLenum_GL_RGBA,
		GLenum_GL_UNSIGNED_BYTE,
		tmp.Ptr(),
	).AddRead(tmp.Range(), dst.Bytes.ID()))
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
