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
	"fmt"

	"github.com/google/gapid/gapis/api/gles"
)

const (
	simpleVSSource = `
precision mediump float;
attribute vec3 position;
uniform float angle;

void main() {
	float c = cos(angle);
	float s = sin(angle);
		mat3 rotation = mat3(vec3(c, -s, 0.0), vec3(s, c, 0.0), vec3(0.0, 0.0, 1.0));
	gl_Position = vec4(rotation * position, 1.0);
}`

	textureVSSource = `
precision mediump float;
attribute vec3 position;
varying vec2 texcoord;
void main() {
	gl_Position = vec4(position, 1.0);
	texcoord = position.xy + vec2(0.5, 0.5);
}`

	textureFSSource = `
precision mediump float;
uniform sampler2D tex;
varying vec2 texcoord;
void main() {
	gl_FragColor = texture2D(tex, texcoord);
}`
)

// simpleFSSource returns a simple fragment shader that returns the given
// constant color.
func simpleFSSource(r, g, b gles.GLfloat) string {
	const template = `
precision mediump float;
void main() {
	gl_FragColor = vec4(%f, %f, %f, 1.0);
}`

	return fmt.Sprintf(template, r, g, b)
}
