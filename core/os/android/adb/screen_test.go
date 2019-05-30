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

package adb_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestScreenState(t_ *testing.T) {
	ctx := log.Testing(t_)

	d := mustConnect(ctx, "screen_off_locked_device")
	res, err := d.UnlockScreen(ctx)
	assert.For(ctx, "UnlockScreen").That(res).Equals(false)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	d = mustConnect(ctx, "screen_off_unlocked_device")
	res, err = d.UnlockScreen(ctx)
	assert.For(ctx, "UnlockScreen").That(res).Equals(false)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	d = mustConnect(ctx, "screen_on_locked_device")
	res, err = d.UnlockScreen(ctx)
	assert.For(ctx, "UnlockScreen").That(res).Equals(false)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	d = mustConnect(ctx, "screen_on_unlocked_device")
	res, err = d.UnlockScreen(ctx)
	assert.For(ctx, "UnlockScreen").That(res).Equals(true)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	d = mustConnect(ctx, "invalid_device")
	res, err = d.UnlockScreen(ctx)
	assert.For(ctx, "UnlockScreen").That(res).Equals(false)
	assert.For(ctx, "err").ThatError(err).Failed()
}
