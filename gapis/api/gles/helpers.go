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

	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
)

// BuildProgram returns the commands to create a shader program with compiled
// vertex and fragment shaders. The returned program is not linked.
func BuildProgram(ctx context.Context, s *api.GlobalState, cb CommandBuilder,
	vertexShaderID, fragmentShaderID ShaderId, programID ProgramId,
	vertexShaderSource, fragmentShaderSource string) []api.Cmd {
	return append([]api.Cmd{cb.GlCreateProgram(programID)},
		CompileProgram(ctx, s, cb, vertexShaderID, fragmentShaderID, programID, vertexShaderSource, fragmentShaderSource)...,
	)
}

// CompileProgram returns the commands to compile and then attach vertex and
// fragment shaders to an existing shader program.
// The returned program is not linked.
func CompileProgram(ctx context.Context, s *api.GlobalState, cb CommandBuilder,
	vertexShaderID, fragmentShaderID ShaderId, programID ProgramId,
	vertexShaderSource, fragmentShaderSource string) []api.Cmd {

	tmpVertexSrcLen := s.AllocDataOrPanic(ctx, GLint(len(vertexShaderSource)))
	tmpVertexSrc := s.AllocDataOrPanic(ctx, vertexShaderSource)
	tmpPtrToVertexSrc := s.AllocDataOrPanic(ctx, tmpVertexSrc.Ptr())
	tmpFragmentSrcLen := s.AllocDataOrPanic(ctx, GLint(len(fragmentShaderSource)))
	tmpFragmentSrc := s.AllocDataOrPanic(ctx, fragmentShaderSource)
	tmpPtrToFragmentSrc := s.AllocDataOrPanic(ctx, tmpFragmentSrc.Ptr())

	cmds := []api.Cmd{
		cb.GlCreateShader(GLenum_GL_VERTEX_SHADER, vertexShaderID),
		cb.GlShaderSource(vertexShaderID, 1, tmpPtrToVertexSrc.Ptr(), tmpVertexSrcLen.Ptr()).
			AddRead(tmpPtrToVertexSrc.Data()).
			AddRead(tmpVertexSrcLen.Data()).
			AddRead(tmpVertexSrc.Data()),
		cb.GlCompileShader(vertexShaderID),
		cb.GlCreateShader(GLenum_GL_FRAGMENT_SHADER, fragmentShaderID),
		cb.GlShaderSource(fragmentShaderID, 1, tmpPtrToFragmentSrc.Ptr(), tmpFragmentSrcLen.Ptr()).
			AddRead(tmpPtrToFragmentSrc.Data()).
			AddRead(tmpFragmentSrcLen.Data()).
			AddRead(tmpFragmentSrc.Data()),
		cb.GlCompileShader(fragmentShaderID),
		cb.GlAttachShader(programID, vertexShaderID),
		cb.GlAttachShader(programID, fragmentShaderID),
		// TODO: We should be able to delete the shaders here.
	}

	tmpVertexSrc.Free()
	tmpFragmentSrc.Free()
	return cmds
}

// GetUniformLocation returns the command to get a uniform location by name.
func GetUniformLocation(ctx context.Context, s *api.GlobalState, cb CommandBuilder,
	programID ProgramId, name string, loc UniformLocation) api.Cmd {

	tmp := s.AllocDataOrPanic(ctx, name)
	cmd := cb.GlGetUniformLocation(programID, tmp.Ptr(), loc).
		AddRead(tmp.Data())

	tmp.Free()
	return cmd
}

// GetAttribLocation returns the command to get an attribute location by name.
func GetAttribLocation(ctx context.Context, s *api.GlobalState, cb CommandBuilder,
	programID ProgramId, name string, loc AttributeLocation) api.Cmd {

	tmp := s.AllocDataOrPanic(ctx, name)
	cmd := cb.GlGetAttribLocation(programID, tmp.Ptr(), GLint(loc)).
		AddRead(tmp.Data())

	tmp.Free()
	return cmd
}

