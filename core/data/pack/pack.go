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
	// ErrIncorrectMagic is the error returned when the file header is not matched.
	ErrIncorrectMagic = fault.Const("Incorrect pack magic header")

	initalBufferSize = 4096
	maxVarintSize    = 10
)

var (
	// MinMajorVersion is the current minimum supported major version of pack files.
	MinMajorVersion = 2

	// MaxMajorVersion is the current maximum supported major version of pack files.
	MaxMajorVersion = 2

	// header is the header written by this package including the version.
	header = []byte("ProtoPack\r\n2.0\n\x00")
)

type Version struct {
	Major int // Major version is incremented for format breaking changes.
	Minor int // Minor version is incremented backwards compatible changes.
}

// ErrUnsupportedVersion is the error returned when the header version is one
// this package cannot handle.
type ErrUnsupportedVersion struct{ Version Version }

func (e ErrUnsupportedVersion) Error() string {
	return fmt.Sprintf("Unsupported pack file version: %v.%v", e.Version.Major, e.Version.Minor)
}
