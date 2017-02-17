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

package test

import "github.com/google/gapid/framework/binary"

type X_V1 struct {
	binary.Frozen `name:"X"`
	A             int32
	B             int32
}

type X struct {
	binary.Generate `java:"disable"`
	A               int32
	B               int32
	C               string
}

func (before *X_V1) upgrade(after *X) {
	after.A = before.A
	after.B = before.B
	after.C = "Hello"
}

type Y struct {
	binary.Generate `java:"disable"`
	Begin           string
	X               X
	End             string
}
