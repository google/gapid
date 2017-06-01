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

	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

// objects returns the path to the Objects field of the currently bound
// context, and the context at p.
func objects(ctx context.Context, p path.Node) (*path.Field, *Context, error) {
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
			MapIndex(state.CurrentThread).
			Field("Objects"), context, nil
	}
	return nil, nil, nil
}

// sharedObjects returns the path to the SharedObjects field of the currently bound
// context, and the context at p.
func sharedObjects(ctx context.Context, p path.Node) (*path.Field, *Context, error) {
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
			MapIndex(state.CurrentThread).
			Field("SharedObjects"), context, nil
	}
	return nil, nil, nil
}

// Link returns the link to the attribute vertex array in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o AttributeLocation) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := objects(ctx, p)
	if i == nil {
		return nil, err
	}
	va, ok := c.Objects.VertexArrays[c.BoundVertexArray]
	if !ok || !va.VertexAttributeArrays.Contains(o) {
		return nil, nil
	}
	return i.
		Field("VertexArrays").
		MapIndex(c.BoundVertexArray).
		Field("VertexAttributeArrays").
		MapIndex(o), nil
}

// Link returns the link to the buffer object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o BufferId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := sharedObjects(ctx, p)
	if i == nil || !c.SharedObjects.Buffers.Contains(o) {
		return nil, err
	}
	return i.Field("Buffers").MapIndex(o), nil
}

// Link returns the link to the framebuffer object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o FramebufferId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := objects(ctx, p)
	if i == nil || !c.Objects.Framebuffers.Contains(o) {
		return nil, err
	}
	return i.Field("Framebuffers").MapIndex(o), nil
}

// Link returns the link to the program in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o ProgramId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := sharedObjects(ctx, p)
	if i == nil || !c.SharedObjects.Programs.Contains(o) {
		return nil, err
	}
	return i.Field("Programs").MapIndex(o), nil
}

// Link returns the link to the query object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o QueryId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := objects(ctx, p)
	if i == nil || !c.Objects.Queries.Contains(o) {
		return nil, err
	}
	return i.Field("Queries").MapIndex(o), nil
}

// Link returns the link to the renderbuffer object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o RenderbufferId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := sharedObjects(ctx, p)
	if i == nil || !c.SharedObjects.Renderbuffers.Contains(o) {
		return nil, err
	}
	return i.Field("Renderbuffers").MapIndex(o), nil
}

// Link returns the link to the shader object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o ShaderId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := sharedObjects(ctx, p)
	if i == nil || !c.SharedObjects.Shaders.Contains(o) {
		return nil, err
	}
	return i.Field("Shaders").MapIndex(o), nil
}

// Link returns the link to the texture object in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o TextureId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := sharedObjects(ctx, p)
	if i == nil || !c.SharedObjects.Textures.Contains(o) {
		return nil, err
	}
	return i.Field("Textures").MapIndex(o), nil
}

// Link returns the link to the uniform in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o UniformLocation) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := sharedObjects(ctx, p)
	if i == nil {
		return nil, err
	}

	atom, err := resolve.Atom(ctx, path.FindCommand(p))
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

	prog, ok := c.SharedObjects.Programs[program]
	if !ok || !prog.Uniforms.Contains(o) {
		return nil, nil
	}

	return i.
		Field("Programs").
		MapIndex(program).
		Field("Uniforms").
		MapIndex(o), nil
}

// Link returns the link to the vertex array in the state block.
// If nil, nil is returned then the path cannot be followed.
func (o VertexArrayId) Link(ctx context.Context, p path.Node) (path.Node, error) {
	i, c, err := objects(ctx, p)
	if i == nil || !c.Objects.VertexArrays.Contains(o) {
		return nil, err
	}
	return i.Field("VertexArrays").MapIndex(o), nil
}
