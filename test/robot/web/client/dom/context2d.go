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

import (
	"math"

	"github.com/gopherjs/gopherjs/js"
)

const radToDeg = math.Pi / 180

// Context2D represents a func (c *Context2D)
type Context2D struct {
	*js.Object

	// LineWidth is the width of lines.
	LineWidth float64 `js:"lineWidth"`

	// LineCap specifies the style of endings on the end of lines.
	LineCap LineCap `js:"lineCap"`

	// LineJoin specifies the type of corners where two lines meet.
	LineJoin LineJoin `js:"lineJoin"`

	// MiterLimit is the miter limit ratio.
	MiterLimit float64 `js:"miterLimit"`

	// LineDashOffset specifies where to start a dash array on a line.
	LineDashOffset float64 `js:"lineDashOffset"`

	// Font is the current font to use for text drawing.
	Font Font `js:"font"`

	// TextAlign specifies text alignment when drawing text.
	TextAlign TextAlign `js:"textAlign"`

	// TextBaseline specifies the text baseline when drawing text.
	TextBaseline TextBaseline `js:"textBaseline"`

	// TextDirection specifies the direction when drawing text.
	TextDirection TextDirection `js:"direction"`

	// FillStyle is the color or style to use inside shapes.
	FillStyle interface{} `js:"fillStyle"`

	// StrokeStyle is the color or style to use for the lines around shapes.
	StrokeStyle interface{} `js:"strokeStyle"`

	// ShadowBlur controls the amount of blur.
	ShadowBlur float64 `js:"shadowBlur"`

	// ShadowColor is the color of the shadow.
	ShadowColor Color `js:"shadowColor"`

	// ShadowOffsetX is the horizonal offset of the shadow.
	ShadowOffsetX float64 `js:"shadowOffsetX"`

	// ShadowOffsetY is the vertical offset of the shadow.
	ShadowOffsetY float64 `js:"shadowOffsetY"`

	// GlobalAlpha is the alpha value that is applied to shapes and images before
	// they are composited onto the canvas.
	GlobalAlpha float64 `js:"globalAlpha"`

	// GlobalCompositeOperation sets how shapes and images are drawn onto the existing bitmap.
	GlobalCompositeOperation CompositeOperation `js:"globalCompositeOperation"`

	// ImageSmoothingEnabled is true if images should be smoothed when scaled.
	ImageSmoothingEnabled bool `js:"imageSmoothingEnabled"`
}

// LineCap is an enumerator of line ending styles.
type LineCap string

const (
	LineCapButt   LineCap = "butt"
	LineCapRound  LineCap = "round"
	LineCapSquare LineCap = "square"
)

// LineJoin is an enumerator of styles where two lines meet.
type LineJoin string

const (
	LineJoinRound LineJoin = "round"
	LineJoinBevel LineJoin = "bevel"
	LineJoinMiter LineJoin = "miter"
)

// TextAlign is an enumerator of text alignments.
type TextAlign string

const (
	TextAlignStart  TextAlign = "start"
	TextAlignEnd    TextAlign = "end"
	TextAlignLeft   TextAlign = "left"
	TextAlignRight  TextAlign = "right"
	TextAlignCenter TextAlign = "center"
)

// TextBaseline is an enumerator of text baselines.
type TextBaseline string

const (
	TextBaselineTop         TextBaseline = "top"
	TextBaselineHanging     TextBaseline = "hanging"
	TextBaselineMiddle      TextBaseline = "middle"
	TextBaselineAlphabetic  TextBaseline = "alphabetic"
	TextBaselineIdeographic TextBaseline = "ideographic"
	TextBaselineBottom      TextBaseline = "bottom"
)

// TextDirection is an enumerator of text flow directions.
type TextDirection string

const (
	TextDirectionLTR     TextDirection = "ltr"
	TextDirectionRTL     TextDirection = "rtl"
	TextDirectionInherit TextDirection = "inherit"
)

// Repetition is an enumerator of pattern repetition modes.
type Repetition string

