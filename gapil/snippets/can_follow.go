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

package snippets

import "github.com/google/gapid/framework/binary"

// CanFollow used to indicate that the item at the associated pathway is
// followable. Following the instance path yields an instance path to a
// related object typically in the state or memory view. Followability
// is not captured in the API file, but only in the Go code by the interface
// path.Linker. Consequently we generate these snippets at startup using
// reflection rather than by using annotate.
type CanFollow struct {
	binary.Generate
	Path Pathway
}
