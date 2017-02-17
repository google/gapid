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

// Package parse provides support functionality for writing scannerless parsers.
//
// The main entry point is the Parse function.
// It works by setting up a parser on the supplied content, and then invoking the
// supplied root parsing function. The CST is built for you inside the ParseLeaf
// and ParseBranch methods of the parser, but it is up to the supplied parsing
// functions to hold on to the CST if you want it, and also to build the AST.
package parse
