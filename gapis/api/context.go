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

package api

import "github.com/google/gapid/core/data/id"

// ContextID is the unique identifier for a context.
type ContextID id.ID

// Context represents a graphics API's unique context of execution.
type Context interface {
	// Name returns the display-name of the context.
	Name() string

	// ID returns the context's unique identifier
	ID() ContextID
}
