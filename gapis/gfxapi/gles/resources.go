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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

// IsResource returns true if this instance should be considered as a resource.
func (t *Texture) IsResource() bool {
	return t.ID != 0
}

// ResourceHandle returns the UI identity for the resource.
func (t *Texture) ResourceHandle() string {
	return fmt.Sprintf("Texture<%d>", t.ID)
}

// ResourceLabel returns an optional debug label for the resource.
func (t *Texture) ResourceLabel() string {
	return t.Label
}

// Order returns an integer used to sort the resources for presentation.
func (t *Texture) Order() uint64 {
	return uint64(t.ID)
}

// ResourceType returns the type of this resource.
func (t *Texture) ResourceType(ctx context.Context) gfxapi.ResourceType {
	return gfxapi.ResourceType_TextureResource
}

// ResourceData returns the resource data given the current state.
func (t *Texture) ResourceData(ctx context.Context, s *gfxapi.State) (*gfxapi.ResourceData, error) {
	ctx = log.Enter(ctx, "Texture.ResourceData()")
	switch t.Kind {
	case GLenum_GL_TEXTURE_2D:
		levels := make([]*image.Info2D, len(t.Levels))
		for i, level := range t.Levels {
			img := level.Layers[0]
			if img.Data.count == 0 {
				// TODO: Make other results available
				return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
			}
			levels[i] = &image.Info2D{
				Format: getImageFormatOrPanic(img.DataFormat, img.DataType),
				Width:  uint32(img.Width),
				Height: uint32(img.Height),
				Data:   image.NewID(img.Data.ResourceID(ctx, s)),
			}
		}
		return gfxapi.NewResourceData(gfxapi.NewTexture(&gfxapi.Texture2D{Levels: levels})), nil

	case GLenum_GL_TEXTURE_CUBE_MAP:
		levels := make([]*gfxapi.CubemapLevel, len(t.Levels))
		for i, level := range t.Levels {
			levels[i] = &gfxapi.CubemapLevel{}
			for j, face := range level.Layers {
				if face.Data.count == 0 {
					// TODO: Make other results available
					return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
				}
				img := &image.Info2D{
					Format: getImageFormatOrPanic(face.DataFormat, face.DataType),
					Width:  uint32(face.Width),
					Height: uint32(face.Height),
					Data:   image.NewID(face.Data.ResourceID(ctx, s)),
				}
				switch GLenum(j) + GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X {
				case GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_X:
					levels[i].NegativeX = img
				case GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X:
					levels[i].PositiveX = img
				case GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Y:
					levels[i].NegativeY = img
				case GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Y:
					levels[i].PositiveY = img
				case GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Z:
					levels[i].NegativeZ = img
				case GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Z:
					levels[i].PositiveZ = img
				}
			}
		}
		return gfxapi.NewResourceData(gfxapi.NewTexture(&gfxapi.Cubemap{Levels: levels})), nil

	default:
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
	}
}

func (t *Texture) SetResourceData(ctx context.Context, at *path.Command,
	data *gfxapi.ResourceData, resources gfxapi.ResourceMap, edits gfxapi.ReplaceCallback) error {
	return fmt.Errorf("SetResourceData is not supported for Texture")
}

// IsResource returns true if this instance should be considered as a resource.
func (s *Shader) IsResource() bool {
	return s.ID != 0
}

// ResourceHandle returns the UI identity for the resource.
func (s *Shader) ResourceHandle() string {
	return fmt.Sprintf("Shader<%d>", s.ID)
}

// ResourceLabel returns an optional debug label for the resource.
func (s *Shader) ResourceLabel() string {
	return s.Label
}

// Order returns an integer used to sort the resources for presentation.
func (s *Shader) Order() uint64 {
	return uint64(s.ID)
}

// ResourceType returns the type of this resource.
func (s *Shader) ResourceType(ctx context.Context) gfxapi.ResourceType {
	return gfxapi.ResourceType_ShaderResource
}

// ResourceData returns the resource data given the current state.
func (s *Shader) ResourceData(ctx context.Context, t *gfxapi.State) (*gfxapi.ResourceData, error) {
	ctx = log.Enter(ctx, "Shader.ResourceData()")
	var ty gfxapi.ShaderType
	switch s.ShaderType {
	case GLenum_GL_VERTEX_SHADER:
		ty = gfxapi.ShaderType_Vertex
	case GLenum_GL_GEOMETRY_SHADER:
		ty = gfxapi.ShaderType_Geometry
	case GLenum_GL_TESS_CONTROL_SHADER:
		ty = gfxapi.ShaderType_TessControl
	case GLenum_GL_TESS_EVALUATION_SHADER:
		ty = gfxapi.ShaderType_TessEvaluation
	case GLenum_GL_FRAGMENT_SHADER:
		ty = gfxapi.ShaderType_Fragment
	case GLenum_GL_COMPUTE_SHADER:
		ty = gfxapi.ShaderType_Compute
	}

	return gfxapi.NewResourceData(&gfxapi.Shader{Type: ty, Source: s.Source}), nil
}