// DefaultConstants30 returns a Constants structure filled with default
// values for a vaild OpenGL ES 3.0 context.
func DefaultConstants30(a arena.Arena) Constants {
	out := MakeConstants(a)
	out.SetSubpixelBits(4)
	out.SetMaxElementIndex(0xFFFFFF)
	out.SetMax3dTextureSize(256)
	out.SetMaxTextureSize(2048)
	out.SetMaxArrayTextureLayers(256)
	out.SetMaxTextureLodBias(2.0)
	out.SetMaxCubeMapTextureSize(2048)
	out.SetMaxRenderbufferSize(2048)
	out.SetMaxDrawBuffers(4)
	out.SetMaxFramebufferWidth(2048)
	out.SetMaxFramebufferHeight(2048)
	out.SetMaxFramebufferLayers(256)
	out.SetMaxFramebufferSamples(4)
	out.SetMaxColorAttachments(4)
	out.SetMinFragmentInterpolationOffset(-0.5)
	out.SetMaxFragmentInterpolationOffset(+0.5)
	out.SetFragmentInterpolationOffsetBits(4)
	out.SetMaxSampleMaskWords(1)
	out.SetMaxColorTextureSamples(1)
	out.SetMaxDepthTextureSamples(1)
	out.SetMaxIntegerSamples(1)
	out.SetMaxVertexAttribRelativeOffset(2047)
	out.SetMaxVertexAttribBindings(16)
	out.SetMaxVertexAttribStride(2048)
	out.SetMaxTextureBufferSize(65536)
	out.SetShaderCompiler(GLboolean_GL_TRUE)
	out.SetTextureBufferOffsetAlignment(256)
	out.SetMajorVersion(3)
	out.SetVersion("OpenGL ES 3.0")
	out.SetMaxVertexAttribs(16)
	out.SetMaxVertexUniformComponents(1024)
	out.SetMaxVertexUniformVectors(256)
	out.SetMaxVertexUniformBlocks(12)
	out.SetMaxVertexOutputComponents(64)
	out.SetMaxVertexTextureImageUnits(16)
	out.SetMaxTessGenLevel(64)
	out.SetMaxPatchVertices(32)
	out.SetMaxTessControlUniformComponents(1024)
	out.SetMaxTessControlTextureImageUnits(16)
	out.SetMaxTessControlOutputComponents(64)
	out.SetMaxTessPatchComponents(120)
	out.SetMaxTessControlTotalOutputComponents(4096)
	out.SetMaxTessControlInputComponents(64)
	out.SetMaxTessControlUniformBlocks(12)
	out.SetMaxTessEvaluationUniformComponents(1024)
	out.SetMaxTessEvaluationTextureImageUnits(16)
	out.SetMaxTessEvaluationOutputComponents(64)
	out.SetMaxTessEvaluationInputComponents(64)
	out.SetMaxTessEvaluationUniformBlocks(12)
	out.SetMaxGeometryUniformComponents(1024)
	out.SetMaxGeometryUniformBlocks(12)
	out.SetMaxGeometryInputComponents(64)
	out.SetMaxGeometryOutputComponents(64)
	out.SetMaxGeometryOutputVertices(256)
	out.SetMaxGeometryTotalOutputComponents(1024)
	out.SetMaxGeometryTextureImageUnits(16)
	out.SetMaxGeometryShaderInvocations(32)
	out.SetMaxFragmentUniformComponents(1024)
	out.SetMaxFragmentUniformVectors(256)
	out.SetMaxFragmentUniformBlocks(12)
	out.SetMaxFragmentInputComponents(60)
	out.SetMaxTextureImageUnits(16)
	out.SetMaxFragmentAtomicCounterBuffers(1)
	out.SetMaxFragmentAtomicCounters(8)
	out.SetMaxFragmentShaderStorageBlocks(4)
	out.SetMinProgramTexelOffset(-8)
	out.SetMaxProgramTexelOffset(7)
	out.SetMaxComputeWorkGroupInvocations(128)
	out.SetMaxComputeUniformBlocks(12)
	out.SetMaxComputeTextureImageUnits(16)
	out.SetMaxComputeSharedMemorySize(16384)
	out.SetMaxComputeUniformComponents(1024)
	out.SetMaxComputeAtomicCounterBuffers(1)
	out.SetMaxComputeAtomicCounters(8)
	out.SetMaxComputeShaderStorageBlocks(4)
	out.SetMaxUniformBufferBindings(72)
	out.SetMaxUniformBlockSize(16384)
	out.SetUniformBufferOffsetAlignment(256)
	out.SetMaxCombinedUniformBlocks(60)
	out.SetMaxVaryingComponents(60)
	out.SetMaxVaryingVectors(15)
	out.SetMaxCombinedTextureImageUnits(96)
	out.SetMaxCombinedShaderOutputResources(4)
	out.SetMaxUniformLocations(1024)
	out.SetMaxAtomicCounterBufferBindings(1)
	out.SetMaxAtomicCounterBufferSize(32)
	out.SetMaxCombinedAtomicCounterBuffers(1)
	out.SetMaxCombinedAtomicCounters(8)
	out.SetMaxImageUnits(4)
	out.SetMaxFragmentImageUniforms(4)
	out.SetMaxComputeImageUniforms(4)
	out.SetMaxCombinedImageUniforms(4)
	out.SetMaxShaderStorageBufferBindings(4)
	out.SetMaxShaderStorageBlockSize(0x8000000)
	out.SetMaxCombinedShaderStorageBlocks(4)
	out.SetShaderStorageBufferOffsetAlignment(256)
	out.SetMaxDebugMessageLength(1)
	out.SetMaxDebugLoggedMessages(1)
	out.SetMaxDebugGroupStackDepth(64)
	out.SetMaxLabelLength(256)
	out.SetMaxTransformFeedbackInterleavedComponents(64)
	out.SetMaxTransformFeedbackSeparateAttribs(4)
	out.SetMaxTransformFeedbackSeparateComponents(4)
	out.SetMaxTextureMaxAnisotropyExt(2.0)
	out.SetMaxViewsExt(2)
	out.SetCompressedTextureFormats(NewU32ːGLenumᵐ(a))
	out.SetProgramBinaryFormats(NewU32ːGLenumᵐ(a))
	out.SetShaderBinaryFormats(NewU32ːGLenumᵐ(a))
	out.SetExtensions(NewU32ːstringᵐ(a))
	return out
}

func NewStaticContextStateForTest(a arena.Arena) StaticContextState {
	constants := DefaultConstants30(a)
	constants.SetVersion("OpenGL ES 2.0")
	return NewStaticContextState(a,
		constants, // Constants
		"",        // ThreadName
	)
}

func NewDynamicContextStateForTest(a arena.Arena, width, height int, preserveBuffersOnSwap bool) DynamicContextState {
	return NewDynamicContextState(a,
		GLsizei(width),              //  BackbufferWidth
		GLsizei(height),             //  BackbufferHeight
		GLenum_GL_RGB565,            //   BackbufferColorFmt
		GLenum_GL_DEPTH_COMPONENT16, //   BackbufferDepthFmt
		GLenum_GL_STENCIL_INDEX8,    //   BackbufferStencilFmt
		preserveBuffersOnSwap,       //     PreserveBuffersOnSwap
		0,                           // RedSize
		0,                           // GreenSize
		0,                           // BlueSize
		0,                           // AlphaSize
		0,                           // DepthSize
		0,                           // StencilSize
	)
}
