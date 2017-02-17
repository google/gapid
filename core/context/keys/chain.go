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

package keys

import (
	"context"
	"fmt"
)

// Link is the type for a link in a chain of values.
// This is used for values where you want to maintain the full list of values stored
// against that key, rather than just the most recently assigned value.
type Link struct {
	Value interface{}
	Next  *Link
}

// Format implements fmt.Formatter to print the full value chain.
func (l *Link) Format(f fmt.State, r rune) {
	if l.Next != nil {
		l.Next.Format(f, r)
		fmt.Fprint(f, "->")
	}
	fmt.Fprint(f, l.Value)
}

// Chain adds a new link to a value chain for the specified key.
func Chain(ctx context.Context, key interface{}, value interface{}) context.Context {
	old := ctx.Value(key)
	if old == nil {
		return WithValue(ctx, key, value)
	}
	link, islink := old.(*Link)
	if !islink {
		link = &Link{Value: old}
	}
	return WithValue(ctx, key, &Link{Value: value, Next: link})
}
