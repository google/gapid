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
	"fmt"
	"math"

	"github.com/gopherjs/gopherjs/js"
)

// Element is represents a DOM element type.
type Element struct {
	node
	text       Text
	Width      int                  `js:"width"`
	Height     int                  `js:"height"`
	OffsetLeft int                  `js:"offsetLeft"`
	OffsetTop  int                  `js:"offsetTop"`
	WidthV     interface{}          `js:"width"`
	HeightV    interface{}          `js:"height"`
	Style      *CSSStyleDeclaration `js:"style"`
}

func el(o *js.Object) *Element   { return &Element{node: node{o}} }
func newEl(name string) *Element { return Doc().createElement(name) }

// ElementProvider is the interface implemented by types that are represented
// by a DOM element.
type ElementProvider interface {
	// Element returns the DOM element that represents the type.
	Element() *Element
}

// Append adds v to the end of element's children.
// If v is a node then it is simply appended, otherwise the value will be
// converted to a string and appended as a text node.
func (e *Element) Append(v interface{}) Node {
	var n Node
	switch v := v.(type) {
	case Node:
		n = v
	case ElementProvider:
		n = v.Element()
	default:
		t := Doc().createTextNode(fmt.Sprintf("%v", v))
		if e.text.Object == nil {
			e.text = t
		}
		n = t
	}
	e.AppendNode(n)
	return n
}

// Text returns the child text node for the element, creating it if it
// doesn't already exist.
func (e *Element) Text() Text {
	if e.text.Object == nil {
		e.Append("") // assigns to e.text
	}
	return e.text
}

// VisibleRect returns the visible region of the element in the viewport.
func (e *Element) VisibleRect() *Rect {
	rect := e.Call("getBoundingClientRect")
	l := math.Max(-rect.Get("left").Float(), 0)
	t := math.Max(-rect.Get("top").Float(), 0)
	r := l + math.Min(rect.Get("right").Float(), float64(Win.InnerWidth))
	b := t + math.Min(rect.Get("bottom").Float(), float64(Win.InnerHeight))
	return NewRect(l, t, r, b)
}

// OnClick registers a click handler for the element.
func (e *Element) OnClick(cb func(MouseEvent)) Unsubscribe {
	e.Call("addEventListener", "click", cb)
	return func() { e.Call("removeEventListener", "click", cb) }
}

// OnMouseMove registers a mouse-move handler for the element.
func (e *Element) OnMouseMove(cb func(MouseEvent)) Unsubscribe {
	e.Call("addEventListener", "mousemove", cb)
	return func() { e.Call("removeEventListener", "mousemove", cb) }
}

// OnMouseDown registers a mouse-down handler for the element.
func (e *Element) OnMouseDown(cb func(MouseEvent)) Unsubscribe {
	e.Call("addEventListener", "mousedown", cb)
	return func() { e.Call("removeEventListener", "mousedown", cb) }
}

// OnMouseUp registers a mouse-up handler for the element.
func (e *Element) OnMouseUp(cb func(MouseEvent)) Unsubscribe {
	e.Call("addEventListener", "mouseup", cb)
	return func() { e.Call("removeEventListener", "mouseup", cb) }
}
