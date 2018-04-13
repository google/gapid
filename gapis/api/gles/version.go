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

package gles

import (
	"fmt"
	"regexp"
	"strconv"
)

// Version represents the GL version major and minor numbers,
// and whether its flavour is ES, as opposed to Desktop GL.
type Version struct {
	IsES  bool
	Major int
	Minor int
}

// AtLeast returns true if the version is greater or equal to major.minor.
func (v Version) AtLeast(major, minor int) bool {
	if v.Major > major {
		return true
	}
	if v.Major < major {
		return false
	}
	return v.Minor >= minor
}

// AtLeastES returns true if the version is OpenGL ES and is greater or equal to
// major.minor.
func (v Version) AtLeastES(major, minor int) bool {
	return v.IsES && v.AtLeast(major, minor)
}

// AtLeastGL returns true if the version is not OpenGL ES and is greater or
// equal to major.minor.
func (v Version) AtLeastGL(major, minor int) bool {
	return !v.IsES && v.AtLeast(major, minor)
}

// MaxGLSL returns the highest supported GLSL version for the given GL version.
func (v Version) MaxGLSL() Version {
	major, minor, isES := v.Major, v.Minor, v.IsES
	switch {
	case major == 2 && isES:
		return Version{Major: 1, Minor: 0}
	case major == 3 && isES:
		return Version{Major: 3, Minor: 0}

	case major == 2 && minor == 0 && !isES:
		return Version{Major: 1, Minor: 1}
	case major == 2 && minor == 1 && !isES:
		return Version{Major: 1, Minor: 2}
	case major == 3 && minor == 0 && !isES:
		return Version{Major: 1, Minor: 3}
	case major == 3 && minor == 1 && !isES:
		return Version{Major: 1, Minor: 4}
	case major == 3 && minor == 2 && !isES:
		return Version{Major: 1, Minor: 5}

	default:
		return Version{Major: major, Minor: minor}
	}
}

// AsInt returns the version in the form Mmm, where M is the major version and
// m is the minor version.
func (v Version) AsInt() int {
	return v.Major*100 + v.Minor*10
}

var versionRe = regexp.MustCompile(`^(OpenGL ES.*? )?(\d+)\.(\d+).*`)

// ParseVersion parses the GL version major, minor and flavour from the output of glGetString(GL_VERSION).
func ParseVersion(str string) (*Version, error) {
	if match := versionRe.FindStringSubmatch(str); match != nil {
		isES := len(match[1]) > 0 // Desktop GL doesn't have a flavour prefix.
		major, _ := strconv.Atoi(match[2])
		minor, _ := strconv.Atoi(match[3])
		return &Version{IsES: isES, Major: major, Minor: minor}, nil
	}
	return nil, fmt.Errorf("Unknown GL_VERSION format: %s", str)
}
