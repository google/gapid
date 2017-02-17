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

import "context"

// keyType is hidden type so nobody can use the key list directly
type keySetType int

// keySet is the hidden key used to store the key list on the context.
const keySet = keySetType(0)

// Get returns the list of potential keys to use.
func Get(ctx context.Context) []interface{} {
	seen := map[interface{}]bool{}
	result := make([]interface{}, 0, 10)
	for link, _ := ctx.Value(keySet).(*Link); link != nil; link = link.Next {
		if !seen[link.Value] {
			seen[link.Value] = true
			result = append(result, link.Value)
		}
	}
	return result
}

// WithValue registers the key as well as adding the value to the context.
func WithValue(ctx context.Context, key interface{}, value interface{}) context.Context {
	old, _ := ctx.Value(keySet).(*Link)
	ctx = context.WithValue(ctx, key, value)
	return context.WithValue(ctx, keySet, &Link{Value: key, Next: old})
}

// Clone copies values from one context to another.
// This is used to produce associated but detached contexts.
func Clone(ctx context.Context, from context.Context) context.Context {
	for _, key := range Get(from) {
		ctx = WithValue(ctx, key, from.Value(key))
	}
	return ctx
}
