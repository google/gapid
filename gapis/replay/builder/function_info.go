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

package builder

import "github.com/google/gapid/gapis/replay/protocol"

// FunctionInfo holds the information about a function that can be called by
// the replay virtual-machine.
type FunctionInfo struct {
	ApiIndex   uint8         // The index of the API this function belongs to.
	ID         uint16        // The unique identifier for the function.
	ReturnType protocol.Type // The returns type of the function.
	Parameters int           // The number of parameters for the function.
}
