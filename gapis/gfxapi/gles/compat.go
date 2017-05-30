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
	"fmt"
	"reflect"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/preprocessor"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/shadertools"
)

type support int

const (
	unsupported support = iota
	supported
	required
)

var (
	// We don't include tests directly in the gles package as it adds
	// signaficantly to the test build time.
	VisibleForTestingCompat     = compat
	VisibleForTestingGlSlCompat = glslCompat
)

// If the default vertex array object (id 0) is not allowed on
// the target platform, we remap the uses to this array.
const DefaultVertexArrayId = VertexArrayId(0xFFFF0001)

type extensions map[string]struct{}

func listToExtensions(list []string) extensions {
	out := extensions{}
	for _, s := range list {
		out[s] = struct{}{}
	}
	return out
}

func translateExtensions(in U32ːstringᵐ) extensions {
	out := make(extensions, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}

func (e extensions) get(name string) support {
	if _, ok := e[name]; ok {
		return supported
	}
	return unsupported
}

func (s support) String() string {
	switch s {
	case unsupported:
		return "unsupported"
	case supported:
		return "supported"
	case required:
		return "required"
	default:
		return fmt.Sprintf("support<%d>", s)
	}
}

type features struct {
	vertexHalfFloatOES        support // support for GL_OES_vertex_half_float
	eglImageExternal          support // support for GL_OES_EGL_image_external
	textureMultisample        support // support for ARB_texture_multisample
	vertexArrayObjects        support // support for VBOs
	supportGenerateMipmapHint bool    // support for GL_GENERATE_MIPMAP_HINT
	compressedTextureFormats  map[GLenum]struct{}
	framebufferSrgb           support // support for GL_FRAMEBUFFER_SRGB
}

func getFeatures(ctx context.Context, version string, ext extensions) (features, *Version, error) {
	v, err := ParseVersion(version)
	if err != nil {
		return features{}, v, err
	}

	f := features{
		vertexHalfFloatOES:        ext.get("GL_OES_vertex_half_float"),
		eglImageExternal:          ext.get("GL_OES_EGL_image_external"),
		textureMultisample:        ext.get("ARB_texture_multisample"),
		compressedTextureFormats:  getSupportedCompressedTextureFormats(ext),
		supportGenerateMipmapHint: v.IsES,
	}

	// TODO: Properly check the specifications for these flags.
	switch {
	case v.AtLeastES(3, 0):
		f.vertexArrayObjects = supported
	case v.AtLeastGL(3, 0):
		f.vertexArrayObjects = required
	}

	if v.AtLeastES(3, 2) {
		f.textureMultisample = required
	}

	// GLES defaults FRAMEBUFFER_SRGB to enabled and only allows changing it via
	// an extension, while desktop GL defaults to disabled.
	if v.IsES {
		f.framebufferSrgb = ext.get("GL_EXT_sRGB_write_control")
	} else {
		f.framebufferSrgb = required
	}

	return f, v, nil
}

type scratchBuffer struct {
	size GLsizeiptr
	id   BufferId
}

func compat(ctx context.Context, device *device.Instance) (transform.Transformer, error) {
	ctx = log.Enter(ctx, "compat")

	glDev := device.Configuration.Drivers.OpenGL
	target, version, err := getFeatures(ctx, glDev.Version, listToExtensions(glDev.Extensions))
	if err != nil {
		return nil, fmt.Errorf(
			"Error '%v' when getting feature list for version: '%s', extensions: '%s'.",
			err, glDev.Version, glDev.Extensions)
	}

	contexts := map[*Context]features{}

	scratchBuffers := map[interface{}]scratchBuffer{}
	nextBufferID := BufferId(0xffff0000)
	newBuffer := func(i atom.ID, out transform.Writer) BufferId {
		s := out.State()
		id := nextBufferID
		tmp := atom.Must(atom.AllocData(ctx, s, id))
		out.MutateAndWrite(ctx, i.Derived(), NewGlGenBuffers(1, tmp.Ptr()).AddWrite(tmp.Data()))
		nextBufferID--
		return id
	}

	// Definitions of Vertex Arrays backed by client memory.
	// We postpone the write of the command until draw call.
	clientVAs := map[*VertexAttributeArray]*GlVertexAttribPointer{}

	textureCompat := &textureCompat{
		f:   target,
		v:   version,
		ctx: ctx,
		origSwizzle: map[GLenum]map[*Texture]GLenum{
			GLenum_GL_TEXTURE_SWIZZLE_R: {},
			GLenum_GL_TEXTURE_SWIZZLE_G: {},
			GLenum_GL_TEXTURE_SWIZZLE_B: {},
			GLenum_GL_TEXTURE_SWIZZLE_A: {},
		},
		compatSwizzle: map[*Texture]map[GLenum]GLenum{},
	}

	// Temporary buffer for each EGLImage which stores copy of its content
	eglImageData := map[GLeglImageOES]memory.Pointer{}

	// Upload last know EGL image content of bound texture (possibly from different context)
	// TODO: Share the data properly between contexts in replay.
	loadEglImageData := func(ctx context.Context, i atom.ID, a atom.Atom, target GLenum, c *Context, out transform.Writer) {
		s := out.State()
		if boundTexture, err := subGetBoundTextureOrErrorInvalidEnum(ctx, a, nil, s, GetState(s), nil, target); err != nil {
			log.W(ctx, "Can not get bound texture for: %v", a)
		} else {
			if !boundTexture.EGLImage.IsNullptr() {
				origUnpackAlignment := c.PixelStorage.UnpackAlignment
				img := boundTexture.Levels[0].Layers[0]
				data := eglImageData[boundTexture.EGLImage]
				out.MutateAndWrite(ctx, i, NewGlPixelStorei(GLenum_GL_UNPACK_ALIGNMENT, 1))
				out.MutateAndWrite(ctx, i, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
					NewGlTexImage2D(GLenum_GL_TEXTURE_2D, 0, GLint(img.DataFormat), img.Width, img.Height, 0, img.DataFormat, img.DataType, data).Call(ctx, s, b)
					return nil
				}))
				out.MutateAndWrite(ctx, i, NewGlPixelStorei(GLenum_GL_UNPACK_ALIGNMENT, origUnpackAlignment))
			}
		}
		return
	}

	// This allows us to avoid the EGLImage "resolve" if draw calls have been optimized away.
	isEglImageDirty := map[*Framebuffer]bool{}

	// If EGLImage is bound to current framebuffer, make a copy of its data.
	// TODO: Share the data properly between contexts in replay.
	resolveEglImageData := func(ctx context.Context, i atom.ID, a atom.Atom, c *Context, out transform.Writer) {
		fb := c.Objects.Framebuffers[c.BoundDrawFramebuffer]
		if !isEglImageDirty[fb] {
			return
		}
		isEglImageDirty[fb] = false
		// TODO: Depth and stencil
		for name, att := range fb.ColorAttachments {
			if att.Type == GLenum_GL_TEXTURE {
				tex := att.Texture
				if !tex.EGLImage.IsNullptr() {
					dID := i.Derived()
					t := newTweaker(ctx, out, dID)
					s := out.State()
					t.glBindFramebuffer_Read(c.BoundDrawFramebuffer)
					t.glReadBuffer(GLenum_GL_COLOR_ATTACHMENT0 + GLenum(name))
					t.setPixelStorage(PixelStorageState{UnpackAlignment: 1, PackAlignment: 1}, 0, 0)
					img := tex.Levels[0].Layers[0]
					data, ok := eglImageData[tex.EGLImage]
					if !ok {
						data = atom.Must(atom.Alloc(ctx, s, img.Data.count)).Ptr()
						eglImageData[tex.EGLImage] = data
					}
					out.MutateAndWrite(ctx, dID, NewGlReadPixels(0, 0, img.Width, img.Height, img.DataFormat, img.DataType, data))
					out.MutateAndWrite(ctx, dID, NewGlGetError(0))
					t.revert()
				}
			}
		}
	}

	// TODO: Implement full support for external images.
	convertTexTarget := func(t *GLenum) {
		if *t == GLenum_GL_TEXTURE_EXTERNAL_OES && target.eglImageExternal == unsupported {
			// Remap external textures to plain 2D textures - this matches GLSL compat.
			// TODO: This aliases GLenum_GL_TEXTURE_EXTERNAL_OES and GLenum_GL_TEXTURE_2D
			*t = GLenum_GL_TEXTURE_2D
		}
	}

	var t transform.Transformer
	t = transform.Transform("compat", func(ctx context.Context, i atom.ID, a atom.Atom, out transform.Writer) {
		dID := i.Derived()
		s := out.State()
		switch a := a.(type) {
		case *EglMakeCurrent: // TODO: Check for GLX, CGL, WGL...
			// The compatibility layer introduces calls to GL functions that are defined for desktop GL
			// and for GLES 3.0+. If the trace originated on a GLES 2.0 device, these new atoms' mutate
			// functions will fail the minRequiredVersion checks (which look at the version coming from
			// the original context from the trace).
			// TODO(dsrbecky): This might make some atoms valid for replay which were invalid on trace.
			scs := FindStaticContextState(a.Extras())
			if scs != nil && !version.IsES && scs.Constants.MajorVersion < 3 {
				clone := *a
				for i, e := range clone.extras.All() {
					if cs, ok := e.(*StaticContextState); ok {
						scs := *cs
						scs.Constants.MajorVersion = 3
						scs.Constants.MinorVersion = 0
						clone.extras[i] = &scs
					}
				}
				a = &clone
			}

			// Mutate to set the context, Version and Extensions strings.
			out.MutateAndWrite(ctx, i, a)

			c := GetContext(s)
			if c == nil || !c.Info.Initialized {
				return
			}
			if _, found := contexts[c]; found {
				return
			}

			source, _, err := getFeatures(ctx, c.Constants.Version, translateExtensions(c.Constants.Extensions))
			if err != nil {
				log.E(log.V{
					"version":    c.Constants.Version,
					"extensions": c.Constants.Extensions,
				}.Bind(ctx), "Error getting feature list: %v", err)
				return
			}

			contexts[c] = source

			if target.vertexArrayObjects == required &&
				source.vertexArrayObjects != required {
				// Replay device requires VAO, but capture did not enforce it.
				// Satisfy the target by creating and binding a single VAO
				// which we will use instead of the default VAO (id 0).
				tmp := atom.Must(atom.AllocData(ctx, s, VertexArrayId(DefaultVertexArrayId)))
				out.MutateAndWrite(ctx, dID, NewGlGenVertexArrays(1, tmp.Ptr()).AddWrite(tmp.Data()))
				out.MutateAndWrite(ctx, dID, NewGlBindVertexArray(DefaultVertexArrayId))
			}
			return
		}

		c := GetContext(s)
		if c == nil || !c.Info.Initialized {
			// The compatibility translations below assume that we have a valid context.
			out.MutateAndWrite(ctx, i, a)
			return
		}

		if a.AtomFlags().IsDrawCall() {
			fb := c.Objects.Framebuffers[c.BoundDrawFramebuffer]
			isEglImageDirty[fb] = true
		}

		switch a := a.(type) {
		case *GlBindBuffer:
			if a.Buffer != 0 && !c.SharedObjects.Buffers.Contains(a.Buffer) {
				// glGenBuffers() was not used to generate the buffer. Legal in GLES 2.
				tmp := atom.Must(atom.AllocData(ctx, s, a.Buffer))
				out.MutateAndWrite(ctx, dID, NewGlGenBuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
			}

		case *GlBindTexture:
			{
				a := *a
				if a.Texture != 0 && !c.SharedObjects.Textures.Contains(a.Texture) {
					// glGenTextures() was not used to generate the texture. Legal in GLES 2.
					tmp := atom.Must(atom.AllocData(ctx, s, VertexArrayId(a.Texture)))
					out.MutateAndWrite(ctx, dID, NewGlGenTextures(1, tmp.Ptr()).AddRead(tmp.Data()))
				}

				convertTexTarget(&a.Target)

				out.MutateAndWrite(ctx, i, &a)

				if !version.IsES {
					loadEglImageData(ctx, i, &a, a.Target, c, out)
				}
				return
			}

		case *GlBindVertexArray:
			if a.Array == VertexArrayId(0) {
				if target.vertexArrayObjects == required &&
					contexts[c].vertexArrayObjects != required {
					// NB: This leaks state change upstream.
					// In particular, when the tweaker saves and then restores vertex array binding,
					// it will restore it to DefaultVertexArrayId instead of 0.  It is harmless.
					out.MutateAndWrite(ctx, i, NewGlBindVertexArray(DefaultVertexArrayId))
					return
				}
			}

		case *GlBindVertexArrayOES:
			if a.Array == VertexArrayId(0) {
				if target.vertexArrayObjects == required &&
					contexts[c].vertexArrayObjects != required {
					out.MutateAndWrite(ctx, i, NewGlBindVertexArray(DefaultVertexArrayId))
					return
				}
			}

		case *GlBindBufferRange:
			misalignment := a.Offset % GLintptr(glDev.UniformBufferAlignment)
			if a.Target == GLenum_GL_UNIFORM_BUFFER && misalignment != 0 {
				// We have a glBindBufferRange() taking a uniform buffer with an
				// illegal offset alignment.
				// TODO: We don't handle the case where the buffer is kept bound
				// while the buffer is updated. It's an unlikely issue, but
				// something that may break us.
				if _, ok := c.SharedObjects.Buffers[a.Buffer]; !ok {
					return // Don't know what buffer this is referring to.
				}

				// We need a scratch buffer to copy the buffer data to a correct
				// alignment.
				key := struct {
					c      *Context
					Target GLenum
					Index  GLuint
				}{c, a.Target, a.Index}

				// Look for pre-existing buffer we can reuse.
				buffer, ok := scratchBuffers[key]
				if !ok {
					buffer.id = newBuffer(dID, out)
					scratchBuffers[key] = buffer
				}

				// Bind the scratch buffer to GL_COPY_WRITE_BUFFER
				origCopyWriteBuffer := c.BoundBuffers.CopyWriteBuffer
				out.MutateAndWrite(ctx, dID, NewGlBindBuffer(GLenum_GL_COPY_WRITE_BUFFER, buffer.id))

				if buffer.size < a.Size {
					// Resize the scratch buffer
					out.MutateAndWrite(ctx, dID, NewGlBufferData(GLenum_GL_COPY_WRITE_BUFFER, a.Size, memory.Nullptr, GLenum_GL_DYNAMIC_COPY))
					buffer.size = a.Size
					scratchBuffers[key] = buffer
				}

				// Copy out the misaligned data to the scratch buffer in the
				// GL_COPY_WRITE_BUFFER binding.
				out.MutateAndWrite(ctx, dID, NewGlBindBuffer(a.Target, a.Buffer))
				out.MutateAndWrite(ctx, dID, NewGlCopyBufferSubData(a.Target, GLenum_GL_COPY_WRITE_BUFFER, a.Offset, 0, a.Size))

				// We can now bind the range with correct alignment.
				out.MutateAndWrite(ctx, i, NewGlBindBufferRange(a.Target, a.Index, buffer.id, 0, a.Size))

				// Restore old GL_COPY_WRITE_BUFFER binding.
				out.MutateAndWrite(ctx, dID, NewGlBindBuffer(GLenum_GL_COPY_WRITE_BUFFER, origCopyWriteBuffer))

				return
			}

		case *GlDisableVertexAttribArray:
			vao := c.Objects.VertexArrays[c.BoundVertexArray]
			if vao.VertexAttributeArrays[a.Location].Enabled == GLboolean_GL_FALSE {
				// Ignore the call if it is redundant (i.e. it is already disabled).
				// Some applications iterate over all arrays and explicitly disable them.
				// This is a problem if the target supports fewer arrays than the capture.
				return
			}

		case *GlVertexAttrib4fv:
			if oldAttrib, ok := c.VertexAttributes[a.Location]; ok {
				oldValue := oldAttrib.Value.Read(ctx, a, s, nil /* builder */)
				a.Mutate(ctx, s, nil /* no builder, just mutate */)
				newAttrib := c.VertexAttributes[a.Location]
				newValue := newAttrib.Value.Read(ctx, a, s, nil /* builder */)
				if reflect.DeepEqual(oldValue, newValue) {
					// Ignore the call if it is redundant.
					// Some applications iterate over all arrays and explicitly initialize them.
					// This is a problem if the target supports fewer arrays than the capture.
					return
				}
			}
			out.MutateAndWrite(ctx, i, a)
			return

		case *GlGetVertexAttribIiv,
			*GlGetVertexAttribIuiv,
			*GlGetVertexAttribPointerv,
			*GlGetVertexAttribfv,
			*GlGetVertexAttribiv:
			// Some applications iterate over all arrays and query their state.
			// This may fail if the target supports fewer arrays than the capture.
			// As these should have no side-effects, just drop them.
			return

		case *GlShaderSource:
			// Apply the state mutation of the unmodified glShaderSource atom.
			// This is so we can grab the source string from the Shader object.
			if err := a.Mutate(ctx, s, nil /* no builder, just mutate */); err != nil {
				return
			}
			shader := c.SharedObjects.Shaders.Get(a.Shader)
			src := ""

			if config.UseGlslang {
				opts := shadertools.Option{
					IsFragmentShader: shader.Type == GLenum_GL_FRAGMENT_SHADER,
					IsVertexShader:   shader.Type == GLenum_GL_VERTEX_SHADER,
				}

				res := shadertools.ConvertGlsl(shader.Source, &opts)
				if !res.Ok {
					log.E(ctx, "Failed to translate GLSL:\n%s\nSource:%s\n", res.Message, shader.Source)
					return
				}
				src = res.SourceCode
			} else {
				lang := ast.LangVertexShader
				switch shader.Type {
				case GLenum_GL_VERTEX_SHADER:
				case GLenum_GL_FRAGMENT_SHADER:
					lang = ast.LangFragmentShader
				default:
					log.W(ctx, "Unknown shader type: %v", shader.Type)
				}

				exts := []preprocessor.Extension{}
				if target.textureMultisample == supported {
					// TODO: Check that this extension is actually used by the shader.
					exts = append(exts, preprocessor.Extension{
						Name: "GL_ARB_texture_multisample", Behaviour: "enable",
					})
				}

				src, err = glslCompat(ctx, shader.Source, lang, exts, device)
				if err != nil {
					log.E(ctx, "Error reformatting GLSL source for atom %d: %v", i, err)
				}
			}

			tmpSrc := atom.Must(atom.AllocData(ctx, s, src))
			tmpPtrToSrc := atom.Must(atom.AllocData(ctx, s, tmpSrc.Ptr()))
			a = NewGlShaderSource(a.Shader, 1, tmpPtrToSrc.Ptr(), memory.Nullptr).
				AddRead(tmpSrc.Data()).
				AddRead(tmpPtrToSrc.Data())
			out.MutateAndWrite(ctx, i, a)
			tmpPtrToSrc.Free()
			tmpSrc.Free()
			return

		// TODO: glVertexAttribIPointer
		case *GlVertexAttribPointer:
			if a.Type == GLenum_GL_HALF_FLOAT_OES && target.vertexHalfFloatOES == unsupported {
				// Convert GL_HALF_FLOAT_OES to GL_HALF_FLOAT_ARB.
				a = NewGlVertexAttribPointer(a.Location, a.Size, GLenum_GL_HALF_FLOAT_ARB, a.Normalized, a.Stride, memory.Pointer(a.Data))
			}
			vaa := c.Objects.VertexArrays[c.BoundVertexArray].VertexAttributeArrays[a.Location]
			if target.vertexArrayObjects == required && c.BoundBuffers.ArrayBuffer == 0 {
				// Client-pointers are not supported, we need to copy this data to a buffer.
				// However, we can't do this now as the observation only happens at the draw call.
				clientVAs[vaa] = a
			} else {
				delete(clientVAs, vaa)
				out.MutateAndWrite(ctx, i, a)
			}
			return

		case *GlDrawArrays:
			if target.vertexArrayObjects == required {
				if clientVAsBound(c, clientVAs) {
					first := uint32(a.FirstIndex)
					count := uint32(a.IndicesCount)
					t := newTweaker(ctx, out, dID)
					defer t.revert()
					moveClientVBsToVAs(ctx, t, clientVAs, first, count, i, a, s, c, out)
				}
			}

		case *GlDrawElements:
			if target.vertexArrayObjects == required {
				e := externs{ctx: ctx, a: a, s: s}
				t := newTweaker(ctx, out, dID)
				defer t.revert()

				ib := c.Objects.VertexArrays[c.BoundVertexArray].ElementArrayBuffer
				clientIB := ib == 0
				clientVB := clientVAsBound(c, clientVAs)
				if clientIB {
					// The indices for the glDrawElements call is in client memory.
					// We need to move this into a temporary buffer.

					// Generate a new element array buffer and bind it.
					id := t.glGenBuffer()
					t.GlBindBuffer_ElementArrayBuffer(id)

					// By moving the draw call's observations earlier, populate the element array buffer.
					size, base := DataTypeSize(a.IndicesType)*int(a.IndicesCount), memory.Pointer(a.Indices)
					glBufferData := NewGlBufferData(GLenum_GL_ELEMENT_ARRAY_BUFFER, GLsizeiptr(size), memory.Pointer(base), GLenum_GL_STATIC_DRAW)
					glBufferData.extras = a.extras
					out.MutateAndWrite(ctx, dID, glBufferData)

					if clientVB {
						// Some of the vertex arrays for the glDrawElements call is in
						// client memory and we need to move this into temporary buffer(s).
						// The indices are also in client memory, so we need to apply the
						// atom's reads now so that the indices can be read from the
						// application pool.
						a.Extras().Observations().ApplyReads(s.Memory[memory.ApplicationPool])
						indexSize := DataTypeSize(a.IndicesType)
						data := U8ᵖ(a.Indices).Slice(0, uint64(indexSize*int(a.IndicesCount)), s.MemoryLayout)
						limits := e.calcIndexLimits(data, indexSize)
						moveClientVBsToVAs(ctx, t, clientVAs, limits.First, limits.Count, i, a, s, c, out)
					}

					glDrawElements := *a
					glDrawElements.Indices.addr = 0
					out.MutateAndWrite(ctx, i, &glDrawElements)
					return

				} else if clientVB { // GL_ELEMENT_ARRAY_BUFFER is bound
					// Some of the vertex arrays for the glDrawElements call is in
					// client memory and we need to move this into temporary buffer(s).
					// The indices are server-side, so can just be read from the internal
					// pooled buffer.
					data := c.SharedObjects.Buffers[ib].Data
					indexSize := DataTypeSize(a.IndicesType)
					start := min(a.Indices.addr, data.count)                               // Clamp
					end := min(start+uint64(indexSize)*uint64(a.IndicesCount), data.count) // Clamp
					limits := e.calcIndexLimits(data.Slice(start, end, s.MemoryLayout), indexSize)
					moveClientVBsToVAs(ctx, t, clientVAs, limits.First, limits.Count, i, a, s, c, out)
				}
			}

		case *GlCompressedTexImage2D:
			if _, supported := target.compressedTextureFormats[a.Internalformat]; !supported {
				if err := decompressTexImage2D(ctx, i, a, s, out); err == nil {
					return
				}
				log.E(ctx, "Error decompressing texture: %v", err)
			}

		case *GlCompressedTexSubImage2D:
			if _, supported := target.compressedTextureFormats[a.Internalformat]; !supported {
				if err := decompressTexSubImage2D(ctx, i, a, s, out); err == nil {
					return
				}
				log.E(ctx, "Error decompressing texture: %v", err)
			}

		// TODO: glTexStorage functions are not guaranteed to be supported. Consider replacing with glTexImage calls.
		// TODO: Handle glTextureStorage family of functions - those use direct state access, not the bound texture.
		case *GlTexStorage1DEXT:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				if !version.IsES { // Strip suffix on desktop.
					a := NewGlTexStorage1D(a.Target, a.Levels, a.Internalformat, a.Width)
					out.MutateAndWrite(ctx, i, a)
					return
				}
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage2D:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage2DEXT:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				if !version.IsES { // Strip suffix on desktop.
					a := NewGlTexStorage2D(a.Target, a.Levels, a.Internalformat, a.Width, a.Height)
					out.MutateAndWrite(ctx, i, a)
					return
				}
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage2DMultisample:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage3D:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage3DEXT:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				if !version.IsES { // Strip suffix on desktop.
					a := NewGlTexStorage3D(a.Target, a.Levels, a.Internalformat, a.Width, a.Height, a.Depth)
					out.MutateAndWrite(ctx, i, a)
					return
				}
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage3DMultisample:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexStorage3DMultisampleOES:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				if !version.IsES { // Strip suffix on desktop.
					a := NewGlTexStorage3DMultisample(a.Target, a.Samples, a.Internalformat, a.Width, a.Height, a.Depth, a.Fixedsamplelocations)
					out.MutateAndWrite(ctx, i, a)
					return
				}
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexImage2D:
			{
				a := *a
				internalformat := GLenum(a.Internalformat)
				textureCompat.convertFormat(a.Target, &internalformat, &a.Format, &a.Type, out, i)
				a.Internalformat = GLint(internalformat)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexImage3D:
			{
				a := *a
				internalformat := GLenum(a.Internalformat)
				textureCompat.convertFormat(a.Target, &internalformat, &a.Format, &a.Type, out, i)
				a.Internalformat = GLint(internalformat)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexImage3DOES:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, &a.Format, &a.Type, out, i)
				if !version.IsES { // Strip suffix on desktop.
					extras := a.extras
					a := NewGlTexImage3D(a.Target, a.Level, GLint(a.Internalformat), a.Width, a.Height, a.Depth, a.Border, a.Format, a.Type, memory.Pointer(a.Pixels))
					a.extras = extras
					out.MutateAndWrite(ctx, i, a)
					return
				}
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexSubImage2D:
			{
				a := *a
				textureCompat.convertFormat(a.Target, nil, &a.Format, &a.Type, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexSubImage3D:
			{
				a := *a
				textureCompat.convertFormat(a.Target, nil, &a.Format, &a.Type, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlTexSubImage3DOES:
			{
				a := *a
				textureCompat.convertFormat(a.Target, nil, &a.Format, &a.Type, out, i)
				if !version.IsES { // Strip suffix on desktop.
					extras := a.extras
					a := NewGlTexSubImage3D(a.Target, a.Level, a.Xoffset, a.Yoffset, a.Zoffset, a.Width, a.Height, a.Depth, a.Format, a.Type, memory.Pointer(a.Pixels))
					a.extras = extras
					out.MutateAndWrite(ctx, i, a)
					return
				}
				out.MutateAndWrite(ctx, i, &a)
				return
			}
		case *GlCopyTexImage2D:
			{
				a := *a
				textureCompat.convertFormat(a.Target, &a.Internalformat, nil, nil, out, i)
				out.MutateAndWrite(ctx, i, &a)
				return
			}

		case *GlTexParameterIivOES:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameterIuivOES:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameterIiv:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameterIuiv:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameterf:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Parameter, out, i)
				return
			}
		case *GlTexParameterfv:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameteri:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Parameter, out, i)
				return
			}
		case *GlTexParameteriv:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameterIivEXT:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}
		case *GlTexParameterIuivEXT:
			{
				a := *a
				convertTexTarget(&a.Target)
				out.MutateAndWrite(ctx, i, &a)
				textureCompat.postTexParameter(a.Target, a.Pname, out, i)
				return
			}

		case *GlProgramBinary:
			if !canUsePrecompiledShader(c, glDev) {
				for _, a := range buildStubProgram(ctx, a.Extras(), s, a.Program) {
					t.Transform(ctx, i, a, out)
				}
				return
			}

		case *GlProgramBinaryOES:
			if !canUsePrecompiledShader(c, glDev) {
				for _, a := range buildStubProgram(ctx, a.Extras(), s, a.Program) {
					t.Transform(ctx, i, a, out)
				}
				return
			}

		case *GlHint:
			if a.Target == GLenum_GL_GENERATE_MIPMAP_HINT && !target.supportGenerateMipmapHint {
				return // Not supported in the core profile of OpenGL.
			}

		case *GlDebugMessageCallback,
			*GlDebugMessageControl,
			*GlDebugMessageCallbackKHR,
			*GlDebugMessageControlKHR:
			// Ignore - the callback function address is invalid in replay.
			return

		case *GlGetBooleani_v,
			*GlGetBooleanv,
			*GlGetFloatv,
			*GlGetInteger64i_v,
			*GlGetInteger64v,
			*GlGetIntegeri_v,
			*GlGetIntegerv,
			*GlGetInternalformativ,
			*GlGetString,
			*GlGetStringi:
			// The acceptable values of these get functions vary between GL versions.
			// As these should have no side-effects, just drop them.
			return

		case *GlGetActiveAttrib,
			*GlGetActiveUniform:
			// The number of active attributes and uniforms can vary between compilers
			// depending on their ability to eliminate dead code. In particular,
			// dead code in pixel shader can allow code removal in the vertex shader.
			// As these should have no side-effects, just drop them.
			return

		case *GlLabelObjectEXT,
			*GlGetObjectLabelEXT,
			*GlObjectLabel,
			*GlObjectLabelKHR,
			*GlGetObjectLabel,
			*GlObjectPtrLabel,
			*GlGetObjectPtrLabel,
			*GlGetObjectLabelKHR:
			// These methods require non-trivial remapping for replay.
			// As they do not affect rendering output, just drop them.
			return

		case *GlInsertEventMarkerEXT,
			*GlPushGroupMarkerEXT,
			*GlPopGroupMarkerEXT:
			// GL_EXT_debug_marker may not be supported on the replay device.
			// As they do not affect rendering output, just drop them.
			return

		case *GlGetProgramBinary,
			*GlGetProgramBinaryOES:
			// Program binaries are very driver specific. This command may fail on replay
			// because one of the arguments must be GL_PROGRAM_BINARY_LENGTH.
			// It has no side effects, so just drop it.
			return

		case *GlEnable:
			if a.Capability == GLenum_GL_FRAMEBUFFER_SRGB &&
				target.framebufferSrgb == required && contexts[c].framebufferSrgb != required &&
				c.BoundDrawFramebuffer == 0 {
				// Ignore enabling of FRAMEBUFFER_SRGB if the capture device did not
				// support an SRGB default framebuffer, but the replay device does. This
				// is only done if the current bound draw framebuffer is the default
				// framebuffer. The state is mutated so that when a non-default
				// framebuffer is bound later on, FRAMEBUFFER_SRGB will be enabled.
				// (see GlBindFramebuffer below)
				a.Mutate(ctx, s, nil /* no builder, just mutate */)
				return
			}

		case *GlDisable:
			// GL_QCOM_alpha_test adds back GL_ALPHA_TEST from GLES 1.0 as extension.
			// It seems that applications only disable it to make sure it is off, so
			// we can safely ignore it. We should not ignore glEnable for it though.
			if a.Capability == GLenum_GL_ALPHA_TEST_QCOM {
				return
			}

		case *GlGetGraphicsResetStatusEXT:
			// From extension GL_EXT_robustness
			// It may not be implemented by the replay driver.
			// It has no effect on rendering so just drop it.
			return

		case *GlInvalidateFramebuffer,
			*GlDiscardFramebufferEXT: // GL_EXT_discard_framebuffer
			// It may not be implemented by the replay driver.
			// It is only a hint so we can just drop it.
			// TODO: It has performance impact so we should not ignore it when profiling.
			return

		case *GlMapBufferOES:
			if !version.IsES { // Remove extension suffix on desktop.
				a := NewGlMapBuffer(a.Target, a.Access, memory.Pointer(a.Result))
				out.MutateAndWrite(ctx, i, a)
				return
			}

		case *GlMapBufferRangeEXT:
			if !version.IsES { // Remove extension suffix on desktop.
				a := NewGlMapBufferRange(a.Target, a.Offset, a.Length, a.Access, memory.Pointer(a.Result))
				out.MutateAndWrite(ctx, i, a)
				return
			}

		case *GlFlushMappedBufferRangeEXT:
			if !version.IsES { // Remove extension suffix on desktop.
				extras := a.extras
				a := NewGlFlushMappedBufferRange(a.Target, a.Offset, a.Length)
				a.extras = extras
				out.MutateAndWrite(ctx, i, a)
				return
			}

		case *GlUnmapBufferOES:
			if !version.IsES { // Remove extension suffix on desktop.
				extras := a.extras
				a := NewGlUnmapBuffer(a.Target, a.Result)
				a.extras = extras
				out.MutateAndWrite(ctx, i, a)
				return
			}

		case *GlBindFramebuffer:
			resolveEglImageData(ctx, i, a, c, out)

			if target.framebufferSrgb == required && contexts[c].framebufferSrgb != required &&
				c.FragmentOperations.FramebufferSrgb != 0 {
				// Replay device defaults FRAMEBUFFER_SRGB to disabled and allows
				// enabling it (desktop), while the capture device defaulted to enabled
				// and may or may not have allowed it to be changed (GLES). While at the
				// same time, we currently assume that the default frame buffer is not
				// SRGB capable. Thus, when SRGB is enabled in the state, and we're
				// binding the default framebuffer, SRGB needs to be disabled, and
				// specifically enabled when binding the non-default framebuffer.
				// (If it was explicetly disabled in the capture, no change is needed.)
				// TODO: Handle the use of the EGL KHR_gl_colorspace extension.
				if a.Target == GLenum_GL_FRAMEBUFFER || a.Target == GLenum_GL_DRAW_FRAMEBUFFER {
					origSrgb := c.FragmentOperations.FramebufferSrgb
					if a.Framebuffer == 0 {
						out.MutateAndWrite(ctx, dID, NewGlDisable(GLenum_GL_FRAMEBUFFER_SRGB))
					} else {
						out.MutateAndWrite(ctx, dID, NewGlEnable(GLenum_GL_FRAMEBUFFER_SRGB))
					}
					// Change the replay driver state, but keep our mutated state,
					// so we know what to do the next time we see glBindFramebuffer.
					// TODO: Handle SRGB better.
					c.FragmentOperations.FramebufferSrgb = origSrgb
				}
			}

		case *GlEGLImageTargetTexture2DOES:
			if !version.IsES {
				a.Mutate(ctx, s, nil /* no builder, just mutate */)
				loadEglImageData(ctx, i, a, a.Target, c, out)
				return
			}

		// EXT_multisampled_render_to_texture
		case *GlRenderbufferStorageMultisampleEXT:
			{
				// TODO: Support multi-sample rendering.
				a := NewGlRenderbufferStorage(a.Target, a.Internalformat, a.Width, a.Height)
				out.MutateAndWrite(ctx, i, a)
				return
			}

		// EXT_multisampled_render_to_texture
		case *GlFramebufferTexture2DMultisampleEXT:
			{
				// TODO: Support multi-sample rendering.
				a := NewGlFramebufferTexture2D(a.Target, a.Attachment, a.Textarget, a.Texture, a.Level)
				out.MutateAndWrite(ctx, i, a)
				return
			}

		case *GlLinkProgram:
			{
				out.MutateAndWrite(ctx, i, a)
				// Forcefully get all uniform locations, so that we can remap for applications that
				// just assume locations (in particular, apps tend to assume arrays are consecutive)
				// TODO: We should warn the developers that the consecutive layout is not guaranteed.
				prog := c.SharedObjects.Programs[a.Program]
				for _, uniformIndex := range prog.ActiveUniforms.KeysSorted() {
					uniform := prog.ActiveUniforms[uniformIndex]
					for i := 0; i < int(uniform.ArraySize); i++ {
						name := fmt.Sprintf("%v[%v]", strings.TrimSuffix(uniform.Name, "[0]"), i)
						loc := uniform.Location + UniformLocation(i) // TODO: Does not have to be consecutive
						out.MutateAndWrite(ctx, dID, NewGlGetUniformLocation(a.Program, name, loc))
					}
				}
				return
			}

		case *GlStartTilingQCOM, *GlEndTilingQCOM:
			if !version.IsES {
				// This extension is not applicable on desktop.
				return
			}

		default:
			if a.AtomFlags().IsDrawCall() && clientVAsBound(c, clientVAs) {
				log.W(ctx, "Draw call with client-pointers not handled by the compatability layer. Atom: %v", a)
			}
		}

		out.MutateAndWrite(ctx, i, a)
	})

	return t, nil
}

