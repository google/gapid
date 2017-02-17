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

package annotate

// Glossary:
//
// Snippet  - this is static information collected from the semantic tree
// Location - this is a place which snippets will be associated with
// Nested   - a snippet with hierarchical composition
//
// container - a Nested for container objects: map, array, slice, pointer
// entity    - a Nested for field objects: class
//
// SymbolCategory    - Global, Local, Parameter
// SymbolTable       - maps names in a particular category to minions
// ScopedSymbolTable - allows scoping with variable hiding
// SymbolSpace       - maps category to ScopedSymbolTable

// Approach:
//
// Traverse the semantic tree of the API program. Generate snippets for
// expressions as required for the particular analysis. Let assignment
// statements and comparison expressions create information flow. When
// information flows from a snippet to a location, the snippet is attached
// to the location. Generate nested structures automatically as required
// for snippets to be attached at the appropriate location.
//
// When information flows from a location to a location the two locations
// become aliases and snippets are merged. Aliasing is achieved by picking
// a leader and a minion. All minions have a parent which is either a
// minion or a leader. A leader does not have a parent. All mutations
// operations on a minion are forwarded to its leader. Whenever the leader
// of a minion is computed, the path to the leader is shortened for all
// minions in the chain. This approach called union-find gives complexity
// of O(A^-1(n,n)) ~ the inverse Ackermann function, which is typically
// regarded as being as good as constant time:
//
// https://en.wikipedia.org/wiki/Disjoint-set_data_structure
