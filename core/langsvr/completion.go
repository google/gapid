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

package langsvr

import "github.com/google/gapid/core/langsvr/protocol"

// CompletionItem represents a completion item to be presented in the editor.
type CompletionItem struct {
	// The label of this completion item. By default also the text that is
	// inserted when selecting this completion.
	Label string

	// The kind of this completion item. Based of the kind an icon is chosen by
	// the editor.
	Kind CompletionItemKind

	// An optional human-readable string with additional information about this
	// item, like type or symbol information.
	Detail string

	// An optional human-readable string that represents a doc-comment.
	Documentation string

	// An optional string that should be used when comparing this item with
	// other items.
	SortText string

	// An optional string that should be used when filtering a set of completion
	// items.
	FilterText string

	// An optional string that should be inserted into the document when
	// selecting this completion. Defaults to the label.
	InsertText string
}

// CompletionList is a list of completion items.
type CompletionList struct {
	// If true then the list it not complete.
	// Further typing should result in recomputing this list.
	Incomplete bool

	// The completion items.
	Items []CompletionItem
}

// Add appends a completion item to the list.
func (c *CompletionList) Add(label string, kind CompletionItemKind, detail string) {
	c.Items = append(c.Items, CompletionItem{
		Label:  label,
		Kind:   kind,
		Detail: detail,
	})
}

func (c CompletionItem) toProtocol() protocol.CompletionItem {
	kind := protocol.CompletionItemKind(c.Kind)
	return protocol.CompletionItem{
		Label:         c.Label,
		Kind:          &kind,
		Detail:        strOrNil(c.Detail),
		Documentation: strOrNil(c.Documentation),
		SortText:      strOrNil(c.SortText),
		FilterText:    strOrNil(c.FilterText),
		InsertText:    strOrNil(c.InsertText),
	}
}

func (c CompletionList) toProtocol() protocol.CompletionList {
	out := protocol.CompletionList{}
	out.IsIncomplete = c.Incomplete
	out.Items = make([]protocol.CompletionItem, len(c.Items))
	for i, c := range c.Items {
		out.Items[i] = c.toProtocol()
	}
	return out
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CompletionItemKind is the kind of a completion entry.
type CompletionItemKind int

const (
	Text        = CompletionItemKind(protocol.Text)
	Method      = CompletionItemKind(protocol.Method)
	Function    = CompletionItemKind(protocol.Function)
	Constructor = CompletionItemKind(protocol.Constructor)
	Field       = CompletionItemKind(protocol.Field)
	Variable    = CompletionItemKind(protocol.Variable)
	Class       = CompletionItemKind(protocol.Class)
	Interface   = CompletionItemKind(protocol.Interface)
	Module      = CompletionItemKind(protocol.Module)
	Property    = CompletionItemKind(protocol.Property)
	Unit        = CompletionItemKind(protocol.Unit)
	Value       = CompletionItemKind(protocol.Value)
	Enum        = CompletionItemKind(protocol.Enum)
	Keyword     = CompletionItemKind(protocol.Keyword)
	Snippet     = CompletionItemKind(protocol.Snippet)
	Color       = CompletionItemKind(protocol.Color)
	File        = CompletionItemKind(protocol.File)
	Reference   = CompletionItemKind(protocol.Reference)
)
