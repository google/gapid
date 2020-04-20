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

package api

import (
	"fmt"
	"strings"
)

// IsColor returns true if a is a color attachment.
func (a FramebufferAttachmentType) IsColor() bool {
	return a == FramebufferAttachmentType_OutputColor
}

// IsDepth returns true if a is a depth attachment.
func (a FramebufferAttachmentType) IsDepth() bool {
	return a == FramebufferAttachmentType_OutputDepth
}

// IsStencil returns true if a is a stencil attachment.
func (a FramebufferAttachmentType) IsStencil() bool {
	return false
}

func (a AspectType) Format(f fmt.State, c rune) {
	switch a {
	case AspectType_COLOR:
		fmt.Fprint(f, "Color")
	case AspectType_DEPTH:
		fmt.Fprint(f, "Depth")
	case AspectType_STENCIL:
		fmt.Fprint(f, "Stencil")
	default:
		fmt.Fprintf(f, "Unknown AspectType %d", int(a))
	}
}

func (t Pipeline_Type) Format(f fmt.State, c rune) {
	fmt.Fprint(f, strings.Title(strings.ToLower(t.String())))
}

func (x ShaderType) Extension() string {
	switch x {
	case ShaderType_Vertex:
		return "vert"
	case ShaderType_Geometry:
		return "geom"
	case ShaderType_TessControl:
		return "tessc"
	case ShaderType_TessEvaluation:
		return "tesse"
	case ShaderType_Fragment:
		return "frag"
	case ShaderType_Compute:
		return "comp"
	case ShaderType_Spirv:
		return "spvasm"
	case ShaderType_SpirvBinary:
		return "spv"
	default:
		return "unknown"
	}
}
