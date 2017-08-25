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

package pack

import (
	"fmt"

	"github.com/google/gapid/core/fault"
)

const (
	// Magic is the file magic that prefixes all pack files.
	Magic = "protopack"

	// ErrIncorrectMagic is the error returned when the file header is not matched.
	ErrIncorrectMagic = fault.Const("Incorrect pack magic header")

	initalBufferSize = 4096
	maxVarintSize    = 10

	tagFirstGroup     = -1
	tagGroupFinalizer = 0
	tagDeclareType    = 1
	tagFirstObject    = 2
)

var (
	// MinVersion is the current minimum supported version of pack files.
	MinVersion = &Version{1, 1}

	// MaxVersion is the current maximum supported version of pack files.
	MaxVersion = &Version{1, 1}

	// version is the version written by this package.
	version = &Version{1, 1}

	magicBytes = []byte(Magic)
)

func tyIdxAndGroupFromTag(tag uint64) (tyIdx uint64, isGroup bool) {
	if int64(tag) <= tagFirstGroup {
		return uint64(tagFirstGroup - int64(tag)), true
	}
	return uint64(tag - tagFirstObject), false
}

func tagFromTyIdxAndGroup(tyIdx uint64, isGroup bool) (tag int64) {
	if isGroup {
		return tagFirstGroup - int64(tyIdx)
	}
	return tagFirstObject + int64(tyIdx)
}

// ErrUnsupportedVersion is the error returned when the header version is one
// this package cannot handle.
type ErrUnsupportedVersion struct{ Version Version }

func (e ErrUnsupportedVersion) Error() string {
	return fmt.Sprintf("Unsupported pack file version: %+v", e.Version)
}

// GreaterThan returns true if v is greater than o.
func (v *Version) GreaterThan(o *Version) bool {
	switch {
	case v.Major > o.Major:
		return true
	case v.Major < o.Major:
		return false
	case v.Minor > o.Minor:
		return true
	case v.Minor < o.Minor:
		return false
	}
	return false
}

// LessThan returns true if v is less than o.
func (v *Version) LessThan(o *Version) bool {
	switch {
	case v.Major < o.Major:
		return true
	case v.Major > o.Major:
		return false
	case v.Minor < o.Minor:
		return true
	case v.Minor > o.Minor:
		return false
	}
	return false
}
