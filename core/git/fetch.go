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

package git

import (
	"context"
	"fmt"
)

// Fetch performs a `git fetch` call.
// The number of new CLs locally and remotely for the current branch is returned.
func (g Git) Fetch(ctx context.Context) (localNew, remoteNew int, err error) {
	if _, _, err := g.run(ctx, "fetch"); err != nil {
		return -1, -1, err
	}
	str, _, err := g.run(ctx, "rev-list", "--count", "--left-right", Head+"..."+FetchHead)
	if err != nil {
		return -1, -1, err
	}
	fmt.Sscanf(str, "%d %d", &localNew, &remoteNew)
	return localNew, remoteNew, nil
}
