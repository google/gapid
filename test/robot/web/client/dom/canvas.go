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

package dom

// Canvas represents an HTML <canvas> element.
type Canvas struct{ *Element }

// Context2D returns the Context2D for the context.
func (c *Canvas) Context2D() *Context2D { return &Context2D{Object: c.Call("getContext", "2d")} }

// Resize sets the size of the canvas.
func (c *Canvas) Resize(width, height int) {
	drp := Win.DevicePixelRatio
	style := c.Style
	c.Width, c.Height = int(float64(width)*drp), int(float64(height)*drp)
	style.Width, style.Height = width, height
	c.Context2D().Scale(drp, drp)
}

// NewCanvas returns a new Canvas element.
func NewCanvas(width, height int) *Canvas {
	canvas := Canvas{newEl("canvas")}
	canvas.Resize(width, height)
	return &canvas
}
