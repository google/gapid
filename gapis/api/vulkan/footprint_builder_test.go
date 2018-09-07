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

package vulkan

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
)

func TestAddResBinding(t *testing.T) {
	ctx := log.Testing(t)
	_ = ctx
}

func TestSubBinding(t *testing.T) {
	ctx := log.Testing(t)
	resSize := uint64(2048)
	memOffset := uint64(1024)
	span := &memorySpan{
		sp:     interval.U64Span{Start: memOffset, End: memOffset + resSize},
		memory: VkDeviceMemory(0xabcd),
	}
	newSubBindingForTest := func(ctx context.Context, base *resBinding, offset, size uint64) *resBinding {
		r, _ := base.newSubBinding(ctx, nil, offset, size)
		return r
	}
	spanBase := newResBinding(ctx, nil, 0, resSize, span)
	labelBase := newResBinding(ctx, nil, 0, resSize, newLabel())

	invalidSubBoundData := func(offset, size uint64, base *resBinding) {
		assert.For(ctx, "Invalid range Offset: %v, size: %v on base: %v, expect return nil",
			offset, size, base).That(newSubBindingForTest(ctx, base, offset, size) == nil).Equals(true)
	}
	validSubBoundData := func(offset, size uint64, base, expected *resBinding) {
		assert.For(ctx, "Offset: %v, size: %v, on base: %v, expect valid subBoundData",
			offset, size, base).That(reflect.DeepEqual(*expected, *(newSubBindingForTest(ctx,
			base, offset, size)))).Equals(true)
	}

	invalidSubBoundData(0, 2047, labelBase)
	invalidSubBoundData(0, 2049, labelBase)
	invalidSubBoundData(1, 2048, labelBase)
	invalidSubBoundData(1, vkWholeSize, labelBase)
	validSubBoundData(0, 2048, labelBase, labelBase)
	validSubBoundData(0, vkWholeSize, labelBase, labelBase)

	invalidSubBoundData(2048, 1, spanBase)
	invalidSubBoundData(0xFFFFFFFF, 1, spanBase)
	invalidSubBoundData(2, vkWholeSize-1, spanBase)

	validSubBoundData(0, 2048, spanBase, spanBase)
	validSubBoundData(0, vkWholeSize, spanBase, spanBase)
	validSubBoundData(1024, 512, spanBase, newResBinding(ctx, nil, 1024, 512, &memorySpan{
		sp: interval.U64Span{
			Start: memOffset + uint64(1024),
			End:   memOffset + uint64(1024) + uint64(512),
		},
		memory: VkDeviceMemory(0xabcd),
	}))
	validSubBoundData(512, vkWholeSize, spanBase, newResBinding(ctx, nil, 512, 1536, &memorySpan{
		sp: interval.U64Span{
			Start: memOffset + uint64(512),
			End:   memOffset + resSize,
		},
		memory: VkDeviceMemory(0xabcd),
	}))
}
