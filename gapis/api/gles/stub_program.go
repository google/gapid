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
	"sort"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

var (
	// We don't include tests directly in the gles package as it adds
	// signaficantly to the test build time.
	VisibleForTestingStubShaderSource = stubShaderSource
)

func buildStubProgram(ctx context.Context, thread uint64, e *api.CmdExtras, s *api.GlobalState, programID ProgramId) []api.Cmd {
	programInfo := FindProgramInfo(e)
	vss, fss, err := stubShaderSource(programInfo)
	if err != nil {
		log.E(ctx, "Unable to build stub shader: %v", err)
	}
	c := GetContext(s, thread)
	vertexShaderID := ShaderId(newUnusedID(ctx, 'S', func(x uint32) bool {
		ok := c.Objects.Buffers.Contains(BufferId(x))
		return ok
	}))
	fragmentShaderID := ShaderId(newUnusedID(ctx, 'S', func(x uint32) bool {
		ok := c.Objects.Buffers.Contains(BufferId(x))
		return ok || x == uint32(vertexShaderID)
	}))
	cb := CommandBuilder{Thread: thread}
	glLinkProgram := cb.GlLinkProgram(programID)
	glLinkProgram.Extras().Add(programInfo)
	return append(
		CompileProgram(ctx, s, cb, vertexShaderID, fragmentShaderID, programID, vss, fss),
		glLinkProgram,
	)
}

func stubShaderSource(pi *ProgramInfo) (vertexShaderSource, fragmentShaderSource string, err error) {
	vsDecls, fsDecls := []string{}, []string{}
	vsTickles, fsTickles := []string{}, []string{}
	if pi != nil {
		for _, u := range pi.ActiveUniforms.Range() {
			var decls, tickles *[]string
			if isSampler(u.Type) {
				decls, tickles = &fsDecls, &fsTickles
			} else {
				decls, tickles = &vsDecls, &vsTickles
			}

			ty, err := glslTypeFor(u.Type)
			if err != nil {
				return "", "", err
			}
			if u.ArraySize > 1 {
				name := strings.TrimRight(u.Name, "[0]")
				*decls = append(*decls, fmt.Sprintf("uniform %s %s[%d];\n", ty, name, u.ArraySize))
				for i := GLint(0); i < u.ArraySize; i++ {
					tkl, err := glslTickle(u.Type, fmt.Sprintf("%s[%d]", name, i))
					if err != nil {
						return "", "", err
					}
					*tickles = append(*tickles, fmt.Sprintf("no_strip += %s;\n    ", tkl))
				}
			} else {
				*decls = append(*decls, fmt.Sprintf("uniform %s %s;\n", ty, u.Name))
				tkl, err := glslTickle(u.Type, u.Name)
				if err != nil {
					return "", "", err
				}
				*tickles = append(*tickles, fmt.Sprintf("no_strip += %s;\n    ", tkl))
			}
		}
		// Deterministic output FTW!
		sort.Strings(vsDecls)
		sort.Strings(fsDecls)
		sort.Strings(vsTickles)
		sort.Strings(fsTickles)
	}
	return fmt.Sprintf(`#version 150

/////////////////////////////////////////////
// GAPID stub shader (no source available) //
/////////////////////////////////////////////

precision highp float;
%svoid main() {
    float no_strip = 0.0;
    %sgl_Position = vec4(no_strip * 0.000001, 0., 0., 1.);
}`, strings.Join(vsDecls, ""), strings.Join(vsTickles, "")),
		fmt.Sprintf(`#version 150

/////////////////////////////////////////////
// GAPID stub shader (no source available) //
/////////////////////////////////////////////

precision highp float;
%svoid main() {
    float no_strip = 0.0;
    %sgl_FragColor = vec4(1., no_strip * 0.000001, 1., 1.);
}`, strings.Join(fsDecls, ""), strings.Join(fsTickles, "")), nil
}

