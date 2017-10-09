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
}

// Callstack is a full stacktrace
type Callstack []uintptr

const stackLimit = 50

// Capture returns a full stacktrace.
func Capture() Callstack {
	callers := make([]uintptr, stackLimit)
	count := runtime.Callers(0, callers)
	return Callstack(callers[:count])
}

// All returns all the entries for the stack trace.
func (c Callstack) All() []Entry {
	calls := make([]Entry, len(c))
	for i := range c {
		calls[i] = c.Get(i)
	}
	return calls
}

// Get returns the callstack entry at idx.
func (c Callstack) Get(idx int) Entry {
	pc := c[idx]
	// See documentation for runtime.Callers for why we use pc-1 in here
	f := runtime.FuncForPC(pc - 1)
	filename, line := f.FileLine(pc - 1)
	dir, basename := path.Split(filename)
	// name is of the form github.com/google/gapid/framework/log.StacktraceOnError
	// we find the last /, then find the next . to split the function name from the package name
	name := f.Name()
	i := strings.LastIndex(name, "/")
	i += strings.IndexRune(name[i+1:], '.')
	return Entry{
		Location: Location{
			Directory: dir,
			File:      basename,
			Line:      line,
		},
		Function: Function{
			Package: name[:i+1],
			Name:    name[i+2:],
		},
	}
}

func (e Entry) String() string {
	return fmt.Sprint("⇒ ", e.Location, ":", e.Function)
}

func (l Location) String() string {
	return fmt.Sprint(l.File, "@", l.Line)
}

func (f Function) String() string {
	return f.Name
}
