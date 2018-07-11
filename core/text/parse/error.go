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

package parse

import (
	"fmt"
	"runtime"

	"github.com/google/gapid/core/data/compare"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/text/parse/cst"
)

var (
	// ParseErrorLimit is the maximum number of errors before a parse is aborted.
	ParseErrorLimit = 10
)

const (
	// AbortParse is paniced when a parse cannot continue. It is recovered at the
	// top level, to allow the errors to be cleanly returned to the caller.
	AbortParse = fault.Const("abort")
)

// Error represents the information that us useful in debugging a parse failure.
type Error struct {
	// At is the parse fragment that was being processed when the error was encountered.
	At cst.Fragment
	// Message is the message associated with the error.
	Message string
	// Stack is the captured stack trace at the point the error was noticed.
	Stack []byte
}

// ErrorList is a convenience type for managing lists of errors.
type ErrorList []Error

func (errs ErrorList) Error() string {
	if len(errs) == 0 {
		return ""
	}
	return fmt.Sprintf("%d errors, first error was: %v", len(errs), errs[0])
}

func (err Error) Error() string {
	return err.Message
}

func (err Error) Format(f fmt.State, c rune) {
	if err.At == nil {
		fmt.Fprintf(f, "%s", err.Message)
		return
	}
	t := err.At.Tok()
	line, column := t.Cursor()
	filename := "-"
	if t.Source != nil && t.Source.Filename != "" {
		filename = t.Source.Filename
	}
	fmt.Fprintf(f, "%s:%v:%v: %s", filename, line, column, err.Message)
}

func init() {
	compare.Register(func(c compare.Comparator, reference, value Error) {
		c.With(c.Path.Member("Message", reference, value)).Compare(reference.Message, value.Message)
	})
}

func (l *ErrorList) Add(r *Reader, at cst.Fragment, message string, args ...interface{}) {
	if len(*l) >= ParseErrorLimit {
		panic(AbortParse)
	}
	err := Error{At: at}
	if at == nil || at.Tok().Len() == 0 {
		if r != nil {
			err.At = r.GuessNextToken()
		}
	}
	if len(args) > 0 {
		err.Message = fmt.Sprintf(message, args...)
	} else {
		err.Message = message
	}
	var stack [1 << 16]byte
	size := runtime.Stack(stack[:], false)
	err.Stack = make([]byte, size)
	copy(err.Stack, stack[:size])
	*l = append(*l, err)
}
