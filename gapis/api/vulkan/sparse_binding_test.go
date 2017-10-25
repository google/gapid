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
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestAddBinding(t *testing.T) {
	checkRecordedBindings := func(incoming []VkSparseMemoryBind, expected sparseBindingList) {
		l := sparseBindingList{}
		for _, i := range incoming {
			l = addBinding(l, i)
		}
		assert.To(t).For("Expected recorded bindings: %v\nActual recorded bindings: %v", expected, l).That(
			reflect.DeepEqual(expected, l)).Equals(true)
	}

	newBinding := func(offset, size, mem, memoffset uint64) VkSparseMemoryBind {
		return VkSparseMemoryBind{
			ResourceOffset: VkDeviceSize(offset),
			Size:           VkDeviceSize(size),
			Memory:         VkDeviceMemory(mem),
			MemoryOffset:   VkDeviceSize(memoffset),
		}
	}

	// empty
	checkRecordedBindings([]VkSparseMemoryBind{}, sparseBindingList{})

	// empty bindings
	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 0, 0, 0),
	}, sparseBindingList{
		newBinding(0, 0, 0, 0),
	})

	// no-empty bindings
	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 512, 0, 10),
	}, sparseBindingList{
		newBinding(0, 512, 0, 10),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(768, 1024*1024, 0xffffffff11223344, 1024*10),
	}, sparseBindingList{
		newBinding(768, 1024*1024, 0xffffffff11223344, 1024*10),
	})

	// order
	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(1024*5, 1024, 0xf, 100),
		newBinding(1024*4, 1024, 0xe, 100),
		newBinding(1024*3, 1024, 0xd, 100),
		newBinding(1024*2, 1024, 0xc, 100),
		newBinding(1024, 1024, 0xb, 100),
		newBinding(0, 1024, 0xa, 100),
	}, sparseBindingList{
		newBinding(0, 1024, 0xa, 100),
		newBinding(1024, 1024, 0xb, 100),
		newBinding(1024*2, 1024, 0xc, 100),
		newBinding(1024*3, 1024, 0xd, 100),
		newBinding(1024*4, 1024, 0xe, 100),
		newBinding(1024*5, 1024, 0xf, 100),
	})

	// conflict with existing bindings
	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(1024, 1024, 0xa, 100),
		newBinding(512, 1024, 0xb, 100),
	}, sparseBindingList{
		newBinding(512, 1024, 0xb, 100),
		newBinding(1024+512, 512, 0xa, 100+512),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 1024, 0xa, 100),
		newBinding(512, 1024, 0xb, 100),
	}, sparseBindingList{
		newBinding(0, 512, 0xa, 100),
		newBinding(512, 1024, 0xb, 100),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 2048, 0xa, 100),
		newBinding(512, 1024, 0xb, 100),
	}, sparseBindingList{
		newBinding(0, 512, 0xa, 100),
		newBinding(512, 1024, 0xb, 100),
		newBinding(1024+512, 512, 0xa, 100+1024+512),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(512, 1024, 0xa, 100),
		newBinding(0, 2048, 0xb, 100),
	}, sparseBindingList{
		newBinding(0, 2048, 0xb, 100),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 1000, 0xa, 100),
		newBinding(100, 1000, 0xb, 100),
		newBinding(200, 1000, 0xc, 100),
		newBinding(500, 500, 0xd, 100),
		newBinding(600, 100, 0xe, 100),
		newBinding(300, 700, 0xf, 100),
	}, sparseBindingList{
		newBinding(0, 100, 0xa, 100),
		newBinding(100, 100, 0xb, 100),
		newBinding(200, 100, 0xc, 100),
		newBinding(300, 700, 0xf, 100),
		newBinding(1000, 200, 0xc, 900),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 1000, 0xa, 100),
		newBinding(100, 1000, 0xb, 100),
		newBinding(200, 1000, 0xc, 100),
		newBinding(500, 500, 0xd, 100),
		newBinding(600, 100, 0xe, 100),
		newBinding(300, 700, 0xf, 100),
		newBinding(500, 100, 0xa, 200),
	}, sparseBindingList{
		newBinding(0, 100, 0xa, 100),
		newBinding(100, 100, 0xb, 100),
		newBinding(200, 100, 0xc, 100),
		newBinding(300, 200, 0xf, 100),
		newBinding(500, 100, 0xa, 200),
		newBinding(600, 400, 0xf, 400),
		newBinding(1000, 200, 0xc, 900),
	})

	checkRecordedBindings([]VkSparseMemoryBind{
		newBinding(0, 1000, 0xa, 100),
		newBinding(100, 1000, 0xb, 100),
		newBinding(200, 1000, 0xc, 100),
		newBinding(500, 500, 0xd, 100),
		newBinding(600, 100, 0xe, 100),
		newBinding(300, 700, 0xf, 100),
		newBinding(500, 100, 0xa, 200),
		newBinding(0, 2000, 0xb, 500),
	}, sparseBindingList{
		newBinding(0, 2000, 0xb, 500),
	})
}