const (
	Repeat   Repetition = "repeat"
	RepeatX  Repetition = "repeat-x"
	RepeatY  Repetition = "repeat-y"
	NoRepeat Repetition = "no-repeat"
)

// CompositeOperation is an enumerator of alpha blending composite functions.
type CompositeOperation string

type CanvasPattern struct{ *js.Object }
type CanvasGradient struct{ *js.Object }
type ImageData struct{ *js.Object }

// CanvasImageSource is any of the following types:
// HTMLImageElement, HTMLVideoElement, HTMLCanvasElement, or ImageBitmap.
type CanvasImageSource interface{}

// Point represents a two-dimensional position.
type Point struct{ X, Y float64 }

// NewPoint returns a new point.
func NewPoint(x, y float64) *Point { return &Point{x, y} }

// Clone returns a copy of p.
func (p *Point) Clone() *Point { return NewPoint(p.X, p.Y) }

// Add returns a new point with the value of p + q.
func (p *Point) Add(q *Point) *Point { return &Point{p.X + q.X, p.Y + q.Y} }

// Sub returns a new point with the value of p - q.
func (p *Point) Sub(q *Point) *Point { return &Point{p.X - q.X, p.Y - q.Y} }

// Neg returns a new point with negated value of p.
func (p *Point) Neg() *Point { return &Point{-p.X, -p.Y} }

// Rect represents a rectangle.
type Rect struct{ TL, BR *Point }

// NewRect returns a new rect.
func NewRect(x0, y0, x1, y1 float64) *Rect { return &Rect{NewPoint(x0, y0), NewPoint(x1, y1)} }

// NewRectWH returns a new rect from the top-left point and width and height values.
func NewRectWH(x, y, w, h float64) *Rect { return &Rect{NewPoint(x, y), NewPoint(x+w, y+h)} }

// Clone returns a copy of r.
func (r *Rect) Clone() *Rect { return &Rect{r.TL.Clone(), r.BR.Clone()} }

// W returns the width of the rectangle.
func (r *Rect) W() float64 { return r.BR.X - r.TL.X }

// H returns the height of the rectangle.
func (r *Rect) H() float64 { return r.BR.Y - r.TL.Y }

// Center returns the center point of the rectangle.
func (r *Rect) Center() *Point { return &Point{(r.TL.X + r.BR.X) / 2, (r.TL.Y + r.BR.Y) / 2} }

// Offset returns a new rect with the given offset.
func (r *Rect) Offset(p *Point) *Rect { return &Rect{r.TL.Add(p), r.BR.Add(p)} }

// Contains returns true if rect r contains point p.
func (r *Rect) Contains(p *Point) bool {
	return r.TL.X <= p.X && p.X < r.BR.X && r.TL.Y <= p.Y && p.Y < r.BR.Y
}

// Overlaps returns true if rect r overlaps rect s.
func (r *Rect) Overlaps(s *Rect) bool {
	return r.TL.X < s.BR.X && r.BR.X > s.TL.X && r.TL.Y < s.BR.Y && r.BR.Y > s.TL.Y
}

// Matrix represents a 3x3 matrix.
//   ╭         ╮
//   │ A  C  E │
//   │ B  D  F │
//   │ 0  0  1 │
//   ╰         ╯
type Matrix struct {
	A, C, E float64
	B, D, F float64
}

// Translation returns the translation part of the matrix as a point.
func (m *Matrix) Translation() *Point { return NewPoint(m.E, m.F) }

// ClearRect sets all pixels in the rectangle to transparent black, erasing any
// previously drawn content.
func (c *Context2D) ClearRect(r *Rect) { c.Call("clearRect", r.TL.X, r.TL.Y, r.W(), r.H()) }

// FillRect draws a filled rectangle.
func (c *Context2D) FillRect(r *Rect) { c.Call("fillRect", r.TL.X, r.TL.Y, r.W(), r.H()) }

