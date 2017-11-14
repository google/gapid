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

import "github.com/google/gapid/core/math/interval"

func (b *VkSparseMemoryBind) span() interval.U64Span {
	return interval.U64Span{Start: uint64(b.ResourceOffset), End: uint64(b.ResourceOffset) + uint64(b.Size)}
}

func (b *VkSparseMemoryBind) size() uint64 {
	return uint64(b.Size)
}

func (b *VkSparseMemoryBind) shrink(offset, size uint64) error {
	if offset+size < offset || offset+size > uint64(b.Size) {
		return shrinkOutOfMemBindingBound{b, offset, size}
	}
	b.Size = VkDeviceSize(size)
	b.MemoryOffset += VkDeviceSize(offset)
	b.ResourceOffset += VkDeviceSize(offset)
	return nil
}

func (b *VkSparseMemoryBind) duplicate() memBinding {
	newB := *b
	return &newB
}

type sparseBindingList memBindingList

func addSparseBinding(l sparseBindingList, b *VkSparseMemoryBind) (sparseBindingList, error) {
	ol := memBindingList(l)
	nl, err := addBinding(ol, b)
	if err != nil {
		return nil, err
	}
	return sparseBindingList(nl), nil
}

func addSparseImageBinding(bs U32ːVkSparseImageMemoryBindᵐ, b VkSparseImageMemoryBind) {
	last := uint32(0)
	for i, eb := range bs.Range() {
		if fullyCover(eb, b) {
			bs.Delete(i)
		}
		if i > last {
			last = i
		}
	}
	bs.Set(last+1, b)
}

func fullyCover(orig, new VkSparseImageMemoryBind) bool {
	origAspect := orig.Subresource.AspectMask
	newAspect := new.Subresource.AspectMask
	if newAspect&origAspect != origAspect {
		return false
	}
	if orig.Subresource.MipLevel != new.Subresource.MipLevel {
		return false
	}
	if orig.Subresource.ArrayLayer != new.Subresource.ArrayLayer {
		return false
	}
	if new.Offset.X > orig.Offset.X || new.Offset.Y > orig.Offset.Y || new.Offset.Z > orig.Offset.Z {
		return false
	}
	if new.Offset.X+int32(new.Extent.Width) < orig.Offset.X+int32(orig.Extent.Width) ||
		new.Offset.Y+int32(new.Extent.Height) < orig.Offset.Y+int32(orig.Extent.Height) ||
		new.Offset.Z+int32(new.Extent.Depth) < orig.Offset.Z+int32(orig.Extent.Depth) {
		return false
	}
	return true
}
