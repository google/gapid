// Copyright (C) 2019 Google Inc.
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

package app

import "context"

// Cleanup is a function that is invoked at a later time to perform the cleanup.
type Cleanup func(ctx context.Context)

// Then combines two clean up functions into a single cleanup function.
func (c Cleanup) Then(next Cleanup) Cleanup {
	if c == nil {
		return next
	}
	if next == nil {
		return c
	}
	return func(ctx context.Context) {
		c(ctx)
		next(ctx)
	}
}

// Invoke invokes the possibly nil cleanup safely. Returns a nil Cleanup, so
// this can be chained when invoking the cleanup as part of the error handling.
func (c Cleanup) Invoke(ctx context.Context) Cleanup {
	if c != nil {
		c(ctx)
	}
	return nil
}
