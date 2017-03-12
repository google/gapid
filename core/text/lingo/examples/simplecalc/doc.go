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

// The simplecalc command implements a very simple calculator.
// This is an example of using the lingo system to parse and evaluate in a single pass.
// No AST or CST is ever built, the return of parsing is the calculated result.
// This pattern is fairly unusual, only the simplest of parsers where speed and
// simplicity matters would follow this approach.

package main

// The following are the imports that generated source files pull in when present
// Having these here helps out tools that can't cope with missing dependancies
import (
	_ "github.com/google/gapid/core/app"
	_ "github.com/google/gapid/core/fault/cause"
	_ "github.com/google/gapid/core/log"
	_ "github.com/google/gapid/core/text/lingo"
)
