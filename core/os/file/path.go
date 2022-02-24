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

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Path is a clean absolute path with platform specific separators.
type Path struct{ value string }

const homeDirTilde = "~"
const homeDirPrefix = homeDirTilde + string(filepath.Separator)

// Abs is the primary constructor of new Path objects from strings using either the / or system separator.
func Abs(path string) Path {
	if strings.HasPrefix(path, homeDirPrefix) {
		u, _ := user.Current()
		path = filepath.Join(u.HomeDir, strings.TrimLeft(path, homeDirTilde))
	}
	abs, err := filepath.Abs(filepath.FromSlash(path))
	if err != nil {
		return Path{path}
	}
	return Path{filepath.Clean(abs)}
}

// Temp creates a new temp file and returns its path.
func Temp() (Path, error) {
	p, err := ioutil.TempFile("", "gapid")
	if err != nil {
		return Path{}, err
	}
	p.Close()
	return Abs(p.Name()), nil
}

// TempWithExt creates a new temp file with the given name and extension and returns its path.
func TempWithExt(name string, ext string) (Path, error) {
	p, err := ioutil.TempFile("", fmt.Sprintf("%s*.%s", name, ext))
	if err != nil {
		return Path{}, err
	}
	p.Close()
	return Abs(p.Name()), nil
}

// ExecutablePath returns the path to the running executable.
func ExecutablePath() Path {
	path := Abs(os.Args[0])
	if path.Exists() {
		return path
	}
	path, err := FindExecutable(os.Args[0])
	if err != nil {
		panic(err)
	}
	return path
}

// FindExecutable searches the system search path for the named binary, and
// returns a non empty Path if found. OS executable file extensions (".exe") are
// automatically considered when searching.
func FindExecutable(name string) (Path, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return Path{}, err
	}
	return Abs(path), nil
}

// IsEmpty returns true if the path has no value.
func (p Path) IsEmpty() bool { return p.value == "" }

// IsDir returns true if the path exists and is a directory.
func (p Path) IsDir() bool {
	info := p.Info()
	return info != nil && info.IsDir()
}

// IsExecutable returns true if the file at p.value exists, is executable
// and is not a directory.
func (p Path) IsExecutable() bool {
	if runtime.GOOS == "windows" {
		return true
	}
	info := p.Info()
	return info != nil && !info.IsDir() && (info.Mode().Perm()&0111) != 0
}

// System returns the full absolute path using the system separator.
func (p Path) System() string { return p.value }

// URL returns the full absolute path using the / separator prefixed by the URL
// file scheme.
func (p Path) URL() *url.URL {
	result, err := url.Parse("file:///" + filepath.ToSlash(p.value))
	if err != nil {
		panic(err)
	}
	return result
}

// Slash returns the full absolute path using the / separator.
func (p Path) Slash() string { return filepath.ToSlash(p.value) }

// The default string form uses the system representation.
func (p Path) String() string { return p.value }

// Set assigns the value of the path.
// This conforms to the flag variable interface.
func (p *Path) Set(value string) error {
	*p = Abs(value)
	return nil
}

// Parent returns the parent directory of the path, if it has one.
// It will return the canonical version of the path if it is a root.
func (p Path) Parent() Path { return Path{filepath.Dir(p.value)} }

// Basename returns the name part of the path (without directories).
func (p Path) Basename() string { return filepath.Base(p.value) }

// Ext returns the extension of the path, including the '.', or an empty string
// if the path has no extension.
func (p Path) Ext() string { return filepath.Ext(p.value) }

// Smash returns the parent, name and extension of a path.
func (p Path) Smash() (Path, string, string) {
	dir, name := filepath.Split(p.value)
	ext := filepath.Ext(name)
	name = name[:len(name)-len(ext)]
	return Path{dir}, name, ext
}

// Split returns the parent and name of the path.
func (p Path) Split() (Path, string) {
	dir, name := filepath.Split(p.value)
	return Path{dir}, name
}

// SplitExt returns the path excluding the file extension and the file
// extension. The extension includes the '.'.
func (p Path) SplitExt() (Path, string) {
	ext := p.Ext()
	return Path{p.value[:len(p.value)-len(ext)]}, ext
}

// NoExt returns the path excluding the file extension or '.'.
func (p Path) NoExt() Path {
	ext := p.Ext()
	return Path{p.value[:len(p.value)-len(ext)]}
}

// Join returns a path formed from joining this base with a child path.
// If there are any / characters in the strings, they will be converted to they system separator.
func (p Path) Join(join ...string) Path {
	if len(join) == 0 {
		return p
	}
	trailing := filepath.Join(join...)
	trailing = filepath.FromSlash(trailing)
	return Path{filepath.Clean(filepath.Join(p.value, trailing))}
}

// Files reads the list of files in path.
func (p Path) Files() PathList {
	infos, _ := ioutil.ReadDir(p.value)
	list := PathList{}
	for _, i := range infos {
		if !i.IsDir() {
			list = append(list, Path{filepath.Join(p.value, i.Name())})
		}
	}
	return list
}

// Walk walks path.
func (p Path) Walk(walkFn func(path Path, err error) error) error {
	return filepath.Walk(p.System(), func(path string, info os.FileInfo, err error) error {
		if err == nil {
			return walkFn(Path{path}, nil)
		} else {
			return walkFn(Path{}, err)
		}
	})
}

// Glob adds each pattern to the path in turn, and uses filepath.Glob to resolve that pattern to a list of files.
func (p Path) Glob(pattern ...string) PathList {
	list := PathList{}
	for _, test := range pattern {
		matches, err := filepath.Glob(filepath.Join(p.value, test))
		if err != nil {
			return PathList{}
		}
		for _, match := range matches {
			list = append(list, Path{match})
		}
	}
	return list
}

// Info returns the file information for ths path.
func (p Path) Info() os.FileInfo {
	info, _ := os.Stat(p.value)
	return info
}

// Exists returns true if this File exists.
func (p Path) Exists() bool {
	return p.Info() != nil
}

// Timestamp returns the last modified time reported by the file system.
func (p Path) Timestamp() time.Time {
	info := p.Info()
	if info == nil || info.IsDir() {
		return time.Time{}
	}
	return info.ModTime()
}

// Contains returns true if this directory contains the other.
func (p Path) Contains(other Path) bool {
	if len(p.value) >= len(other.value) {
		return false
	}
	return strings.HasPrefix(other.value, p.value)
}

// RelativeTo returns the path of this File relative to base.
func (p Path) RelativeTo(base Path) (string, error) {
	return filepath.Rel(base.value, p.value)
}

// ChangeExt returns a new Path with the extension changed to ext.
// ext should being with a '.' to change extension, or be an empty string to
// remove the extension.
func (p Path) ChangeExt(ext string) Path {
	prev := filepath.Ext(p.value)
	return Path{p.value[:len(p.value)-len(prev)] + ext}
}

// Matches returns true if the Path matches any of the patterns.
func (p Path) Matches(patterns ...string) bool {
	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, p.Basename())
		if matched {
			return true
		}
	}
	return false
}

// MarshalJSON implements custom JSON marshalling to write paths as simple JSON strings.
func (p Path) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

// UnmarshalJSON implements custom JSON unmarshalling to read strings a paths.
func (p *Path) UnmarshalJSON(data []byte) error {
	v := ""
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*p = Abs(v)
	return nil
}

// SanitizePath returns the path p with illegal file path characters replaced with '-'.
func SanitizePath(p string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(illegalPathChars, r) {
			return '-'
		}
		return r
	}, p)
}
