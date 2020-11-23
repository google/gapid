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
package api_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/test"
)

func TestToServiceToCmd(t *testing.T) {
	ctx := log.Testing(t)
	for n, cmd := range map[string]api.Cmd{"A": test.Cmds.A, "B": test.Cmds.B} {
		s, err := api.CmdToService(cmd)
		if !assert.For(ctx, "CmdToService(%v)", n).ThatError(err).Succeeded() {
			continue
		}
		g, err := api.ServiceToCmd(s)
		if !assert.For(ctx, "ServiceToCmd(%v)", n).ThatError(err).Succeeded() {
			continue
		}
		assert.For(ctx, "CmdToService(%v) -> ServiceToCmd", n).That(g).DeepEquals(cmd)
	}
}
