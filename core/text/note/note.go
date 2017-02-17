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

package note

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type (
	// Item is a single tagged value entry for a page.
	Item struct {
		// Key is the key for this item.
		Key interface{}
		// Value is the full value of the item.
		Value interface{}

		// orderBy is only filled in and used when disambiguating the sort order
		orderBy string
	}

	// Order is the interface a key should support to control it's sort order.
	Order interface {
		OrderBy() string
	}

	// OmitKeyMarker is the interface that should be satisfied by item keys that want to control
	// whether their key is used or not.
	OmitKeyMarker interface {
		// OmitKey should return true if the key should not be displayed.
		OmitKey() bool
	}

	// SectionInfo describes the properties of a section.
	SectionInfo struct {
		// Key is the unique identifier of the section within a page.
		Key string
		// Order is used to sort sections.
		Order int
		// Relevance is the default relevance for items in this section.
		Relevance Relevance
		// Multiline indicates this section should multi-line in verbose modes.
		Multiline bool
	}

	// Section holds a list of items.
	Section struct {
		// SectionInfo is the settigs for the section.
		SectionInfo
		// Value is the full value of the item.
		Content []Item
	}

	// Page is a collection of sections.
	Page []Section

	// Pad is a list of pages.
	Pad []Page

	// Handler is the type for a function that can be handed pages to process.
	Handler func(Page) error
)

// OrderBy returns a string used for sorting items on a note page.
func (i *Item) OrderBy() string {
	if i.orderBy != "" {
		return i.orderBy
	}
	switch key := i.Key.(type) {
	case Order:
		i.orderBy = key.OrderBy()
	case string:
		i.orderBy = key
	case fmt.Stringer:
		i.orderBy = key.String()
	default:
		i.orderBy = fmt.Sprint(key)
	}
	return i.orderBy
}

// Transcribe allows a SectionInfo to add values to the corresponding section of a page.
func (i SectionInfo) Transcribe(ctx context.Context, page *Page, value interface{}) {
	page.Append(i, nil, value)
}

// Len returns the number of items in the section.
func (s Section) Len() int { return len(s.Content) }

// Swap does an in place swap of two items in the section.
func (s Section) Swap(i, j int) { s.Content[i], s.Content[j] = s.Content[j], s.Content[i] }

// Less can be used to form a total ordering of the items on the page.
// They are orderd by section, and then by key within a section.
func (s Section) Less(i, j int) bool {
	return strings.Compare(s.Content[i].OrderBy(), s.Content[j].OrderBy()) < 0
}

// Sort does a deep in page sort of the page.
func (s Section) Sort() {
	sort.Stable(s)
	for _, item := range s.Content {
		if child, isPage := item.Value.(Page); isPage {
			child.Sort()
		}
	}
}

// Clone does a deep clone of the section.
// This clones all nested pages, but not the keys or values they refer to.
func (s Section) Clone() Section {
	clone := s
	clone.Content = make([]Item, len(s.Content))
	for i, item := range s.Content {
		clone.Content[i] = item
		if child, isPage := item.Value.(Page); isPage {
			clone.Content[i].Value = child.Clone()
		}
	}
	return clone
}

// Append adds an item to a section.
// If the section is not present, it will be added.
func (p *Page) Append(info SectionInfo, key interface{}, value interface{}) {
	for i, match := range *p {
		if match.Key == info.Key {
			match.Content = append(match.Content, Item{Key: key, Value: value})
			(*p)[i] = match
			return
		}
	}
	*p = append(*p, Section{SectionInfo: info, Content: []Item{{Key: key, Value: value}}})
}

// Len returns the number of items on the page.
func (p Page) Len() int { return len(p) }

// Swap does an in place swap of two items on the page.
func (p Page) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Less can be used to form a total ordering of the items on the page.
// They are orderd by section, and then by key within a section.
func (p Page) Less(i, j int) bool {
	return p[i].Order < p[j].Order
}

// Sort does a deep in page sort of the page.
func (p Page) Sort() {
	sort.Stable(p)
	for _, section := range p {
		section.Sort()
	}
}

// Clone does a deep clone of the page.
// This clones all nested pages, but not the keys or values they refer to.
func (p Page) Clone() Page {
	clone := make([]Section, len(p))
	for i, section := range p {
		clone[i] = section.Clone()
	}
	return clone
}

// Sorter returns a handler that sorts pages before handing them on to the supplied handler.
// Note that this will sort the pages in place, so the original will be modified!
func Sorter(h Handler) Handler {
	return func(page Page) error {
		page.Sort()
		return h(page)
	}
}

// Cloner returns a handler that clones pages and hands the clone to the supplied handler.
// This can be used before handlers that modify the pages in place, like Sorter.
func Cloner(h Handler) Handler {
	return func(page Page) error {
		return h(page.Clone())
	}
}
