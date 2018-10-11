// Copyright (C) 2018 Google Inc.
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

package device

// VersionMatch indicates how accuratly two OS versions match.
type VersionMatch int

const (
	// NoMatch means the versions don't match at all.
	NoMatch VersionMatch = iota
	// MajorMatch means the versions only match up to the major version.
	MajorMatch
	// MajorAndMinorMatch means the versions only match up to the minor version.
	MajorAndMinorMatch
	// CompleteMatch means the versions matched completely.
	CompleteMatch
)

// CompareVersions compares the version of the two OSes.
func (o1 *OS) CompareVersions(o2 *OS) VersionMatch {
	switch {
	case o1 == nil, o2 == nil, o1.Kind != o2.Kind, o1.MajorVersion != o2.MajorVersion:
		return NoMatch
	case o1.MinorVersion != o2.MinorVersion:
		return MajorMatch
	case o1.PointVersion != o2.PointVersion:
		return MajorAndMinorMatch
	default:
		return CompleteMatch
	}
}

func (o *OSKind) Choose(c interface{}) {
	*o = c.(OSKind)
}
