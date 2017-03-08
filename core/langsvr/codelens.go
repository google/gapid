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

import "context"

// CodeLens represents a command that should be shown along with source text,
// like the number of references, a way to run tests, etc.
type CodeLens struct {
	// Range is the document range this codelens spans.
	Range Range

	// Resolve returns the Command for this CodeLens
	Resolve func(context.Context) Command
}
