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

package assert

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/google/gapid/core/data/compare"
)

type (
	// Level is used to control what output level is used when flushing assertion text.
	level int

	// Assertion is the type for the start of an assertion line.
	// You construct an assertion from an Output using assert.For.
	Assertion struct {
		level level
		out   *bytes.Buffer
		to    Output
	}
)

const (
	// Log is the informational level.
	Log = level(iota)
	// Error is used for things that cause test failures but do not abort.
	Error
	// Fatal is used for failures that cause the running test to immediately stop.
	Fatal
)

func (l level) String() string {
	switch l {
	case Log:
		return "Info"
	case Error:
		return "Error"
	case Fatal:
		return "Critical"
	default:
		return "Unknown"
	}
}

// Critical switches this assertion from Error to Fatal.
func (a *Assertion) Critical() *Assertion {
	a.level = Fatal
	return a
}

// Log appends the supplied message to the cached output, and then flushes to the underlying output at Log level.
func (a *Assertion) Log(args ...interface{}) {
	fmt.Fprint(a.out, args...)
	a.level = Log
	a.Commit()
}

// Error appends the supplied message to the cached output, and then flushes to the underlying output at Error level.
func (a *Assertion) Error(args ...interface{}) {
	fmt.Fprint(a.out, args...)
	a.level = Error
	a.Commit()
}

// Fatal appends the supplied message to the cached output, and then flushes to the underlying output at Fatal level.
func (a *Assertion) Fatal(args ...interface{}) {
	fmt.Fprint(a.out, args...)
	a.level = Fatal
	a.Commit()
}

// PrintPretty writes a value to the output buffer.
// It performs standardised transformations (mostly quoting)
func (a Assertion) PrintPretty(value interface{}) {
	switch value := value.(type) {
	case error:
		a.out.WriteRune('`')
		fmt.Fprint(a.out, value)
		a.out.WriteRune('`')
	case string:
		a.out.WriteRune('`')
		a.out.WriteString(value)
		a.out.WriteRune('`')
	default:
		fmt.Fprint(a.out, value)
	}
}

// Print writes a set of values to the output buffer, joined by tabs.
// The values will be printed with PrintValue.
func (a *Assertion) Print(args ...interface{}) *Assertion {
	if len(args) == 0 {
		return a
	}
	for i, v := range args {
		if i != 0 {
			a.out.WriteString("\t")
		}
		a.PrintPretty(v)
	}
	return a
}

// Raw writes a set of values to the output buffer, joined by tabs.
// It does not use the pretty printer.
func (a *Assertion) Raw(args ...interface{}) *Assertion {
	if len(args) == 0 {
		return a
	}
	for i, v := range args {
		if i != 0 {
			a.out.WriteString("\t")
		}
		fmt.Fprint(a.out, v)
	}
	return a
}

// Println prints the values using Print and then starts a new indented line.
func (a *Assertion) Println(args ...interface{}) *Assertion {
	a.Print(args...)
	a.out.WriteString("\n    ")
	return a
}

// Println prints the values using Print and then starts a new indented line.
func (a *Assertion) Rawln(args ...interface{}) *Assertion {
	a.Raw(args...)
	a.out.WriteString("\n    ")
	return a
}

// Printf writes a formatted unquoted string to the output buffer.
func (a *Assertion) Printf(format string, args ...interface{}) *Assertion {
	fmt.Fprintf(a.out, format, args...)
	return a
}

// Add appends a key value pair to the output buffer.
func (a *Assertion) Add(key string, values ...interface{}) *Assertion {
	a.out.WriteString(key)
	a.out.WriteString("\t\t")
	a.Println(values...)
	return a
}

// Got adds the standard "Got" entry to the output buffer.
func (a *Assertion) Got(values ...interface{}) *Assertion {
	a.out.WriteString("Got\t\t")
	a.Println(values...)
	return a
}

