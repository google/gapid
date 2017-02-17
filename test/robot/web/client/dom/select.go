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

// Select represents an HTML <select> element.
type Select struct {
	*Element

	// Value is the currently currently selected value.
	Value string `js:"value"`

	// SelectedIndex is the index of the currently selected option.
	SelectedIndex int `js:"selectedIndex"`
}

// NewSelect returns a new Select element.
func NewSelect() *Select { return &Select{Element: newEl("select")} }

// Options returns the full list of options added to the select.
func (s *Select) Options() []*Option {
	l := s.Get("options")
	c := l.Length()
	out := make([]*Option, c)
	for i := 0; i < c; i++ {
		out[i] = &Option{Element: &Element{node: node{l.Index(i)}}}
	}
	return out
}

// OnChange registers a change handler for the Select element.
func (s *Select) OnChange(cb func()) Unsubscribe {
	s.Call("addEventListener", "change", cb)
	return func() { s.Call("removeEventListener", "change", cb) }
}

// Option represents an HTML <option> element.
type Option struct {
	*Element

	// Selected is true if the option is currently selected.
	Selected bool `js:"selected"`

	// Disabled is true if the option is currently disabled.
	Disabled bool `js:"disabled"`

	// Value is the value of the option.
	Value string `js:"value"`
}

// NewOption returns a new Option element.
func NewOption(text string, value string) *Option {
	o := &Option{Element: newEl("option")}
	o.Append(text)
	o.Value = value
	return o
}
