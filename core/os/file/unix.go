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

//go:build !windows

package file

// IsJunction returns false as junctions are not supported by this platform.
func IsJunction(path Path) bool {
	return false
}

// Junction calls through to Symlink as junctions are not supported by this
// platform.
func Junction(link, target Path) error {
	return Symlink(link, target)
}

// Illegal characters in unix file paths.
// Excluding slashes here as we're considering paths, not filenames.
var illegalPathChars = "\x00"
