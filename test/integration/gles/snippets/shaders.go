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

package snippets

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
)

// CreateProgram appends the commands to create and link a new program.
func (b *Builder) CreateProgram(ctx context.Context,
	vertexShaderSource, fragmentShaderSource string) gles.ProgramId {

	a := b.state.Arena

	vs, fs, prog := b.newShaderID(), b.newShaderID(), b.newProgramID()

	b.Add(gles.BuildProgram(ctx, b.state, b.CB, vs, fs, prog,
		vertexShaderSource, fragmentShaderSource)...,
	)

	resources := gles.MakeActiveProgramResourcesʳ(a)

	lpe := gles.MakeLinkProgramExtra(a)
	lpe.SetLinkStatus(gles.GLboolean_GL_TRUE)
	lpe.SetActiveResources(resources)

	cmd := api.WithExtras(b.CB.GlLinkProgram(prog), lpe)
	b.programResources[prog] = resources
	b.Add(cmd)

	return prog
}

// AddUniformSampler adds a sampler 2D uniform to the given program.
func (b *Builder) AddUniformSampler(ctx context.Context, prog gles.ProgramId, name string) gles.UniformLocation {
	a := b.state.Arena

	resources := b.programResources[prog]
	index := resources.DefaultUniformBlock().Len()

	uniform := gles.MakeProgramResourceʳ(a)
	uniform.SetType(gles.GLenum_GL_SAMPLER_2D)
	uniform.SetName(name)
	uniform.SetArraySize(1)
	uniform.Locations().Add(0, gles.GLint(index))

	resources.DefaultUniformBlock().Add(gles.UniformIndex(index), uniform)

	location := gles.UniformLocation(index)
	b.Add(gles.GetUniformLocation(ctx, b.state, b.CB, prog, name, location))
	return location
}

// AddAttributeVec3 adds a vec3 attribute to the given program.
func (b *Builder) AddAttributeVec3(ctx context.Context, prog gles.ProgramId, name string) gles.AttributeLocation {
	a := b.state.Arena

	resources := b.programResources[prog]
	index := resources.ProgramInputs().Len()

	attribute := gles.MakeProgramResourceʳ(a)
	attribute.SetType(gles.GLenum_GL_FLOAT_VEC3)
	attribute.SetName(name)
	attribute.SetArraySize(1)
	attribute.Locations().Add(0, gles.GLint(index))

	resources.ProgramInputs().Add(uint32(index), attribute)

	location := gles.AttributeLocation(index)
	b.Add(gles.GetAttribLocation(ctx, b.state, b.CB, prog, name, location))
	return location
}
