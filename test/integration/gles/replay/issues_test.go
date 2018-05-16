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

package replay

import (
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/test/integration/gles/snippets"
)

// TestNoIssues tests the QueryIssues replay command returns with no issues for
// a clean capture.
func TestNoIssues(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, true)
	b.Add(
		b.CB.GlClearColor(0.0, 0.0, 1.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	c := buildAndMaybeExportCapture(ctx, b, "issues")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkIssues(ctx, c, d, []replay.Issue{}, nil)
	})
}

// TestWithIssues tests the QueryIssues replay command returns expected issues.
func TestWithIssues(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)

	missingProg := gles.ProgramId(1234)

	textureNames := []gles.TextureId{1}
	textureNamesR := b.Data(ctx, textureNames)

	squareIndicesR := b.Data(ctx, squareIndices)
	squareVerticesR := b.Data(ctx, squareVertices)

	someString := b.Data(ctx, "hello world")

	prog := b.CreateProgram(ctx, textureVSSource, textureFSSource)
	texLoc := b.AddUniformSampler(ctx, prog, "tex")
	posLoc := b.AddAttributeVec3(ctx, prog, "position")

	b.Add(
		b.CB.GlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
		b.CB.GlClearColor(0.0, 1.0, 0.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_2X_BIT_ATI), // INVALID
		b.CB.GlUseProgram(missingProg),                                                  // INVALID
		b.CB.GlLabelObjectEXT(gles.GLenum_GL_TEXTURE, 123, gles.GLsizei(someString.Range().Size), someString.Ptr()).AddRead(someString.Data()), // INVALID
		b.CB.GlGetError(0),
		b.CB.GlUseProgram(prog),
		b.CB.GlGenTextures(1, textureNamesR.Ptr()).AddWrite(textureNamesR.Data()),
		b.CB.GlActiveTexture(gles.GLenum_GL_TEXTURE0),
		b.CB.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		b.CB.GlUniform1i(texLoc, 0),
		b.CB.GlEnableVertexAttribArray(posLoc),
		b.CB.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, squareVerticesR.Ptr()),
		b.CB.GlDrawElements(gles.GLenum_GL_TRIANGLES, 6, gles.GLenum_GL_UNSIGNED_SHORT, squareIndicesR.Ptr()).
			AddRead(squareIndicesR.Data()).
			AddRead(squareVerticesR.Data()),
	)
	b.SwapBuffers()

	c := buildAndMaybeExportCapture(ctx, b, "with-issues")
	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkReport(ctx, c, d, b.Cmds, []string{
			"ErrorLevel@[18]: glClear(mask: GLbitfield(16385)): <ERR_INVALID_VALUE_CHECK_EQ [constraint: 16385, value: 16384]>",
			"ErrorLevel@[19]: glUseProgram(program: 1234): <ERR_INVALID_VALUE [value: 1234]>",
			"ErrorLevel@[20]: glLabelObjectEXT(type: GL_TEXTURE, object: 123, length: 12, label: 4216): <ERR_INVALID_OPERATION_OBJECT_DOES_NOT_EXIST [id: 123]>",
		}, nil)
	})
}
