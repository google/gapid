// Copyright (C) 2020 Google Inc.
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

// Package controlFlowGenerator contains the interface and
// implementations to generate a control flow to create a
// replay.
package controlFlowGenerator

import "context"

// ControlFlowGenerator is an interface to generate a control flow
// that runs all the required transforms on commands. Implementers of
// this interface should accept required inputs to be able to
// transform commands and generate replay.
type ControlFlowGenerator interface {
	TransformAll(ctx context.Context) error
}
