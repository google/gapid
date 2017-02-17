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

import "fmt"

// *************************************************************
// ** This file contains legacy shims only, it should be removed
// ** once all existing tests are converted
// *************************************************************

const (
	expected  = "Got:      %+v¶Expected: %+v"
	unmatched = "Got: %+v¶%s: %+v"
)

// Deprecated: Context is deprecated, To instead.
func Context(out Output) Manager {
	return To(out)
}

// Deprecated: For(m) is deprecated, use m.For() instead.
func For(t interface{}, msg string, args ...interface{}) *Assertion {
	type valueSource interface {
		Value(interface{}) interface{}
	}
	switch t := t.(type) {
	case Manager:
		return t.For(msg, args...)
	case valueSource:
		out, _ := t.Value("TestingOutput").(Output)
		return To(out).For(msg, args...)
	default:
		panic("Not a valid assertion manager source")
	}
}

// Deprecated: With is deprecated, use For with a message.
func With(t interface{}) *Assertion {
	return For(t, "")
}

// Deprecated: Print is deprecated, use Log instead.
func (m Manager) Print(args ...interface{}) {
	a := m.For(fmt.Sprint(args...))
	a.level = Log
	a.Commit()
}
