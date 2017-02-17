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
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/google/gapid/core/text"
	"github.com/google/gapid/core/text/reflow"
)

type (
	// KeyToValue is the interface for keys that control their own assignement character.
	KeyToValue interface {
		// KeyToValue returns the string to use to join the key to the value.
		KeyToValue() string
	}

	format struct {
		noKeys    bool
		include   Relevance
		multiline bool
		indent    string
		limit     int
		elide     string
	}
)

func printer(format *format) func(w io.Writer) Handler {
	return func(w io.Writer) Handler {
		flow := reflow.New(w)
		flow.Indent = format.indent
		buf := bufio.NewWriter(flow)
		return func(page Page) error {
			printPage(buf, page, format)
			buf.Flush()
			flow.Flush()
			return nil
		}
	}
}

func printValue(w io.Writer, value interface{}) {
	switch value := value.(type) {
	case time.Time:
		io.WriteString(w, value.Format(time.Stamp))
	case error:
		io.WriteString(w, "⦕")
		fmt.Fprint(w, value)
		io.WriteString(w, "⦖")
	default:
		fmt.Fprint(w, value)
	}
}

func printPage(w *bufio.Writer, page Page, format *format) {
	joinSection := ""
	for _, section := range page {
		if len(section.Content) <= 0 {
			continue
		}
		if section.Relevance > format.include {
			continue
		}
		multiline := section.Multiline && format.multiline
		if multiline {
			w.WriteRune(reflow.EOL)
			w.WriteRune(reflow.Indent)
			joinSection = ""
		} else {
			w.WriteString(joinSection)
			joinSection = ""
		}
		joinItem := ""
		for _, item := range section.Content {
			// do we have anything to print
			child, isPage := item.Value.(Page)
			if item.Value == nil || isPage && len(child) <= 0 {
				continue
			}
			if item.Key != nil && !format.noKeys && !OmitKey(item.Key) {
				// Print the key
				w.WriteString(joinItem)
				fmt.Fprint(w, item.Key)
				// Set the key to value joiner
				joinItem = "="
				if joiner, ok := item.Key.(KeyToValue); ok {
					joinItem = joiner.KeyToValue()
				}
				if multiline {
					joinItem = string(reflow.Column) + joinItem + string(reflow.Space)
				}
			}
			w.WriteString(joinItem)
			// See if we are a nested value
			switch {
			case isPage && multiline:
				printPage(w, child, format)
			case isPage:
				w.WriteString("{")
				printPage(w, child, format)
				w.WriteString("}")
			case format.limit <= 0:
				w.WriteRune(reflow.Disable)
				printValue(w, item.Value)
				w.WriteRune(reflow.Enable)
			default:
				// A simple value, print it with limits
				w.WriteRune(reflow.Disable)
				out := text.NewLimitWriter(w, format.limit, format.elide)
				printValue(out, item.Value)
				out.Flush()
				w.WriteRune(reflow.Enable)
			}
			if multiline {
				joinItem = string(reflow.EOL)
			} else {
				joinItem = ","
			}
		}
		if multiline {
			w.WriteRune(reflow.Unindent)
			joinSection = string(reflow.EOL)
		} else {
			joinSection = ":"
		}
	}
}
