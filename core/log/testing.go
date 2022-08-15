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

package log

import "context"

// Testing returns a default context with a TestHandler installed.
func Testing(t delegate) context.Context {
	return SubTest(context.Background(), t)
}

// SubTest returns the context with the TestHandler replaced with t.
// This is intended to be used for sub-tests. For example:
//
//	func TestExample(t *testing.T) {
//	  ctx := log.Testing(t)
//	  for _, test := range tests {
//	    t.Run(test.name, func(t *testing.T) {
//	      test.run(log.SubTest(ctx, t))
//	    }
//	  }
//	}
func SubTest(ctx context.Context, t delegate) context.Context {
	return PutHandler(ctx, TestHandler(t, Normal))
}

// TestHandler is a Writer that uses the style to write records to t's using the
// style s.
func TestHandler(t delegate, s Style) Handler {
	if t == nil {
		panic("delegate cannot be nil")
	}
	return handler{
		handle: func(m *Message) {
			switch {
			case m.Severity >= Fatal:
				t.Fatal(s.Print(m))
			case m.Severity >= Error:
				t.Error(s.Print(m))
			default:
				t.Log(s.Print(m))
			}
		},
		close: func() {},
	}
}

// delegate matches the logging methods of the test host types.
type delegate interface {
	Fatal(...interface{})
	Error(...interface{})
	Log(...interface{})
}