func (shader *Shader) SetResourceData(
	ctx context.Context,
	at *path.Command,
	data *gfxapi.ResourceData,
	resourceIDs gfxapi.ResourceMap,
	edits gfxapi.ReplaceCallback) error {

	atomIdx := at.Indices[0]
	if len(at.Indices) > 1 {
		return fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}

	// Dirty. TODO: Make separate type for getting info for a single resource.
	capturePath := at.Capture
	resources, err := resolve.Resources(ctx, capturePath)
	if err != nil {
		return err
	}
	resourceID := resourceIDs[shader]

	resource := resources.Find(shader.ResourceType(ctx), resourceID)
	if resource == nil {
		return fmt.Errorf("Couldn't find resource")
	}

	c, err := capture.ResolveFromPath(ctx, capturePath)
	if err != nil {
		return err
	}

	index := len(resource.Accesses) - 1
	for resource.Accesses[index].Indices[0] > atomIdx && index >= 0 { // TODO: Subcommands
		index--
	}
	for j := index; j >= 0; j-- {
		i := resource.Accesses[j].Indices[0] // TODO: Subcommands
		if a, ok := c.Atoms[i].(*GlShaderSource); ok {
			edits(uint64(i), a.Replace(ctx, c, data))
			return nil
		}
	}
	return fmt.Errorf("No atom to set data in")
}

func (a *GlShaderSource) Replace(ctx context.Context, c *capture.Capture, data *gfxapi.ResourceData) interface{} {
	state := c.NewState()
	shader := data.GetShader()
	source := shader.Source
	src, _ := atom.AllocData(ctx, state, source)
	srcLen, _ := atom.AllocData(ctx, state, GLint(len(source)))
	srcPtr, _ := atom.AllocData(ctx, state, src.Ptr())
	return NewGlShaderSource(a.Shader, 1, srcPtr.Ptr(), srcLen.Ptr()).
		AddRead(srcPtr.Data()).
		AddRead(srcLen.Data()).
		AddRead(src.Data())
}

// IsResource returns true if this instance should be considered as a resource.
func (p *Program) IsResource() bool {
	return p.ID != 0
}

// ResourceHandle returns the UI identity for the resource.
func (p *Program) ResourceHandle() string {
	return fmt.Sprintf("Program<%d>", p.ID)
}

// ResourceLabel returns an optional debug label for the resource.
func (p *Program) ResourceLabel() string {
	return p.Label
}

// Order returns an integer used to sort the resources for presentation.
func (p *Program) Order() uint64 {
	return uint64(p.ID)
}

// ResourceType returns the type of this resource.
func (p *Program) ResourceType(ctx context.Context) gfxapi.ResourceType {
	return gfxapi.ResourceType_ProgramResource
}

