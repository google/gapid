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

// Package script provides a fairly terse programming language for expressing
// search queries as strings.
// This is much more compact and easy to read than the full structured form.
// It's intended use is for places where you want a user to be able to type a query
// in a fairly natural language. Programmatic uses should prefer using the query
// package directly.
package script

// The following are the imports that generated source files pull in when present
// Having these here helps out tools that can't cope with missing dependancies
import (
	_ "context"
	_ "regexp"
	_ "strconv"

	_ "github.com/google/gapid/core/log"
	_ "github.com/google/gapid/test/robot/lingo"
	_ "github.com/google/gapid/test/robot/search/query"
)