func glslTypeFor(ty GLenum) (string, error) {
	switch ty {
	case GLenum_GL_FLOAT:
		return "float", nil
	case GLenum_GL_FLOAT_VEC2:
		return "vec2", nil
	case GLenum_GL_FLOAT_VEC3:
		return "vec3", nil
	case GLenum_GL_FLOAT_VEC4:
		return "vec4", nil
	case GLenum_GL_INT:
		return "int", nil
	case GLenum_GL_INT_VEC2:
		return "ivec2", nil
	case GLenum_GL_INT_VEC3:
		return "ivec3", nil
	case GLenum_GL_INT_VEC4:
		return "ivec4", nil
	case GLenum_GL_UNSIGNED_INT:
		return "unsigned int", nil
	case GLenum_GL_UNSIGNED_INT_VEC2:
		return "uvec2", nil
	case GLenum_GL_UNSIGNED_INT_VEC3:
		return "uvec3", nil
	case GLenum_GL_UNSIGNED_INT_VEC4:
		return "uvec4", nil
	case GLenum_GL_BOOL:
		return "bool", nil
	case GLenum_GL_BOOL_VEC2:
		return "bvec2", nil
	case GLenum_GL_BOOL_VEC3:
		return "bvec3", nil
	case GLenum_GL_BOOL_VEC4:
		return "bvec4", nil
	case GLenum_GL_FLOAT_MAT2:
		return "mat2", nil
	case GLenum_GL_FLOAT_MAT3:
		return "mat3", nil
	case GLenum_GL_FLOAT_MAT4:
		return "mat4", nil
	case GLenum_GL_FLOAT_MAT2x3:
		return "mat2x3", nil
	case GLenum_GL_FLOAT_MAT2x4:
		return "mat2x4", nil
	case GLenum_GL_FLOAT_MAT3x2:
		return "mat3x2", nil
	case GLenum_GL_FLOAT_MAT3x4:
		return "mat3x4", nil
	case GLenum_GL_FLOAT_MAT4x2:
		return "mat4x2", nil
	case GLenum_GL_FLOAT_MAT4x3:
		return "mat4x3", nil
	case GLenum_GL_SAMPLER_2D:
		return "sampler2D", nil
	case GLenum_GL_SAMPLER_3D:
		return "sampler3D", nil
	case GLenum_GL_SAMPLER_CUBE:
		return "samplerCube", nil
	case GLenum_GL_SAMPLER_2D_SHADOW:
		return "sampler2DShadow", nil
	case GLenum_GL_SAMPLER_2D_ARRAY:
		return "sampler2DArray", nil
	case GLenum_GL_SAMPLER_2D_ARRAY_SHADOW:
		return "sampler2DArrayShadow", nil
	case GLenum_GL_SAMPLER_CUBE_SHADOW:
		return "samplerCubeShadow", nil
	case GLenum_GL_INT_SAMPLER_2D:
		return "isampler2D", nil
	case GLenum_GL_INT_SAMPLER_3D:
		return "isampler3D", nil
	case GLenum_GL_INT_SAMPLER_CUBE:
		return "isamplerCube", nil
	case GLenum_GL_INT_SAMPLER_2D_ARRAY:
		return "isampler2DArray", nil
	case GLenum_GL_UNSIGNED_INT_SAMPLER_2D:
		return "usampler2D", nil
	case GLenum_GL_UNSIGNED_INT_SAMPLER_3D:
		return "usampler3D", nil
	case GLenum_GL_UNSIGNED_INT_SAMPLER_CUBE:
		return "usamplerCube", nil
	case GLenum_GL_UNSIGNED_INT_SAMPLER_2D_ARRAY:
		return "usampler2DArray", nil
	default:
		return "", fmt.Errorf("Unknown uniform type %s", ty)
	}
}

