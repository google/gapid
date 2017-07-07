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

package atom

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

// Writer is the interface that wraps the basic Write method.
//
// Write writes or processes the given atom and identifier. Write must not
// modify the atom in any way.
type Writer interface {
	Write(ctx context.Context, id ID, c api.Cmd)
}
