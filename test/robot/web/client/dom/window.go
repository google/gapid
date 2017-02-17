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
	"time"

	"github.com/gopherjs/gopherjs/js"
)

// Window represents a window that holds a document.
type Window struct {
	*js.Object

	onload           *eventListeners
	onscroll         *eventListeners
	Location         Location `js:"location"`
	DevicePixelRatio float64  `js:"devicePixelRatio"`
	InnerWidth       int      `js:"innerWidth"`
	InnerHeight      int      `js:"innerHeight"`
}

type eventListeners struct {
	listeners map[int]func()
	nextID    int
}

func (l *eventListeners) add(f func()) Unsubscribe {
	if l.listeners == nil {
		l.listeners = map[int]func(){}
	}
	id := l.nextID
	l.listeners[id] = f
	l.nextID++
	return func() { delete(l.listeners, id) }
}

func (l *eventListeners) invoke() {
	for _, f := range l.listeners {
		f()
	}
}

// Win is the global window.
var Win = &Window{Object: js.Global.Get("window")}

// OnLoad registers a load handler for the window.
func (w *Window) OnLoad(cb func()) Unsubscribe {
	if w.onload == nil {
		w.onload = &eventListeners{}
		w.Set("onload", w.onload.invoke)
	}
	return w.onload.add(cb)
}

// OnScroll calles cb whenever the window is scrolled.
func (w *Window) OnScroll(cb func()) Unsubscribe {
	if w.onscroll == nil {
		w.onscroll = &eventListeners{}
		w.Set("onscroll", w.onscroll.invoke)
	}
	return w.onscroll.add(cb)
}

// Timer attempts to call cb after delay duration, unless it is cancelled.
func (w *Window) Timer(delay time.Duration, cb func()) Unsubscribe {
	var cancel Unsubscribe
	cancel = w.RepeatTimer(delay, func() { cancel(); cb() })
	return cancel
}

// RepeatTimer calls cb at the specified frequency until cancelled.
func (w *Window) RepeatTimer(frequency time.Duration, cb func()) Unsubscribe {
	id := w.Call("setInterval", cb, int(frequency/time.Millisecond))
	return func() { w.Call("clearInterval", id) }
}

// RequestAnimationFrame attempts to call cb as soon as possible.
func (w *Window) RequestAnimationFrame(cb func()) Unsubscribe {
	var cancelled bool
	w.Call("requestAnimationFrame", func() {
		if !cancelled {
			cb()
		}
	})
	return func() { cancelled = true }
}

// Location holds information about a URL
type Location struct {
	*js.Object

	// Hash is the anchor part (#) of a URL
	Hash string `js:"hash"`

	// Host is the hostname and port number of a URL
	Host string `js:"host"`

	// Hostname is the hostname of a URL
	Hostname string `js:"hostname"`

	// Href is the entire URL
	Href string `js:"href"`

	// Origin is the protocol, hostname and port number of a URL
	Origin string `js:"origin"`

	// Pathname is the path name of a URL
	Pathname string `js:"pathname"`

	// Port is the port number of a URL
	Port int `js:"port"`

	// Protocol is the protocol of a URL
	Protocol string `js:"protocol"`

	// Search is the querystring part of a URL
	Search string `js:"search"`
}

// OnHashChange registers a handler for changes to the Hash of the window's URL.
func (Location) OnHashChange(cb func(HashChangeEvent)) Unsubscribe {
	Win.Call("addEventListener", "hashchange", cb)
	return func() { Win.Call("removeEventListener", "hashchange", cb) }
}