// Expect adds the standard "Expect" entry to the output buffer.
func (a *Assertion) Expect(op string, values ...interface{}) *Assertion {
	a.out.WriteString("Expect\t")
	a.out.WriteString(op)
	a.out.WriteString("\t")
	a.Println(values...)
	return a
}

// ExpectRaw adds the standard "Expect" entry to the output buffer, without pretty printing.
func (a *Assertion) ExpectRaw(op string, values ...interface{}) *Assertion {
	a.out.WriteString("Expect\t")
	a.out.WriteString(op)
	a.out.WriteString("\t")
	a.Rawln(values...)
	return a
}

// Compare adds both the "Got" and "Expect" entries to the output buffer, with the operator being
// prepended to the expect list.
func (a *Assertion) Compare(value interface{}, op string, expect ...interface{}) *Assertion {
	return a.Got(value).Expect(op, expect...)
}

// CompareRaw is like Compare except it does not push the values through the pretty printer.
func (a *Assertion) CompareRaw(value interface{}, op string, expect ...interface{}) *Assertion {
	return a.Got(value).ExpectRaw(op, expect...)
}

// Test commits the pending output if the condition is not true.
func (a *Assertion) Test(condition bool) bool {
	if !condition {
		if a.level <= Error {
			a.level = Error
		}
		a.Commit()
	}
	return condition
}

// TestDeepEqual adds the entries for Got and Expect, then tests if they are the
// same using compare.DeepEqual, commiting if they are not.
func (a *Assertion) TestDeepEqual(value, expect interface{}) bool {
	return a.Compare(value, "deep ==", expect).Test(compare.DeepEqual(value, expect))
}

// TestCustomDeepEqual adds the entries for Got and Expect, then tests if they
// are the same using c.DeepEqual, commiting if they are not.
func (a *Assertion) TestCustomDeepEqual(value, expect interface{}, c compare.Custom) bool {
	return a.Compare(value, "deep ==", expect).Test(c.DeepEqual(value, expect))
}

// TestDeepNotEqual adds the entries for Got and Expect, then tests if they are
// the same using compare.DeepEqual, commiting if they are.
func (a *Assertion) TestDeepNotEqual(value, expect interface{}) bool {
	return a.Compare(value, "deep !=", expect).Test(!compare.DeepEqual(value, expect))
}

// TestCustomDeepNotEqual adds the entries for Got and Expect, then tests if
// they are the same using c.DeepEqual, commiting if they are.
func (a *Assertion) TestCustomDeepNotEqual(value, expect interface{}, c compare.Custom) bool {
	return a.Compare(value, "deep !=", expect).Test(!c.DeepEqual(value, expect))
}

// TestDeepDiff is equivalent to TestDeepEqual except it also prints a diff.
func (a *Assertion) TestDeepDiff(value, expect interface{}) bool {
	diff := compare.Diff(value, expect, 10)
	if len(diff) == 0 {
		return true
	}
	for _, diff := range diff {
		a.Println(diff)
	}
	a.Commit()
	return false
}

// TestCustomDeepDiff is equivalent to TestCustomDeepEqual except it also prints
//
//	a diff.
func (a *Assertion) TestCustomDeepDiff(value, expect interface{}, c compare.Custom) bool {
	diff := c.Diff(value, expect, 10)
	if len(diff) == 0 {
		return true
	}
	for _, diff := range diff {
		a.Println(diff)
	}
	a.Commit()
	return false
}

// Commit writes the output lines to the main output object.
func (a Assertion) Commit() {
	// push the output buffer through a tabwriter to align columns
	buf := &bytes.Buffer{}
	tabs := tabwriter.NewWriter(buf, 1, 4, 1, ' ', tabwriter.StripEscape)
	tabs.Write(a.out.Bytes())
	tabs.Flush()
	message := a.level.String() + ":" + strings.TrimRightFunc(buf.String(), unicode.IsSpace)
	switch a.level {
	case Log:
		a.to.Log(message)
	case Error:
		a.to.Error(message)
	case Fatal:
		a.to.Fatal(message)
	default:
		a.to.Log(message)
	}
}
