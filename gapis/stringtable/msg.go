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

package stringtable

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Text returns a plain-text representation of the message.
func (m *Msg) Text(tbl *StringTable) string {
	if m == nil {
		return ""
	}
	if tbl != nil {
		if entry, ok := tbl.Entries[m.Identifier]; ok {
			b := &bytes.Buffer{}
			writeNodes(entry, m.Arguments, b)
			return b.String()
		}
	}
	if len(m.Arguments) == 0 {
		return fmt.Sprintf("<%v>", m.Identifier)
	}
	args := make([]string, 0, len(m.Arguments))
	for k, v := range m.Arguments {
		args = append(args, fmt.Sprintf("%v: %v", k, v.Unpack()))
	}
	sort.Strings(args)
	return fmt.Sprintf("<%v [%v]>", m.Identifier, strings.Join(args, ", "))
}

func writeNodes(n interface{}, args map[string]*Value, w *bytes.Buffer) {
	switch n := n.(type) {
	case *Block:
		for _, n := range n.Children {
			writeNodes(n, args, w)
		}
	case *Text:
		w.WriteString(n.Text)
	case *Whitespace:
		w.WriteRune(' ')
	case *Parameter:
		v := args[n.Key]
		w.WriteString(fmt.Sprintf("%v", v))
	case *Link:
		writeNodes(n.Body, args, w)
	case *Bold:
		writeNodes(n.Body, args, w)
	case *Italic:
		writeNodes(n.Body, args, w)
	case *Underlined:
		writeNodes(n.Body, args, w)
	case *Heading:
		writeNodes(n.Body, args, w)
	case *Code:
		writeNodes(n.Body, args, w)
	case *List:
		for _, n := range n.Items {
			w.WriteString(" * ")
			writeNodes(n, args, w)
			w.WriteRune('\n')
		}
	}
}
