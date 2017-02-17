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

package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	st "github.com/google/gapid/gapis/stringtable"
	"github.com/google/gapid/gapis/stringtable/minidown"
	"github.com/google/gapid/gapis/stringtable/minidown/node"
)

// ParameterID is a unique identifier for a stringtable parameter
type ParameterID struct {
	ParameterKey string
	EntryKey     string
}

// ParameterTypeMap is a map of parameter to its type.
type ParameterTypeMap map[ParameterID]string

type tagSet map[node.Tag]bool

// Parse parses the given string table file.
func Parse(filename, data string) (*st.StringTable, ParameterTypeMap, []error) {
	var errs []error

	// Start by parsing the minidown
	minidown, parseErrs := minidown.Parse(filename, data)
	if len(parseErrs) > 0 {
		errs = append(errs, parseErrs)
	}

	// Process the minidown into stringtable entries.
	entries, paramTypeMap, processErrs := process(minidown)
	if len(processErrs) > 0 {
		errs = append(errs, processErrs...)
	}

	table := &st.StringTable{
		Info: &st.Info{
			CultureCode: strings.Split(filepath.Base(filename), ".")[0],
		},
		Entries: entries,
	}
	return table, paramTypeMap, errs
}

func process(in node.Node) (map[string]*st.Node, ParameterTypeMap, []error) {
	var errs []error
	table := map[string]*st.Node{}
	paramTypeMap := ParameterTypeMap{}

	var current string        // Name of the current entry.
	processedTags := tagSet{} // Map of processed tags for the current entry.
	newlines := 0             // Number of pending newlines.

	add := func(n node.Node) {
		if len(current) == 0 {
			return // No entry assigned yet. Perhaps a file header message?
		}

		// Convert the minidown node to a stringtable node.
		converted, e := convert(n, current, paramTypeMap, processedTags)
		errs = append(errs, e...)

		// Check to see if this is a new entry, or appending to an existing.
		existing := table[current]
		if existing == nil {
			// New entry. Simply assign to the table.
			table[current] = converted
			newlines = 0
			return
		}

		block := existing.GetBlock()
		if block == nil {
			// Multiple nodes for entry. Box first entry into a block.
			block = &st.Block{Children: []*st.Node{existing}}
		}

		// Add any pending line breaks
		if newlines > 0 {
			block.Children = append(block.Children, &st.Node{
				Node: &st.Node_LineBreak{
					LineBreak: &st.LineBreak{Lines: uint32(newlines)},
				},
			})
			newlines = 0
		}

		// Check if conversion was successful.
		if converted != nil {
			// Add new node to the block.
			block.Children = append(block.Children, converted)
		}

		// Update table.
		table[current] = &st.Node{Node: &st.Node_Block{Block: block}}
	}

	root, ok := in.(*node.Block)
	if !ok {
		return nil, nil, []error{fmt.Errorf("Expected root to be a block, instead got %T", in)}
	}

	for _, n := range root.Children {
		switch n := n.(type) {
		case *node.NewLine:
			newlines++

		case *node.Heading:
			// Check for a new table entry. These are declared as H1 headings.
			if n.Scale == 1 {
				if text, ok := n.Body.(*node.Text); ok {
					// Change current.
					current = text.Text

					// Check for duplicates.
					if _, dup := table[current]; dup {
						errs = append(errs, fmt.Errorf("Duplicate stringtable entry '%s'", current))
					}
					// Clear occurrence map
					processedTags = make(tagSet)
				} else {
					errs = append(errs, fmt.Errorf("Entry name must be simple text, instead got %T", n.Body))
				}
			} else {
				add(n) // A heading, but not H1. Add to existing entry.
			}

		default:
			add(n)
		}
	}
	return table, paramTypeMap, errs
}

func convert(in node.Node, currentKey string, typeMap ParameterTypeMap, tagSet tagSet) (*st.Node, []error) {
	switch in := in.(type) {
	case *node.Text:
		return &st.Node{Node: &st.Node_Text{Text: &st.Text{Text: in.Text}}}, nil

	case *node.Whitespace:
		return &st.Node{Node: &st.Node_Whitespace{Whitespace: &st.Whitespace{}}}, nil

	case *node.Link:
		body, errsA := convert(in.Body, currentKey, typeMap, tagSet)
		target, errsB := convert(in.Target, currentKey, typeMap, tagSet)
		return &st.Node{Node: &st.Node_Link{Link: &st.Link{Body: body, Target: target}}}, append(errsA, errsB...)

	case *node.Tag:
		if exists := tagSet[*in]; !exists {
			tagSet[*in] = true
			param := &st.Parameter{Key: in.Identifier}
			paramID := ParameterID{ParameterKey: param.Key, EntryKey: currentKey}
			// Check if we've met parameter with such an identifier, but type differs
			// (therefore tagSet contains different node).
			if t, typeExists := typeMap[paramID]; typeExists {
				return nil, []error{fmt.Errorf("Entry %s contains duplicate parameters"+
					" %s with different types: %s != %s", paramID.EntryKey,
					paramID.ParameterKey, t, in.Type)}
			}
			typeMap[paramID] = in.Type
			return &st.Node{Node: &st.Node_Parameter{Parameter: param}}, nil
		}
		return nil, nil

	default:
		return nil, []error{fmt.Errorf("Stringtable does not currently support %T", in)}
	}
}
