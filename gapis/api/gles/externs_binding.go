// Copyright (C) 2018 Google Inc.
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

// #include "gapis/api/gles/ctypes.h"
//
// // Implemented in externs.c
// extern void externIndexLimits(gapil_context*, IndexLimits_args*, IndexLimits_res*);
import "C"

import (
	"unsafe"

	"github.com/google/gapid/gapil/executor"
)

func init() {
	executor.RegisterGoExtern("gles.GetAndroidNativeBufferExtra", externGetAndroidNativeBufferExtra)
	executor.RegisterGoExtern("gles.GetCompileShaderExtra", externGetCompileShaderExtra)
	executor.RegisterGoExtern("gles.GetEGLDynamicContextState", externGetEGLDynamicContextState)
	executor.RegisterGoExtern("gles.GetEGLImageData", externGetEGLImageData)
	executor.RegisterGoExtern("gles.GetEGLStaticContextState", externGetEGLStaticContextState)
	executor.RegisterGoExtern("gles.GetLinkProgramExtra", externGetLinkProgramExtra)
	executor.RegisterGoExtern("gles.GetValidateProgramExtra", externGetValidateProgramExtra)
	executor.RegisterGoExtern("gles.GetValidateProgramPipelineExtra", externGetValidateProgramPipelineExtra)
	executor.RegisterGoExtern("gles.ReadGPUTextureData", externReadGPUTextureData)
	executor.RegisterGoExtern("gles.addTag", externAddTag)
	executor.RegisterGoExtern("gles.mapMemory", externMapMemory)
	executor.RegisterGoExtern("gles.newMsg", externNewMsg)
	executor.RegisterGoExtern("gles.onGlError", externOnGlError)
	executor.RegisterGoExtern("gles.unmapMemory", externUnmapMemory)

	executor.RegisterCExtern("gles.IndexLimits", C.externIndexLimits)
	//executor.RegisterGoExtern("gles.IndexLimits", externIndexLimits)
}

func externsFromEnv(env *executor.Env) *externs {
	return &externs{
		ctx:   env.Context(),
		cmd:   env.Cmd(),
		cmdID: env.CmdID(),
		s:     env.State,
		b:     nil,
	}
}

func externGetAndroidNativeBufferExtra(env *executor.Env, args, out unsafe.Pointer) {
	// e := externsFromEnv(env)
	// a := (*C.GetAndroidNativeBufferExtra_args)(args)
	// o := (*C.GetAndroidNativeBufferExtra_res)(out)
	panic("externGetAndroidNativeBufferExtra not implemented")
}

func externGetCompileShaderExtra(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetCompileShaderExtra_args)(args)
	o := (*C.GetCompileShaderExtra_res)(out)
	*o = e.GetCompileShaderExtra(
		Contextʳ{a.ctx},
		Shaderʳ{a.p},
		BinaryExtraʳ{a.binary},
	).c
}

func externGetEGLDynamicContextState(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetEGLDynamicContextState_args)(args)
	o := (*C.GetEGLDynamicContextState_res)(out)

	*o = e.GetEGLDynamicContextState(EGLDisplay(a.display), EGLSurface(a.surface), EGLContext(a.context)).c
}

func externGetEGLImageData(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetEGLImageData_args)(args)

	e.GetEGLImageData(EGLImageKHR(a.img), GLsizei(a.width), GLsizei(a.height))
}

func externGetEGLStaticContextState(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetEGLStaticContextState_args)(args)
	o := (*C.GetEGLStaticContextState_res)(out)

	*o = e.GetEGLStaticContextState(
		EGLDisplay(a.display),
		EGLContext(a.context),
	).c
}

func externGetLinkProgramExtra(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetLinkProgramExtra_args)(args)
	o := (*C.GetLinkProgramExtra_res)(out)

	*o = e.GetLinkProgramExtra(
		Contextʳ{a.ctx},
		Programʳ{a.p},
		BinaryExtraʳ{a.binary},
	).c
}

func externGetValidateProgramExtra(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetValidateProgramExtra_args)(args)
	o := (*C.GetValidateProgramExtra_res)(out)

	*o = e.GetValidateProgramExtra(
		Contextʳ{a.ctx},
		Programʳ{a.p},
	).c
}

func externGetValidateProgramPipelineExtra(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.GetValidateProgramPipelineExtra_args)(args)
	o := (*C.GetValidateProgramPipelineExtra_res)(out)

	*o = e.GetValidateProgramPipelineExtra(
		Contextʳ{a.ctx},
		Pipelineʳ{a.p},
	).c
}

func externReadGPUTextureData(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.ReadGPUTextureData_args)(args)
	o := (*C.ReadGPUTextureData_res)(out)

	*o = *e.ReadGPUTextureData(
		Textureʳ{a.texture},
		GLint(a.level),
		GLint(a.layer),
	).c
}

func externAddTag(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.addTag_args)(args)
	e.addTag(uint32(a.msgID), env.Message(unsafe.Pointer(a.tag)))
}

func externMapMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.mapMemory_args)(args)

	e.mapMemory(U8ˢ{&a.slice})
}

func externNewMsg(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.newMsg_args)(args)
	o := (*C.newMsg_res)(out)
	*o = C.uint32_t(e.newMsg(Severity(a.s), env.Message(unsafe.Pointer(a.msg))))
}

func externOnGlError(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.onGlError_args)(args)

	e.onGlError(GLenum(a.v))
}

func externUnmapMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.unmapMemory_args)(args)

	e.unmapMemory(U8ˢ{&a.slice})
}

func externIndexLimits(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.IndexLimits_args)(args)
	o := (*C.IndexLimits_res)(out)

	*o = *e.IndexLimits(U8ˢ{&a.indices}, int32(a.sizeof_index)).c
}
