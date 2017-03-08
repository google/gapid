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

package adb

import "context"

// Pushes the local file to the remote one.
func (b *binding) Push(ctx context.Context, local, remote string) error {
	return b.Command("push", local, remote).Run(ctx)
}

// Pulls the remote file to the local one.
func (b *binding) Pull(ctx context.Context, remote, local string) error {
	return b.Command("pull", remote, local).Run(ctx)
}
