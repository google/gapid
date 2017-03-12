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

// The cstcalc command implements a simple calculator.
// This is an example of using the lingo system to parse, ignoring the
// parsed nodes, and then directly evaluating the cst.
// This example is more about showing how to use the cst, it would be very
// unusual to actually use the parser in this way.

package main

// The following are the imports that generated source files pull in when present
// Having these here helps out tools that can't cope with missing dependancies
import (
	_ "github.com/google/gapid/core/app"
	_ "github.com/google/gapid/core/fault/cause"
	_ "github.com/google/gapid/core/log"
	_ "github.com/google/gapid/core/text/lingo"
)
