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

package reporting

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestFilterStack(t *testing.T) {
	ctx := log.Testing(t)
	stack := `⇒ core/app/crash/crash.go@74:Crash
⇒ core/app/crash/crash.go@56:handler
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/asm_amd64.s@514:
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/panic.go@489:
⇒ github.com/gapid/gapis/database/debug.go@106:(*memory).resolvePanicHandler
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/asm_amd64.s@514:
⇒ \usr\local\google\home\bobbobson\src\go-1.8.3\src\runtime\panic.go@489:
⇒ github.com\gapid\gapis\api\cmd_foreach.go@39:ForeachCmd.func1
⇒ \usr\local\google\home\bobbobson\src\go-1.8.3\src\runtime\asm_amd64.s@514:
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/panic.go@489:
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/alg.go@166:
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/alg.go@154:
⇒ /usr/local/google/home/bobbobson/src/go-1.8.3/src/runtime/hashmap.go@380:
⇒ github.com/gapid/gapis/resolve/dependencygraph/dependency_graph.go@111:(*AddressMapping).addressOf
⇒ github.com/gapid/gapis/resolve/dependencygraph/dependency_graph.go@123:(*CommandBehaviour).Read
⇒ github.com/gapid/gapis/api/vulkan/mutate.go@12431:(*VkQueueSubmit).GetCommandBehaviour
⇒ github.com/gapid/gapis/api/vulkan/replay.go@559:(*VulkanDependencyGraphBehaviourProvider).GetBehaviourForCommand
⇒ github.com/gapid/gapis/resolve/dependencygraph/dependency_graph.go@213:(*DependencyGraphResolvable).Resolve.func1.1
⇒ github.com/gapid/gapis/api/cmd_foreach.go@46:ForeachCmd
⇒ github.com/gapid/gapis/resolve/dependencygraph/dependency_graph.go@215:(*DependencyGraphResolvable).Resolve.func1
⇒ core/app/benchmark/counter.go@163:(*DurationCounter).Time
⇒ github.com/gapid/gapis/resolve/dependencygraph/dependency_graph.go@216:(*DependencyGraphResolvable).Resolve
⇒ github.com/gapid/gapis/database/memory.go@127:(*record).resolve
⇒ github.com/gapid/gapis/database/memory.go@214:(*memory).resolveLocked.func1
⇒ core/app/crash/crash.go@65:Go.func1`

	filtered := filterStack(stack)
	expected := `⇒ core/app/crash/crash.go@74:Crash
⇒ core/app/crash/crash.go@56:handler
⇒ <RUNTIME>/asm_amd64.s@514:
⇒ <RUNTIME>/panic.go@489:
⇒ <GAPID>/gapis/database/debug.go@106:(*memory).resolvePanicHandler
⇒ <RUNTIME>/asm_amd64.s@514:
⇒ <RUNTIME>\panic.go@489:
⇒ <GAPID>\gapis\api\cmd_foreach.go@39:ForeachCmd.func1
⇒ <RUNTIME>\asm_amd64.s@514:
⇒ <RUNTIME>/panic.go@489:
⇒ <RUNTIME>/alg.go@166:
⇒ <RUNTIME>/alg.go@154:
⇒ <RUNTIME>/hashmap.go@380:
⇒ <GAPID>/gapis/resolve/dependencygraph/dependency_graph.go@111:(*AddressMapping).addressOf
⇒ <GAPID>/gapis/resolve/dependencygraph/dependency_graph.go@123:(*CommandBehaviour).Read
⇒ <GAPID>/gapis/api/vulkan/mutate.go@12431:(*VkQueueSubmit).GetCommandBehaviour
⇒ <GAPID>/gapis/api/vulkan/replay.go@559:(*VulkanDependencyGraphBehaviourProvider).GetBehaviourForCommand
⇒ <GAPID>/gapis/resolve/dependencygraph/dependency_graph.go@213:(*DependencyGraphResolvable).Resolve.func1.1
⇒ <GAPID>/gapis/api/cmd_foreach.go@46:ForeachCmd
⇒ <GAPID>/gapis/resolve/dependencygraph/dependency_graph.go@215:(*DependencyGraphResolvable).Resolve.func1
⇒ core/app/benchmark/counter.go@163:(*DurationCounter).Time
⇒ <GAPID>/gapis/resolve/dependencygraph/dependency_graph.go@216:(*DependencyGraphResolvable).Resolve
⇒ <GAPID>/gapis/database/memory.go@127:(*record).resolve
⇒ <GAPID>/gapis/database/memory.go@214:(*memory).resolveLocked.func1
⇒ core/app/crash/crash.go@65:Go.func1`

	assert.For(ctx, "stack").ThatString(filtered).Equals(expected)
}
