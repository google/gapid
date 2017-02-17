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

package parser

import "github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"

// This variable contains stub declarations of symbols normally present in a shader, but which
// are not yet fully supported. This allows us to parse programs referencing these symbols, even
// though the later stages (semantic analysis will fail). The symbols commented with 1.0 are
// present only in the 1.0 version of the specification.
var builtinSymbols = [...]ast.Symbol{
	&ast.VariableSym{SymName: "gl_VertexID"},
	&ast.VariableSym{SymName: "gl_InstanceID"},
	&ast.VariableSym{SymName: "gl_Position"},
	&ast.VariableSym{SymName: "gl_PointSize"},

	&ast.VariableSym{SymName: "gl_FragCoord"},
	&ast.VariableSym{SymName: "gl_FrontFacing"},
	&ast.VariableSym{SymName: "gl_FragColor"}, // 1.0
	&ast.VariableSym{SymName: "gl_FragData"},  // 1.0
	&ast.VariableSym{SymName: "gl_FragDepth"},
	&ast.VariableSym{SymName: "gl_PointCoord"},

	&ast.VariableSym{SymName: "gl_MaxVertexAttribs"},
	&ast.VariableSym{SymName: "gl_MaxVertexUniformVectors"},
	&ast.VariableSym{SymName: "gl_MaxVaryingVectors"},          // 1.0
	&ast.VariableSym{SymName: "gl_MaxVertexTextureImageUnits"}, // 1.0
	&ast.VariableSym{SymName: "gl_MaxVertexOutputVectors"},
	&ast.VariableSym{SymName: "gl_MaxFragmentInputVectors"},
	&ast.VariableSym{SymName: "gl_MaxCombinedTextureImageUnits"},
	&ast.VariableSym{SymName: "gl_MaxTextureImageUnits"},
	&ast.VariableSym{SymName: "gl_MaxFragmentUniformVectors"},
	&ast.VariableSym{SymName: "gl_MaxDrawBuffers"},
	&ast.VariableSym{SymName: "gl_MinProgramTexelOffset"},
	&ast.VariableSym{SymName: "gl_MaxProgramTexelOffset"},

	&ast.StructSym{SymName: "gl_DepthRangeParameters"},
	&ast.VariableSym{SymName: "gl_DepthRange"},

	&ast.FunctionDecl{SymName: "textureSize"},
	&ast.FunctionDecl{SymName: "texture"},
	&ast.FunctionDecl{SymName: "textureProj"},
	&ast.FunctionDecl{SymName: "textureLod"},
	&ast.FunctionDecl{SymName: "textureOffset"},
	&ast.FunctionDecl{SymName: "texelFetch"},
	&ast.FunctionDecl{SymName: "texelFetchOffset"},
	&ast.FunctionDecl{SymName: "textureProjOffset"},
	&ast.FunctionDecl{SymName: "textureLodOffset"},
	&ast.FunctionDecl{SymName: "textureProjLod"},
	&ast.FunctionDecl{SymName: "textureProjLodOffset"},
	&ast.FunctionDecl{SymName: "textureGrad"},
	&ast.FunctionDecl{SymName: "textureGradOffset"},
	&ast.FunctionDecl{SymName: "textureProjGrad"},
	&ast.FunctionDecl{SymName: "textureProjGradOffset"},
	&ast.FunctionDecl{SymName: "texture2D"},        // 1.0
	&ast.FunctionDecl{SymName: "texture2DProj"},    // 1.0
	&ast.FunctionDecl{SymName: "texture2DLod"},     // 1.0
	&ast.FunctionDecl{SymName: "texture2DProjLod"}, // 1.0
	&ast.FunctionDecl{SymName: "textureCube"},      // 1.0
	&ast.FunctionDecl{SymName: "textureCubeLod"},   // 1.0

	&ast.FunctionDecl{SymName: "shadow2DEXT"},     // GL_SAMPLER_2D_SHADOW_EXT
	&ast.FunctionDecl{SymName: "shadow2DEXTProj"}, // GL_SAMPLER_2D_SHADOW_EXT

	&ast.FunctionDecl{SymName: "dFdx"},
	&ast.FunctionDecl{SymName: "dFdy"},
	&ast.FunctionDecl{SymName: "fwidth"},
}

// FindBuiltin searches and returns the builtins for the symbol with the
// specified name. If no builtin has the specified name then nil is returned.
func FindBuiltin(name string) ast.Symbol {
	for _, b := range builtinSymbols {
		if b.Name() == name {
			return b
		}
	}
	return nil
}