// StrokeRect paints a rectangle using the current stroke style.
func (c *Context2D) StrokeRect(r *Rect) { c.Call("strokeRect", r.TL.X, r.TL.Y, r.W(), r.H()) }

// FillText draws with a fill the text at the given position.
func (c *Context2D) FillText(text string, p *Point) { c.Call("fillText", text, p.X, p.Y) }

// StrokeText draws with a stroke a the text at the given position.
func (c *Context2D) StrokeText(text string, p *Point) { c.Call("strokeText", text, p.X, p.Y) }

// MeasureText returns the measured width of the text.
func (c *Context2D) MeasureText(text string) float64 {
	return c.Call("measureText", text).Get("width").Float()
}

// LineDash returns the current line dash pattern as an array array containing
// an even number of non-negative numbers.
func (c *Context2D) LineDash() []int {
	arr := c.Call("getLineDash")
	out := make([]int, arr.Length())
	for i := range out {
		out[i] = arr.Index(i).Int()
	}
	return out
}

// SetLineDash sets the current line dash pattern.
func (c *Context2D) SetLineDash(pattern []float64) { c.Call("setLineDash", pattern) }

// CreateLinearGradient creates a linear gradient along the line given by the
// coordinates represented by the parameters.
func (c *Context2D) CreateLinearGradient(start, end *Point) CanvasGradient {
	return CanvasGradient{c.Call("createLinearGradient", start.X, start.Y, end.X, end.Y)}
}

// CreateRadialGradient creates a radial gradient given by the coordinates of
// the two circles and radii.
func (c *Context2D) CreateRadialGradient(centerA *Point, radiusA float64, centerB *Point, radiusB float64) CanvasGradient {
	return CanvasGradient{c.Call("createRadialGradient", centerA.X, centerA.Y, radiusA, centerB.X, centerB.Y, radiusB)}
}

// CreatePattern creates a pattern using the specified image and repetition.
func (c *Context2D) CreatePattern(s CanvasImageSource, r Repetition) CanvasPattern {
	return CanvasPattern{c.Call("createPattern", s, r)}
}

// BeginPath starts a new path by emptying the list of sub-paths.
// Call this method when you want to create a new path.
func (c *Context2D) BeginPath() { c.Call("beginPath") }

// ClosePath causes the point of the pen to move back to the start of the
// current sub-path. It tries to draw a straight line from the current point to
// the start. If the shape has already been closed or has only one point, this
// function does nothing.
func (c *Context2D) ClosePath() { c.Call("closePath") }

// MoveTo jumps the starting point of a new sub-path to p.
func (c *Context2D) MoveTo(p *Point) { c.Call("moveTo", p.X, p.Y) }

// LineTo connects the last point in the subpath to p with a straight line.
func (c *Context2D) LineTo(p *Point) { c.Call("lineTo", p.X, p.Y) }

// BézierCurveTo adds a cubic Bézier curve to the path.
// It requires three points. The first two points are control points and the
// third one is the end point. The starting point is the last point in the
// current path, which can be changed using moveTo() before creating the Bézier
// curve.
func (c *Context2D) BézierCurveTo(p, q, r *Point) {
	c.Call("bezierCurveTo", p.X, p.Y, q.X, q.Y, r.X, r.Y)
}

// QuadraticCurveTo adds a quadratic curve to the current path.
func (c *Context2D) QuadraticCurveTo(p, q, r *Point) { c.Call("quadraticCurveTo", p.X, p.Y, q.X, q.Y) }

// Arc adds an arc to the path which is centered at center with the specified
// radius starting at startAngle and ending at endAngle going in the given
// direction.
func (c *Context2D) Arc(center *Point, radius, startAngle, endAngle float64, clockwise bool) {
	c.Call("arc", center.X, center.Y, radius, startAngle*radToDeg, endAngle*radToDeg, clockwise)
}

// ArcTo adds an arc to the path with the given control points and radius,
// connected to the previous point by a straight line.
func (c *Context2D) ArcTo(p, q *Point, radius float64) {
	c.Call("arcTo", p.X, p.Y, q.X, q.Y, radius)
}

