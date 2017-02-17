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
	"fmt"
	"io"
)

func canonical(w io.Writer) Handler {
	return func(page Page) error {
		canonicalPage(w, page)
		return nil
	}
}

func canonicalPage(w io.Writer, page Page) {
	joinSection := ""
	for _, section := range page {
		if len(section.Content) <= 0 {
			continue
		}
		io.WriteString(w, joinSection)
		fmt.Fprint(w, section.Key)
		io.WriteString(w, "{")
		joinItem := ""
		for _, item := range section.Content {
			// do we have anything to print
			child, isPage := item.Value.(Page)
			if item.Value == nil || isPage && len(child) <= 0 {
				continue
			}
			if item.Key != nil {
				io.WriteString(w, joinItem)
				fmt.Fprint(w, item.Key)
				joinItem = "="
			}
			// See if we are a nested value
			io.WriteString(w, joinItem)
			if isPage {
				io.WriteString(w, "[")
				canonicalPage(w, child)
				io.WriteString(w, "]")
			} else {
				switch value := item.Value.(type) {
				case string:
					fmt.Fprintf(w, "%q", value)
				case fmt.Stringer:
					fmt.Fprintf(w, "%q", value.String())
				case error:
					fmt.Fprintf(w, "%q", value.Error())
				default:
					fmt.Fprint(w, value)
				}
			}
			joinItem = ","
		}
		io.WriteString(w, "}")
		joinSection = ","
	}
}
