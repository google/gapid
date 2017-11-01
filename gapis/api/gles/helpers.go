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

	"github.com/google/gapid/gapis/api"
)

// BuildProgram returns the atoms to create a shader program with compiled vertex
// and fragment shaders. The returned program is not linked.
func BuildProgram(ctx context.Context, s *api.GlobalState, cb CommandBuilder,
	vertexShaderID, fragmentShaderID ShaderId, programID ProgramId,
	vertexShaderSource, fragmentShaderSource string) []api.Cmd {
	return append([]api.Cmd{cb.GlCreateProgram(programID)},
		CompileProgram(ctx, s, cb, vertexShaderID, fragmentShaderID, programID, vertexShaderSource, fragmentShaderSource)...,
	)
}

// CompileProgram returns the atoms to compile and then attach vertex and
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

// DefaultConstants30 returns a Constants structure filled with default
// values for a vaild OpenGL ES 3.0 context.
func DefaultConstants30() Constants {
	return Constants{
		SubpixelBits:                              4,
		MaxElementIndex:                           0xFFFFFF,
		Max3dTextureSize:                          256,
		MaxTextureSize:                            2048,
		MaxArrayTextureLayers:                     256,
		MaxTextureLodBias:                         2.0,
		MaxCubeMapTextureSize:                     2048,
		MaxRenderbufferSize:                       2048,
		MaxDrawBuffers:                            4,
		MaxFramebufferWidth:                       2048,
		MaxFramebufferHeight:                      2048,
		MaxFramebufferLayers:                      256,
		MaxFramebufferSamples:                     4,
		MaxColorAttachments:                       4,
		MinFragmentInterpolationOffset:            -0.5,
		MaxFragmentInterpolationOffset:            +0.5,
		FragmentInterpolationOffsetBits:           4,
		MaxSampleMaskWords:                        1,
		MaxColorTextureSamples:                    1,
		MaxDepthTextureSamples:                    1,
		MaxIntegerSamples:                         1,
		MaxVertexAttribRelativeOffset:             2047,
		MaxVertexAttribBindings:                   16,
		MaxVertexAttribStride:                     2048,
		MaxTextureBufferSize:                      65536,
		ShaderCompiler:                            GLboolean_GL_TRUE,
		TextureBufferOffsetAlignment:              256,
		MajorVersion:                              3,
		Version:                                   "OpenGL ES 3.0",
		MaxVertexAttribs:                          16,
		MaxVertexUniformComponents:                1024,
		MaxVertexUniformVectors:                   256,
		MaxVertexUniformBlocks:                    12,
		MaxVertexOutputComponents:                 64,
		MaxVertexTextureImageUnits:                16,
		MaxTessGenLevel:                           64,
		MaxPatchVertices:                          32,
		MaxTessControlUniformComponents:           1024,
		MaxTessControlTextureImageUnits:           16,
		MaxTessControlOutputComponents:            64,
		MaxTessPatchComponents:                    120,
		MaxTessControlTotalOutputComponents:       4096,
		MaxTessControlInputComponents:             64,
		MaxTessControlUniformBlocks:               12,
		MaxTessEvaluationUniformComponents:        1024,
		MaxTessEvaluationTextureImageUnits:        16,
		MaxTessEvaluationOutputComponents:         64,
		MaxTessEvaluationInputComponents:          64,
		MaxTessEvaluationUniformBlocks:            12,
		MaxGeometryUniformComponents:              1024,
		MaxGeometryUniformBlocks:                  12,
		MaxGeometryInputComponents:                64,
		MaxGeometryOutputComponents:               64,
		MaxGeometryOutputVertices:                 256,
		MaxGeometryTotalOutputComponents:          1024,
		MaxGeometryTextureImageUnits:              16,
		MaxGeometryShaderInvocations:              32,
		MaxFragmentUniformComponents:              1024,
		MaxFragmentUniformVectors:                 256,
		MaxFragmentUniformBlocks:                  12,
		MaxFragmentInputComponents:                60,
		MaxTextureImageUnits:                      16,
		MaxFragmentAtomicCounterBuffers:           1,
		MaxFragmentAtomicCounters:                 8,
		MaxFragmentShaderStorageBlocks:            4,
		MinProgramTexelOffset:                     -8,
		MaxProgramTexelOffset:                     7,
		MaxComputeWorkGroupInvocations:            128,
		MaxComputeUniformBlocks:                   12,
		MaxComputeTextureImageUnits:               16,
		MaxComputeSharedMemorySize:                16384,
		MaxComputeUniformComponents:               1024,
		MaxComputeAtomicCounterBuffers:            1,
		MaxComputeAtomicCounters:                  8,
		MaxComputeShaderStorageBlocks:             4,
		MaxUniformBufferBindings:                  72,
		MaxUniformBlockSize:                       16384,
		UniformBufferOffsetAlignment:              256,
		MaxCombinedUniformBlocks:                  60,
		MaxVaryingComponents:                      60,
		MaxVaryingVectors:                         15,
		MaxCombinedTextureImageUnits:              96,
		MaxCombinedShaderOutputResources:          4,
		MaxUniformLocations:                       1024,
		MaxAtomicCounterBufferBindings:            1,
		MaxAtomicCounterBufferSize:                32,
		MaxCombinedAtomicCounterBuffers:           1,
		MaxCombinedAtomicCounters:                 8,
		MaxImageUnits:                             4,
		MaxFragmentImageUniforms:                  4,
		MaxComputeImageUniforms:                   4,
		MaxCombinedImageUniforms:                  4,
		MaxShaderStorageBufferBindings:            4,
		MaxShaderStorageBlockSize:                 0x8000000,
		MaxCombinedShaderStorageBlocks:            4,
		ShaderStorageBufferOffsetAlignment:        256,
		MaxDebugMessageLength:                     1,
		MaxDebugLoggedMessages:                    1,
		MaxDebugGroupStackDepth:                   64,
		MaxLabelLength:                            256,
		MaxTransformFeedbackInterleavedComponents: 64,
		MaxTransformFeedbackSeparateAttribs:       4,
		MaxTransformFeedbackSeparateComponents:    4,
		MaxTextureMaxAnisotropyExt:                2.0,
		CompressedTextureFormats:                  NewU32ːGLenumᵐ(),
		ProgramBinaryFormats:                      NewU32ːGLenumᵐ(),
		ShaderBinaryFormats:                       NewU32ːGLenumᵐ(),
		Extensions:                                NewU32ːstringᵐ(),
	}
}

func NewStaticContextState() *StaticContextState {
	constants := DefaultConstants30()
	constants.Version = "OpenGL ES 2.0"
	return &StaticContextState{
		Constants: constants,
	}
}

func NewDynamicContextState(width, height int, preserveBuffersOnSwap bool) *DynamicContextState {
	return &DynamicContextState{
		BackbufferWidth:       GLsizei(width),
		BackbufferHeight:      GLsizei(height),
		BackbufferColorFmt:    GLenum_GL_RGB565,
		BackbufferDepthFmt:    GLenum_GL_DEPTH_COMPONENT16,
		BackbufferStencilFmt:  GLenum_GL_STENCIL_INDEX8,
		ResetViewportScissor:  true,
		PreserveBuffersOnSwap: preserveBuffersOnSwap,
	}
}
