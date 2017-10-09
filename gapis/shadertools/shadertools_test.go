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

package shadertools_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/shadertools"
)

func TestConvertGlsl(t *testing.T) {
	for _, test := range []struct {
		desc     string
		opts     shadertools.Options
		src      string
		expected string
	}{
		{
			"Test shadertools propagates early_fragment_tests",
			shadertools.Options{
				ShaderType:        shadertools.TypeFragment,
				CheckAfterChanges: true,
			},
			`#version 310 es
layout(early_fragment_tests) in;
out highp vec4 color;
void main() { color = vec4(1., 1., 1., 1.); }`,
			`#version 330
#ifdef GL_ARB_shading_language_420pack
#extension GL_ARB_shading_language_420pack : require
#endif
#extension GL_ARB_shader_image_load_store : require
layout(early_fragment_tests) in;

out vec4 color;

void main()
{
    color = vec4(1.0);
}

`}, {
			"Test shadertools strips early_fragment_tests",
			shadertools.Options{
				ShaderType:         shadertools.TypeFragment,
				CheckAfterChanges:  true,
				StripOptimizations: true,
			},
			`#version 310 es
layout(early_fragment_tests) in;
out highp vec4 color;
void main() { color = vec4(1., 1., 1., 1.); }`,
			`#version 330
#ifdef GL_ARB_shading_language_420pack
#extension GL_ARB_shading_language_420pack : require
#endif

out vec4 color;

void main()
{
    color = vec4(1.0);
}

`},
	} {
		t.Run(test.desc, func(t *testing.T) {
			ctx := log.Testing(t)
			out, err := shadertools.ConvertGlsl(test.src, &test.opts)
			if assert.For(ctx, "err").ThatError(err).Succeeded() {
				assert.For(ctx, "src").ThatString(out.SourceCode).Equals(test.expected)
			}
		})
	}
}
