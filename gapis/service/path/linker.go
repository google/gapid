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

package path

import "context"

// Linker is the interface implemented by types that can point to other objects.
type Linker interface {
	// Link returns the link to the pointee of this object at p.
	// If nil, nil is returned then the path cannot be followed.
	Link(ctx context.Context, p Node, r *ResolveConfig) (Node, error)
}
