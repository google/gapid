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

package stacktrace

import (
	"fmt"
	"path"
	"runtime"
	"strings"
)

// Location holds the physical location of a stack entry.
type Location struct {
	// Directory is the directory the source file is from.
	Directory string
	// File is the filename of the source file.
	File string
	// Line is the line index in the file.
	Line int
}

// Function holds the logical location of a stack entry.
type Function struct {
	// Package is the go package the stack entry is from.
	Package string
	// Name is the function name the stack entry is from.
	Name string
}

// Entry holds the human understandable form of a StackTrace entry.
type Entry struct {
	// Location holds the physical location for this entry.
	Location Location
	// Location holds the logical location for this entry.
	Function Function
	// PC is the program counter for this entry.
	PC uintptr
}

// Callstack is a full stacktrace
type Callstack []uintptr

const stackLimit = 50

// Capture returns a full stacktrace.
func Capture() Callstack {
	callers := make([]uintptr, stackLimit)
	count := runtime.Callers(2, callers)
	stack := callers[:count]
	return Callstack(stack)
}

// Entries returns all the entries for the stack trace.
func (c Callstack) Entries() []Entry {
	frames := runtime.CallersFrames([]uintptr(c))
	out := []Entry{}
	for {
		frame, more := frames.Next()
		dir, file := path.Split(frame.File)
		fullname := frame.Function
		var pkg, name string
		if i := strings.LastIndex(fullname, "/"); i > 0 {
			i += strings.IndexRune(fullname[i+1:], '.')
			// name is of the form github.com/google/gapid/framework/log.StacktraceOnError
			// we find the last /, then find the next . to split the function name from the package name
			pkg, name = fullname[:i+1], fullname[i+2:]
		}
		out = append(out, Entry{
			Location: Location{
				Directory: dir,
				File:      file,
				Line:      frame.Line,
			},
			Function: Function{
				Package: pkg,
				Name:    name,
			},
			PC: frame.PC,
		})
		if !more {
			break
		}
	}

	return out
}

func (c Callstack) String() string {
	lines := make([]string, len(c))
	for i, e := range c.Entries() {
		lines[i] = e.String()
	}
	return strings.Join(lines, "\n")
}

func (e Entry) String() string {
	return fmt.Sprint("⇒ ", e.Location, ":", e.Function)
}

func (l Location) String() string {
	const strip = "github.com/google/gapid/"
	dir := l.Directory
	if i := strings.LastIndex(dir, strip); i > 0 {
		dir = dir[i+len(strip):]
	}
	return fmt.Sprint(dir, l.File, "@", l.Line)
}

func (f Function) String() string {
	return f.Name
}