// canUsePrecompiledShader returns true if precompiled shaders / programs
// captured with the context c can be replayed on the device d.
func canUsePrecompiledShader(c *Context, d *device.OpenGLDriver) bool {
	return c.Constants.Vendor == d.Vendor && c.Constants.Version == d.Version
}

// clientVAsBound returns true if there are any vertex attribute arrays enabled
// with pointers to client-side memory.
func clientVAsBound(c *Context, clientVAs map[*VertexAttributeArray]*GlVertexAttribPointer) bool {
	va := c.Objects.VertexArrays[c.BoundVertexArray]
	for _, arr := range va.VertexAttributeArrays {
		if arr.Enabled == GLboolean_GL_TRUE {
			if _, ok := clientVAs[arr]; ok {
				return true
			}
		}
	}
	return false
}

// moveClientVBsToVAs is a compatability helper for transforming client-side
// vertex array data (which is not supported by glVertexAttribPointer in later
// versions of GL), into array-buffers.
func moveClientVBsToVAs(
	ctx context.Context,
	t *tweaker,
	clientVAs map[*VertexAttributeArray]*GlVertexAttribPointer,
	first, count uint32, // vertex indices
	i atom.ID,
	a atom.Atom,
	s *gfxapi.State,
	c *Context,
	out transform.Writer) {

	if count == 0 {
		return
	}

	rngs := interval.U64RangeList{}
	// Gather together all the client-buffers in use by the vertex-attribs.
	// Merge together all the memory intervals that these use.
	va := c.Objects.VertexArrays[c.BoundVertexArray]
	for _, arr := range va.VertexAttributeArrays {
		if arr.Enabled == GLboolean_GL_TRUE {
			vb := va.VertexBufferBindings[arr.Binding]
			if a, ok := clientVAs[arr]; ok {
				// TODO: We're currently ignoring the Offset and Stride fields of the VBB.
				// TODO: We're currently ignoring the RelativeOffset field of the VA.
				// TODO: Merge logic with ReadVertexArrays macro in vertex_arrays.api.
				if vb.Divisor != 0 {
					panic("Instanced draw calls not currently supported by the compatibility layer")
				}
				stride, size := int(a.Stride), DataTypeSize(a.Type)*int(a.Size)
				if stride == 0 {
					stride = size
				}
				rng := memory.Range{
					Base: a.Data.addr, // Always start from the 0'th vertex to simplify logic.
					Size: uint64(int(first+count-1)*stride + size),
				}
				interval.Merge(&rngs, rng.Span(), true)
			}
		}
	}

	// Create an array-buffer for each chunk of overlapping client-side buffers in
	// use. These are populated with data below.
	ids := make([]BufferId, len(rngs))
	for i := range rngs {
		ids[i] = t.glGenBuffer()
	}

	// Apply the memory observations that were made by the draw call now.
	// We need to do this as the glBufferData calls below will require the data.
	dID := i.Derived()
	out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
		a.Extras().Observations().ApplyReads(s.Memory[memory.ApplicationPool])
		return nil
	}))

	// Note: be careful of overwriting the observations made above, before the
	// calls to glBufferData below.

	// Fill the array-buffers with the observed memory data.
	for i, rng := range rngs {
		base := memory.BytePtr(rng.First, memory.ApplicationPool)
		size := GLsizeiptr(rng.Count)
		t.GlBindBuffer_ArrayBuffer(ids[i])
		out.MutateAndWrite(ctx, dID, NewGlBufferData(GLenum_GL_ARRAY_BUFFER, size, base, GLenum_GL_STATIC_DRAW))
	}

	// Redirect all the vertex attrib arrays to point to the array-buffer data.
	for _, l := range va.VertexAttributeArrays.KeysSorted() {
		arr := va.VertexAttributeArrays[l]
		if arr.Enabled == GLboolean_GL_TRUE {
			if a, ok := clientVAs[arr]; ok {
				a := *a // Copy
				i := interval.IndexOf(&rngs, a.Data.addr)
				t.GlBindBuffer_ArrayBuffer(ids[i])
				a.Data = VertexPointer{a.Data.addr - rngs[i].First, memory.ApplicationPool} // Offset
				out.MutateAndWrite(ctx, dID, &a)
			}
		}
	}
}
