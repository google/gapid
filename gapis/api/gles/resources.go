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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
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
func (t *Texture) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_TextureResource
}

// ResourceData returns the resource data given the current state.
func (t *Texture) ResourceData(ctx context.Context, s *api.GlobalState) (*api.ResourceData, error) {
	ctx = log.Enter(ctx, "Texture.ResourceData()")
	switch t.Kind {
	case GLenum_GL_TEXTURE_1D, GLenum_GL_TEXTURE_2D, GLenum_GL_TEXTURE_2D_MULTISAMPLE:
		levels := make([]*image.Info, t.Levels.Len())
		for i, level := range t.Levels.Range() {
			img, err := level.Layers.Get(0).ImageInfo(ctx, s)
			if err != nil {
				return nil, err
			}
			levels[i] = img
		}
		switch t.Kind {
		case GLenum_GL_TEXTURE_1D:
			return api.NewResourceData(api.NewTexture(&api.Texture1D{Levels: levels})), nil
		case GLenum_GL_TEXTURE_2D:
			return api.NewResourceData(api.NewTexture(&api.Texture2D{Levels: levels})), nil
		case GLenum_GL_TEXTURE_2D_MULTISAMPLE:
			return api.NewResourceData(api.NewTexture(&api.Texture2D{Levels: levels, Multisampled: true})), nil
		default:
			panic(fmt.Errorf("Unhandled texture kind %v", t.Kind))
		}

	case GLenum_GL_TEXTURE_EXTERNAL_OES:
		levels := make([]*image.Info, 1)
		if ei := t.EGLImage; ei != nil && ei.Image != nil {
			img, err := ei.Image.ImageInfo(ctx, s)
			if err != nil {
				return nil, err
			}
			levels[0] = img
		}
		return api.NewResourceData(api.NewTexture(&api.Texture2D{Levels: levels})), nil

	case GLenum_GL_TEXTURE_1D_ARRAY:
		numLayers := t.LayerCount()
		layers := make([]*api.Texture1D, numLayers)
		for layer := range layers {
			levels := make([]*image.Info, t.Levels.Len())
			for level := range levels {
				img, err := t.Levels.Get(GLint(level)).Layers.Get(GLint(layer)).ImageInfo(ctx, s)
				if err != nil {
					return nil, err
				}
				levels[level] = img
			}
			layers[layer] = &api.Texture1D{Levels: levels}
		}
		return api.NewResourceData(api.NewTexture(&api.Texture1DArray{Layers: layers})), nil

	case GLenum_GL_TEXTURE_2D_ARRAY, GLenum_GL_TEXTURE_2D_MULTISAMPLE_ARRAY:
		numLayers := t.LayerCount()
		layers := make([]*api.Texture2D, numLayers)
		for layer := range layers {
			levels := make([]*image.Info, t.Levels.Len())
			for level := range levels {
				img, err := t.Levels.Get(GLint(level)).Layers.Get(GLint(layer)).ImageInfo(ctx, s)
				if err != nil {
					return nil, err
				}
				levels[level] = img
			}
			layers[layer] = &api.Texture2D{Levels: levels}
		}
		switch t.Kind {
		case GLenum_GL_TEXTURE_2D_ARRAY:
			return api.NewResourceData(api.NewTexture(&api.Texture2DArray{Layers: layers})), nil
		case GLenum_GL_TEXTURE_2D_MULTISAMPLE_ARRAY:
			return api.NewResourceData(api.NewTexture(&api.Texture2DArray{Layers: layers, Multisampled: true})), nil
		default:
			panic(fmt.Errorf("Unhandled texture kind %v", t.Kind))
		}

	case GLenum_GL_TEXTURE_3D:
		levels := make([]*image.Info, t.Levels.Len())
		for i, level := range t.Levels.Range() {
			img := level.Layers.Get(0)
			l := &image.Info{
				Width:  uint32(img.Width),
				Height: uint32(img.Height),
				Depth:  uint32(level.Layers.Len()),
			}
			levels[i] = l
			if img.Data.count == 0 {
				continue
			}
			bytes := []byte{}
			for i, c := 0, level.Layers.Len(); i < c; i++ {
				l := level.Layers.Get(GLint(i))
				if l == nil {
					continue
				}
				pool, err := s.Memory.Get(l.Data.Pool())
				if err != nil {
					return nil, err
				}
				data := pool.Slice(l.Data.Range(s.MemoryLayout))
				buf := make([]byte, data.Size())
				if err := data.Get(ctx, 0, buf); err != nil {
					return nil, err
				}
				bytes = append(bytes, buf...)
			}
			id, err := database.Store(ctx, bytes)
			if err != nil {
				return nil, err
			}
			format, err := getImageFormat(img.DataFormat, img.DataType)
			if err != nil {
				return nil, err
			}
			l.Format = format
			l.Bytes = image.NewID(id)
		}
		return api.NewResourceData(api.NewTexture(&api.Texture3D{Levels: levels})), nil

	case GLenum_GL_TEXTURE_CUBE_MAP:
		levels := make([]*api.CubemapLevel, t.Levels.Len())
		for i, level := range t.Levels.Range() {
			levels[i] = &api.CubemapLevel{}
			for j, face := range level.Layers.Range() {
				img, err := face.ImageInfo(ctx, s)
				if err != nil {
					return nil, err
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
		return api.NewResourceData(api.NewTexture(&api.Cubemap{Levels: levels})), nil
	}
	return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
}

func (t *Texture) SetResourceData(ctx context.Context, at *path.Command,
	data *api.ResourceData, resources api.ResourceMap, edits api.ReplaceCallback) error {
	return fmt.Errorf("SetResourceData is not supported for Texture")
}

// ImageInfo returns the Image as a image.Info.
func (i *Image) ImageInfo(ctx context.Context, s *api.GlobalState) (*image.Info, error) {
	out := &image.Info{
		Width:  uint32(i.Width),
		Height: uint32(i.Height),
		Depth:  1,
	}
	if i.Data.count == 0 {
		return out, nil
	}
	dataFormat, dataType := i.getUnsizedFormatAndType()
	format, err := getImageFormat(dataFormat, dataType)
	if err != nil {
		return nil, err
	}
	out.Format = format
	out.Bytes = image.NewID(i.Data.ResourceID(ctx, s))
	return out, nil
}

// LayerCount returns the maximum number of layers across all levels.
func (t *Texture) LayerCount() int {
	max := 0
	for _, l := range t.Levels.Range() {
		if l.Layers.Len() > max {
			max = l.Layers.Len()
		}
	}
	return max
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
func (s *Shader) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_ShaderResource
}

// ResourceData returns the resource data given the current state.
func (s *Shader) ResourceData(ctx context.Context, t *api.GlobalState) (*api.ResourceData, error) {
	ctx = log.Enter(ctx, "Shader.ResourceData()")
	var ty api.ShaderType
	switch s.ShaderType {
	case GLenum_GL_VERTEX_SHADER:
		ty = api.ShaderType_Vertex
	case GLenum_GL_GEOMETRY_SHADER:
		ty = api.ShaderType_Geometry
	case GLenum_GL_TESS_CONTROL_SHADER:
		ty = api.ShaderType_TessControl
	case GLenum_GL_TESS_EVALUATION_SHADER:
		ty = api.ShaderType_TessEvaluation
	case GLenum_GL_FRAGMENT_SHADER:
		ty = api.ShaderType_Fragment
	case GLenum_GL_COMPUTE_SHADER:
		ty = api.ShaderType_Compute
	}

	return api.NewResourceData(&api.Shader{Type: ty, Source: s.Source}), nil
}

func (shader *Shader) SetResourceData(
	ctx context.Context,
	at *path.Command,
	data *api.ResourceData,
	resourceIDs api.ResourceMap,
	edits api.ReplaceCallback) error {

	atomIdx := at.Indices[0]
	if len(at.Indices) > 1 {
		return fmt.Errorf("Subcommands currently not supported for GLES resources") // TODO: Subcommands
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
		if a, ok := c.Commands[i].(*GlShaderSource); ok {
			edits(uint64(i), a.Replace(ctx, c, data))
			return nil
		}
	}
	return fmt.Errorf("No command to set data in")
}

func (a *GlShaderSource) Replace(ctx context.Context, c *capture.Capture, data *api.ResourceData) interface{} {
	state := c.NewState()
	shader := data.GetShader()
	source := shader.Source
	src := state.AllocDataOrPanic(ctx, source)
	srcLen := state.AllocDataOrPanic(ctx, GLint(len(source)))
	srcPtr := state.AllocDataOrPanic(ctx, src.Ptr())
	cb := CommandBuilder{Thread: a.thread}
	return cb.GlShaderSource(a.Shader, 1, srcPtr.Ptr(), srcLen.Ptr()).
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
func (p *Program) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_ProgramResource
}

// ResourceData returns the resource data given the current state.
func (p *Program) ResourceData(ctx context.Context, s *api.GlobalState) (*api.ResourceData, error) {
	ctx = log.Enter(ctx, "Program.ResourceData()")

	shaders := make([]*api.Shader, 0, p.Shaders.Len())
	for shaderType, shader := range p.Shaders.Range() {
		var ty api.ShaderType
		switch shaderType {
		case GLenum_GL_VERTEX_SHADER:
			ty = api.ShaderType_Vertex
		case GLenum_GL_GEOMETRY_SHADER:
			ty = api.ShaderType_Geometry
		case GLenum_GL_TESS_CONTROL_SHADER:
			ty = api.ShaderType_TessControl
		case GLenum_GL_TESS_EVALUATION_SHADER:
			ty = api.ShaderType_TessEvaluation
		case GLenum_GL_FRAGMENT_SHADER:
			ty = api.ShaderType_Fragment
		case GLenum_GL_COMPUTE_SHADER:
			ty = api.ShaderType_Compute
		}
		shaders = append(shaders, &api.Shader{
			Type:   ty,
			Source: shader.Source,
		})
	}

	uniforms := make([]*api.Uniform, 0, p.ActiveUniforms.Len())
	for _, activeUniform := range p.ActiveUniforms.Range() {
		uniform := p.Uniforms.Get(activeUniform.Location)

		var uniformFormat api.UniformFormat
		var uniformType api.UniformType

		switch activeUniform.Type {
		case GLenum_GL_FLOAT:
			uniformFormat = api.UniformFormat_Scalar
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_VEC2:
			uniformFormat = api.UniformFormat_Vec2
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_VEC3:
			uniformFormat = api.UniformFormat_Vec3
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_VEC4:
			uniformFormat = api.UniformFormat_Vec4
			uniformType = api.UniformType_Float
		case GLenum_GL_INT:
			uniformFormat = api.UniformFormat_Scalar
			uniformType = api.UniformType_Int32
		case GLenum_GL_INT_VEC2:
			uniformFormat = api.UniformFormat_Vec2
			uniformType = api.UniformType_Int32
		case GLenum_GL_INT_VEC3:
			uniformFormat = api.UniformFormat_Vec3
			uniformType = api.UniformType_Int32
		case GLenum_GL_INT_VEC4:
			uniformFormat = api.UniformFormat_Vec4
			uniformType = api.UniformType_Int32
		case GLenum_GL_UNSIGNED_INT:
			uniformFormat = api.UniformFormat_Scalar
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_VEC2:
			uniformFormat = api.UniformFormat_Vec2
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_VEC3:
			uniformFormat = api.UniformFormat_Vec3
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_VEC4:
			uniformFormat = api.UniformFormat_Vec4
			uniformType = api.UniformType_Uint32
		case GLenum_GL_BOOL:
			uniformFormat = api.UniformFormat_Scalar
			uniformType = api.UniformType_Bool
		case GLenum_GL_BOOL_VEC2:
			uniformFormat = api.UniformFormat_Vec2
			uniformType = api.UniformType_Bool
		case GLenum_GL_BOOL_VEC3:
			uniformFormat = api.UniformFormat_Vec3
			uniformType = api.UniformType_Bool
		case GLenum_GL_BOOL_VEC4:
			uniformFormat = api.UniformFormat_Vec4
			uniformType = api.UniformType_Bool
		case GLenum_GL_FLOAT_MAT2:
			uniformFormat = api.UniformFormat_Mat2
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT3:
			uniformFormat = api.UniformFormat_Mat3
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT4:
			uniformFormat = api.UniformFormat_Mat4
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT2x3:
			uniformFormat = api.UniformFormat_Mat2x3
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT2x4:
			uniformFormat = api.UniformFormat_Mat2x4
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT3x2:
			uniformFormat = api.UniformFormat_Mat3x2
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT3x4:
			uniformFormat = api.UniformFormat_Mat3x4
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT4x2:
			uniformFormat = api.UniformFormat_Mat4x2
			uniformType = api.UniformType_Float
		case GLenum_GL_FLOAT_MAT4x3:
			uniformFormat = api.UniformFormat_Mat4x3
			uniformType = api.UniformType_Float
		case GLenum_GL_SAMPLER_2D:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_SAMPLER_3D:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_SAMPLER_CUBE:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_SAMPLER_2D_SHADOW:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_SAMPLER_2D_ARRAY:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_SAMPLER_2D_ARRAY_SHADOW:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_SAMPLER_CUBE_SHADOW:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_2D:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_3D:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_CUBE:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_INT_SAMPLER_2D_ARRAY:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_2D:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_3D:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_CUBE:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		case GLenum_GL_UNSIGNED_INT_SAMPLER_2D_ARRAY:
			uniformFormat = api.UniformFormat_Sampler
			uniformType = api.UniformType_Uint32
		default:
			uniformFormat = api.UniformFormat_Scalar
			uniformType = api.UniformType_Float
		}

		uniforms = append(uniforms, &api.Uniform{
			UniformLocation: uint32(activeUniform.Location),
			Name:            activeUniform.Name,
			Format:          uniformFormat,
			Type:            uniformType,
			Value:           box.NewValue(uniformValue(ctx, s, uniformType, uniform.Value)),
		})
	}

	return api.NewResourceData(&api.Program{Shaders: shaders, Uniforms: uniforms}), nil
}

func uniformValue(ctx context.Context, s *api.GlobalState, kind api.UniformType, data U8ˢ) interface{} {
	r := data.Reader(ctx, s)

	switch kind {
	case api.UniformType_Int32:
		a := make([]int32, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Int32()
		}
		return a
	case api.UniformType_Uint32:
		a := make([]uint32, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Uint32()
		}
		return a
	case api.UniformType_Bool:
		a := make([]bool, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Int32() != 0
		}
		return a
	case api.UniformType_Float:
		a := make([]float32, data.count/4)
		for i := 0; i < len(a); i++ {
			a[i] = r.Float32()
		}
		return a
	case api.UniformType_Double:
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
	data *api.ResourceData, resources api.ResourceMap, edits api.ReplaceCallback) error {
	return fmt.Errorf("SetResourceData is not supported for Program")
}
