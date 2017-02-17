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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

// instances returns the path to the Instances field of the currently bound
// context, and the context at p.
func instances(ctx log.Context, p path.Node) (*path.Field, *Context, error) {
	if cmdPath := path.FindCommand(p); cmdPath != nil {
		stateObj, err := resolve.APIState(ctx, cmdPath.StateAfter())
		if err != nil {
			return nil, nil, err
		}
		state := stateObj.(*State)
		context, ok := state.Contexts[state.CurrentThread]
		if !ok {
			return nil, nil, nil
		}
		return cmdPath.StateAfter().
			Field("Contexts").
			MapIndex(uint64(state.CurrentThread)).
			Field("Instances"), context, nil
	}
	return nil, nil, nil
}

// Link returns the link to the attribute vertex array in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o AttributeLocation) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil {
		return nil, err
	}
	va, ok := c.Instances.VertexArrays[c.BoundVertexArray]
	if !ok || !va.VertexAttributeArrays.Contains(o) {
		return nil, nil
	}
	return i.
		Field("VertexArrays").
		MapIndex(uint64(c.BoundVertexArray)).
		Field("VertexAttributeArrays").
		MapIndex(uint64(o)), nil
}

// Link returns the link to the buffer object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o BufferId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Buffers.Contains(o) {
		return nil, err
	}
	return i.Field("Buffers").MapIndex(int64(o)), nil
}

// Link returns the link to the framebuffer object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o FramebufferId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Framebuffers.Contains(o) {
		return nil, err
	}
	return i.Field("Framebuffers").MapIndex(int64(o)), nil
}

// Link returns the link to the program in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o ProgramId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Programs.Contains(o) {
		return nil, err
	}
	return i.Field("Programs").MapIndex(int64(o)), nil
}

// Link returns the link to the query object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o QueryId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Queries.Contains(o) {
		return nil, err
	}
	return i.Field("Queries").MapIndex(int64(o)), nil
}

// Link returns the link to the renderbuffer object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o RenderbufferId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Renderbuffers.Contains(o) {
		return nil, err
	}
	return i.Field("Renderbuffers").MapIndex(int64(o)), nil
}

// Link returns the link to the shader object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o ShaderId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Shaders.Contains(o) {
		return nil, err
	}
	return i.Field("Shaders").MapIndex(int64(o)), nil
}

// Link returns the link to the texture object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o TextureId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.Textures.Contains(o) {
		return nil, err
	}
	return i.Field("Textures").MapIndex(int64(o)), nil
}

// Link returns the link to the uniform in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o UniformLocation) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil {
		return nil, err
	}

	atom, err := resolve.Command(ctx, path.FindCommand(p))
	if err != nil {
		return nil, err
	}

	var program ProgramId
	switch atom := atom.(type) {
	case *GlGetActiveUniform:
		program = atom.Program
	case *GlGetUniformLocation:
		program = atom.Program
	default:
		program = c.BoundProgram
	}

	prog, ok := c.Instances.Programs[program]
	if !ok || !prog.Uniforms.Contains(o) {
		return nil, nil
	}

	return i.
		Field("Programs").
		MapIndex(int64(program)).
		Field("Uniforms").
		MapIndex(int64(o)), nil
}

// Link returns the link to the vertex array in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o VertexArrayId) Link(ctx log.Context, p path.Node) (path.Node, error) {
	i, c, err := instances(ctx, p)
	if i == nil || !c.Instances.VertexArrays.Contains(o) {
		return nil, err
	}
	return i.Field("VertexArrays").MapIndex(int64(o)), nil
}