// Rect creates a path for a rectangle.
func (c *Context2D) Rect(r *Rect) {
	c.Call("rect", r.TL.X, r.TL.Y, r.W(), r.H())
}

// Fill fills the subpaths with the current fill style.
func (c *Context2D) Fill() { c.Call("fill") }

// Stroke strokes the subpaths with the current stroke style.
func (c *Context2D) Stroke() { c.Call("stroke") }

// DrawFocusIfNeeded will draw a focus ring around the current path if a given
// element is focused.
func (c *Context2D) DrawFocusIfNeeded() { c.Call("drawFocusIfNeeded") }

// ScrollPathIntoView scrolls the current path or a given path into the view.
func (c *Context2D) ScrollPathIntoView() { c.Call("scrollPathIntoView") }

// Clip creates a clipping path from the current sub-paths. Everything drawn
// after calling Clip appears inside the clipping path only.
func (c *Context2D) Clip() { c.Call("clip") }

// IsPointInPath returns true if the specified point is contained in the current
// path.
func (c *Context2D) IsPointInPath(p *Point) bool { return c.Call("isPointInPath", p.X, p.Y).Bool() }

// IsPointInStroke returns true if the specified point is inside the area
// contained by the stroking of a path.
func (c *Context2D) IsPointInStroke(p *Point) bool { return c.Call("isPointInStroke", p.X, p.Y).Bool() }

// Rotate applies a clockwise rotation in angle degrees to the current
// transformation matrix.
func (c *Context2D) Rotate(angle float64) { c.Call("rotate", angle*radToDeg) }

// Scale applies a scale to the current transformation matrix.
func (c *Context2D) Scale(x, y float64) { c.Call("scale", x, y) }

// Translate applies a translation to the current transformation matrix.
func (c *Context2D) Translate(p *Point) { c.Call("translate", p.X, p.Y) }

// Transform multiplies the current transformation matrix with m.
func (c *Context2D) Transform(m Matrix) { c.Call("transform", m.A, m.B, m.C, m.D, m.E, m.F) }

// SetTransform replaces the current transformation matrix with m.
func (c *Context2D) SetTransform(m Matrix) { c.Call("setTransform", m.A, m.B, m.C, m.D, m.E, m.F) }

// ResetTransform resets the current transformation matrix with identity.
func (c *Context2D) ResetTransform() { c.Call("resetTransform") }

// DrawImage draws the specified image.
func (c *Context2D) DrawImage(srcImage CanvasImageSource, srcRect, dstRect *Rect) {
	drp := Win.DevicePixelRatio
	c.Call("drawImage", srcImage,
		srcRect.TL.X*drp, srcRect.TL.Y*drp, srcRect.W()*drp, srcRect.H()*drp,
		dstRect.TL.X, dstRect.TL.Y, dstRect.W(), dstRect.H())
}

// CreateImageData creates a new, blank ImageData object with the specified
// dimensions. All of the pixels in the new object are transparent black.
func (c *Context2D) CreateImageData() ImageData { panic("TODO") }

// GetImageData returns an ImageData object representing the underlying pixel
// data for the area of the canvas denoted by rect.
func (c *Context2D) GetImageData(r *Rect) ImageData { panic("TODO") }

// PutImageData paints data from the given ImageData object onto the bitmap.
// If a dirty rectangle is provided, only the pixels from that rectangle are
// painted.
func (c *Context2D) PutImageData() { panic("TODO") }

// Save stores the current drawing style state using a stack so you can revert
// any change you make to it using Restore().
func (c *Context2D) Save() { c.Call("save") }

// Restore restores the drawing style state to the last element on the 'state
// stack' saved by Save().
func (c *Context2D) Restore() { c.Call("restore") }

// SaveRestore calls Save() and returns Restore.
// This is a convenience function so you can start a function with:
//
//  defer ctx.SaveRestore()()
func (c *Context2D) SaveRestore() func() { c.Save(); return c.Restore }