// ResourceData returns the resource data given the current state.
func (p *Program) ResourceData(ctx context.Context, s *gfxapi.State) (*gfxapi.ResourceData, error) {
	ctx = log.Enter(ctx, "Program.ResourceData()")

	shaders := make([]*gfxapi.Shader, 0, len(p.Shaders))
	for shaderType, shader := range p.Shaders {
		var ty gfxapi.ShaderType
		switch shaderType {
		case GLenum_GL_VERTEX_SHADER:
			ty = gfxapi.ShaderType_Vertex
		case GLenum_GL_GEOMETRY_SHADER:
			ty = gfxapi.ShaderType_Geometry
		case GLenum_GL_TESS_CONTROL_SHADER:
			ty = gfxapi.ShaderType_TessControl
		case GLenum_GL_TESS_EVALUATION_SHADER:
			ty = gfxapi.ShaderType_TessEvaluation
		case GLenum_GL_FRAGMENT_SHADER:
			ty = gfxapi.ShaderType_Fragment
		case GLenum_GL_COMPUTE_SHADER:
			ty = gfxapi.ShaderType_Compute
		}
		shaders = append(shaders, &gfxapi.Shader{
			Type:   ty,
			Source: shader.Source,
		})
	}

	uniforms := make([]*gfxapi.Uniform, 0, len(p.ActiveUniforms))
	for _, activeUniform := range p.ActiveUniforms {
		uniform := p.Uniforms[activeUniform.Location]

		var uniformFormat gfxapi.UniformFormat
		var uniformType gfxapi.UniformType

		switch activeUniform.Type {
		case GLenum_GL_FLOAT:
			uniformFormat = gfxapi.UniformFormat_Scalar
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_VEC2:
			uniformFormat = gfxapi.UniformFormat_Vec2
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_VEC3:
			uniformFormat = gfxapi.UniformFormat_Vec3
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_VEC4:
			uniformFormat = gfxapi.UniformFormat_Vec4
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_INT:
			uniformFormat = gfxapi.UniformFormat_Scalar
			uniformType = gfxapi.UniformType_Int32
		case GLenum_GL_INT_VEC2:
			uniformFormat = gfxapi.UniformFormat_Vec2
			uniformType = gfxapi.UniformType_Int32
		case GLenum_GL_INT_VEC3:
			uniformFormat = gfxapi.UniformFormat_Vec3
			uniformType = gfxapi.UniformType_Int32
		case GLenum_GL_INT_VEC4:
			uniformFormat = gfxapi.UniformFormat_Vec4
			uniformType = gfxapi.UniformType_Int32
		case GLenum_GL_UNSIGNED_INT:
			uniformFormat = gfxapi.UniformFormat_Scalar
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_VEC2:
			uniformFormat = gfxapi.UniformFormat_Vec2
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_VEC3:
			uniformFormat = gfxapi.UniformFormat_Vec3
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_VEC4:
			uniformFormat = gfxapi.UniformFormat_Vec4
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_BOOL:
			uniformFormat = gfxapi.UniformFormat_Scalar
			uniformType = gfxapi.UniformType_Bool
		case GLenum_GL_BOOL_VEC2:
			uniformFormat = gfxapi.UniformFormat_Vec2
			uniformType = gfxapi.UniformType_Bool
		case GLenum_GL_BOOL_VEC3:
			uniformFormat = gfxapi.UniformFormat_Vec3
			uniformType = gfxapi.UniformType_Bool
		case GLenum_GL_BOOL_VEC4:
			uniformFormat = gfxapi.UniformFormat_Vec4
			uniformType = gfxapi.UniformType_Bool
		case GLenum_GL_FLOAT_MAT2:
			uniformFormat = gfxapi.UniformFormat_Mat2
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT3:
			uniformFormat = gfxapi.UniformFormat_Mat3
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT4:
			uniformFormat = gfxapi.UniformFormat_Mat4
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT2x3:
			uniformFormat = gfxapi.UniformFormat_Mat2x3
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT2x4:
			uniformFormat = gfxapi.UniformFormat_Mat2x4
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT3x2:
			uniformFormat = gfxapi.UniformFormat_Mat3x2
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT3x4:
			uniformFormat = gfxapi.UniformFormat_Mat3x4
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT4x2:
			uniformFormat = gfxapi.UniformFormat_Mat4x2
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_FLOAT_MAT4x3:
			uniformFormat = gfxapi.UniformFormat_Mat4x3
			uniformType = gfxapi.UniformType_Float
		case GLenum_GL_SAMPLER_2D:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_SAMPLER_3D:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_SAMPLER_CUBE:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_SAMPLER_2D_SHADOW:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_SAMPLER_2D_ARRAY:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_SAMPLER_2D_ARRAY_SHADOW:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_SAMPLER_CUBE_SHADOW:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_2D:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_3D:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_CUBE:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_2D_ARRAY:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_2D:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_3D:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_CUBE:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_2D_ARRAY:
			uniformFormat = gfxapi.UniformFormat_Sampler
			uniformType = gfxapi.UniformType_Uint32
		default:
			uniformFormat = gfxapi.UniformFormat_Scalar
			uniformType = gfxapi.UniformType_Float
		}

		uniforms = append(uniforms, &gfxapi.Uniform{
			UniformLocation: uint32(activeUniform.Location),
			Name:            activeUniform.Name,
			Format:          uniformFormat,
			Type:            uniformType,
			Value:           box.NewValue(uniformValue(ctx, s, uniformType, uniform.Value)),
		})
	}

	return gfxapi.NewResourceData(&gfxapi.Program{Shaders: shaders, Uniforms: uniforms}), nil
}

func uniformValue(ctx context.Context, s *gfxapi.State, kind gfxapi.UniformType, data U8ˢ) interface{} {
	r := data.Reader(ctx, s)

	switch kind {
	case gfxapi.UniformType_Int32:
		a := make([]int32, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Int32()
		}
		return a
	case gfxapi.UniformType_Uint32:
		a := make([]uint32, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Uint32()
		}
		return a
	case gfxapi.UniformType_Bool:
		a := make([]bool, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Int32() != 0
		}
		return a
	case gfxapi.UniformType_Float:
		a := make([]float32, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Float32()
		}
		return a
	case gfxapi.UniformType_Double:
		a := make([]float64, data.count/8)
		for i := 0; i < len(a); i++ {
			a[i] = r.Float64()
		}
		return a
	default:
		panic(fmt.Errorf("Can't box uniform data type %v", kind))
	}
}

func (program *Program) SetResourceData(ctx context.Context, at *path.Command,
	data *gfxapi.ResourceData, resources gfxapi.ResourceMap, edits gfxapi.ReplaceCallback) error {
	return fmt.Errorf("SetResourceData is not supported for Program")
}