func isSampler(ty GLenum) bool {
	switch ty {
	case GLenum_GL_SAMPLER_2D,
		GLenum_GL_SAMPLER_3D,
		GLenum_GL_SAMPLER_CUBE,
		GLenum_GL_SAMPLER_2D_SHADOW,
		GLenum_GL_SAMPLER_2D_ARRAY,
		GLenum_GL_SAMPLER_2D_ARRAY_SHADOW,
		GLenum_GL_SAMPLER_CUBE_SHADOW,
		GLenum_GL_INT_SAMPLER_2D,
		GLenum_GL_INT_SAMPLER_3D,
		GLenum_GL_INT_SAMPLER_CUBE,
		GLenum_GL_INT_SAMPLER_2D_ARRAY,
		GLenum_GL_UNSIGNED_INT_SAMPLER_2D,
		GLenum_GL_UNSIGNED_INT_SAMPLER_3D,
		GLenum_GL_UNSIGNED_INT_SAMPLER_CUBE,
		GLenum_GL_UNSIGNED_INT_SAMPLER_2D_ARRAY:
		return true
	default:
		return false
	}
}

func glslTickle(ty GLenum, name string) (string, error) {
	switch ty {
	case GLenum_GL_FLOAT:
		return name, nil
	case GLenum_GL_FLOAT_VEC2, GLenum_GL_FLOAT_VEC3, GLenum_GL_FLOAT_VEC4:
		return fmt.Sprintf("%s.x", name), nil
	case GLenum_GL_INT, GLenum_GL_UNSIGNED_INT:
		return fmt.Sprintf("float(%s)", name), nil
	case GLenum_GL_INT_VEC2, GLenum_GL_INT_VEC3, GLenum_GL_INT_VEC4,
		GLenum_GL_UNSIGNED_INT_VEC2, GLenum_GL_UNSIGNED_INT_VEC3, GLenum_GL_UNSIGNED_INT_VEC4:
		return fmt.Sprintf("float(%s.x)", name), nil
	case GLenum_GL_BOOL:
		return fmt.Sprintf("(%s?1.:0.)", name), nil
	case GLenum_GL_BOOL_VEC2, GLenum_GL_BOOL_VEC3, GLenum_GL_BOOL_VEC4:
		return fmt.Sprintf("(%s?1.:0.)", name), nil
	case GLenum_GL_FLOAT_MAT2, GLenum_GL_FLOAT_MAT3, GLenum_GL_FLOAT_MAT4,
		GLenum_GL_FLOAT_MAT2x3, GLenum_GL_FLOAT_MAT2x4,
		GLenum_GL_FLOAT_MAT3x2, GLenum_GL_FLOAT_MAT3x4,
		GLenum_GL_FLOAT_MAT4x2, GLenum_GL_FLOAT_MAT4x3:
		return fmt.Sprintf("%s[0].x", name), nil
	case GLenum_GL_SAMPLER_2D:
		return fmt.Sprintf("texture2D(%s, vec2(0.)).x", name), nil
	case GLenum_GL_SAMPLER_3D:
		return fmt.Sprintf("texture3D(%s, vec3(0.)).x", name), nil
	case GLenum_GL_SAMPLER_CUBE:
		return fmt.Sprintf("textureCube(%s, vec3(0.)).x", name), nil

	// The below types do not exist in GLSL 110.
	case GLenum_GL_SAMPLER_2D_SHADOW,
		GLenum_GL_SAMPLER_2D_ARRAY_SHADOW,
		GLenum_GL_SAMPLER_CUBE_SHADOW:
		return fmt.Sprintf("texture(%s, vec3(0.))", name), nil

	case GLenum_GL_SAMPLER_2D_ARRAY,
		GLenum_GL_INT_SAMPLER_2D,
		GLenum_GL_INT_SAMPLER_3D,
		GLenum_GL_INT_SAMPLER_CUBE,
		GLenum_GL_INT_SAMPLER_2D_ARRAY,
		GLenum_GL_UNSIGNED_INT_SAMPLER_2D,
		GLenum_GL_UNSIGNED_INT_SAMPLER_3D,
		GLenum_GL_UNSIGNED_INT_SAMPLER_CUBE,
		GLenum_GL_UNSIGNED_INT_SAMPLER_2D_ARRAY:
		return fmt.Sprintf("texture(%s, vec3(0.)).x", name), nil
	default:
		return "", fmt.Errorf("Unknown uniform type %s", ty)
	}
}
