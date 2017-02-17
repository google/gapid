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

import "github.com/gopherjs/gopherjs/js"

// Unsubscribe is the function returned by event subscription methods, such as
// OnClick. Calling Unsubscribe will prevent the event notification from firing
// again.
type Unsubscribe func()

type EventTarget interface{}
type DOMString string
type WindowProxy interface{}

// MouseEvent holds information about a mouse event.
type MouseEvent struct {
	*js.Object
	// Target represents the event target (the topmost target in the DOM tree).
	// Target EventTarget `js:"target"`
	// Type represents the type of event.
	Type DOMString `js:"type"`
	// Bubbles represents whether the event normally bubbles or not
	Bubbles bool `js:"bubbles"`
	// Cancelable represents whether the event is cancellable or not?
	Cancelable bool `js:"cancelable"`
	// View is the document.defaultView (window of the document)
	View WindowProxy `js:"view"`
	// Detail is a count of consecutive clicks that happened in a short amount
	// of time, incremented by one.
	Detail int `js:"detail"`
	// CurrentTarget is the node that had the event listener attached.
	CurrentTarget EventTarget `js:"currentTarget"`
	// RelatedTarget is used for mouseover, mouseout, mouseenter and mouseleave
	// events: the target of the complementary event (the mouseleave target in
	// the case of a mouseenter event). null otherwise.
	RelatedTarget EventTarget `js:"relatedTarget"`
	// ScreenX is the X coordinate of the mouse pointer in global (screen)
	// coordinates.
	ScreenX int `js:"screenX"`
	// ScreenY is the Y coordinate of the mouse pointer in global (screen)
	// coordinates.
	ScreenY int `js:"screenY"`
	// ClientX is the X coordinate of the mouse pointer in local (DOM content)
	// coordinates.
	ClientX int `js:"clientX"`
	// ClientY is the Y coordinate of the mouse pointer in local (DOM content)
	// coordinates.
	ClientY int `js:"clientY"`
	// PageX is the X coordinate of the mouse pointer in relative to the whole
	// document.
	PageX int `js:"pageX"`
	// PageY is the Y coordinate of the mouse pointer in relative to the whole
	// document.
	PageY int `js:"pageY"`
	// Button is the mouse button pressed when the event was fired.
	// For mice configured for left handed use in which the button actions are
	// reversed the values are instead read from right to left.
	Button MouseButton `js:"button"`
	// Buttons represents the buttons being pressed when the mouse event was
	// fired: Left button=1, Right button=2, Middle (wheel) button=4, 4th button
	// (typically, "Browser Back" button)=8, 5th button (typically,
	// "Browser Forward" button)=16. If two or more buttons are pressed, returns
	// the logical sum of the values. E.g., if Left button and Right button are
	// pressed, returns 3 (=1 | 2).
	Buttons uint `js:"buttons"`
	// CtrlKey is true if the control key was down when the event was fired. false otherwise.
	CtrlKey bool `js:"ctrlKey"`
	// ShiftKey is true if the shift key was down when the event was fired. false otherwise.
	ShiftKey bool `js:"shiftKey"`
	// AltKey is true if the alt key was down when the event was fired. false otherwise.
	AltKey bool `js:"altKey"`
	// MetaKey is true if the meta key was down when the event was fired. false otherwise.
	MetaKey bool `js:"metaKey"`
}

// MouseButton is an enumerator of mouse buttons
type MouseButton uint

const (
	LeftMouseButton = MouseButton(iota)
	MiddleMouseButton
	RightMouseButton
)

// HashChangeEvent holds information about a URL hash change.
type HashChangeEvent struct {
	*js.Object

	// NewURL is the new URL to which the window is navigating.
	NewURL DOMString `js:"newURL"`

	// OldURL is the old URL from which the window is navigating.
	OldURL DOMString `js:"oldURL"`
}

func (m *MouseEvent) PreventDefault() {
	m.Call("preventDefault")
}
