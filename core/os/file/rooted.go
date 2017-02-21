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

package file

type (
	// Rooted is the type for a root path and fragment that when joined together form a full path.
	// The fragment must not be an absolute path.
	Rooted struct {
		Root     Path
		Fragment string
	}
)

// Join returns a Rooted path from the root and fragment supplied.
func Join(root Path, fragment string) Rooted {
	return Rooted{Root: root, Fragment: fragment}
}

// Path joins the root and fragment together into a normal absolute path.
func (r Rooted) Path() Path {
	return r.Root.Join(string(r.Fragment))
}
