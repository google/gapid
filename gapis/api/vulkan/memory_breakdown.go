// Copyright (C) 2018 Google Inc.
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
	"fmt"
	"strconv"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service/path"
)

// Interface compliance test
var (
	_ = api.MemoryBreakdownProvider(API{})
)

type sparseBindingMap map[VkDeviceMemory][]*api.MemoryBinding

// Implements api.MemoryBreakdownProvider
func (a API) MemoryBreakdown(st *api.GlobalState) (*api.MemoryBreakdown, error) {
	s := GetState(st)

	// Iterate over images and buffers looking for sparsely-bound objects.
	// They are not listed on the VkDeviceMemoryObject and so we need to
	// collect them beforehand.
	sparseBindings := sparseBindingMap{}
	for _, info := range s.Buffers().All() {
		if (info.Info().CreateFlags() & VkBufferCreateFlags(
			VkBufferCreateFlagBits_VK_BUFFER_CREATE_SPARSE_BINDING_BIT)) ==
			VkBufferCreateFlags(0) {

			continue
		}
		if err := sparseBindings.getBufferSparseBindings(info); err != nil {
			return nil, err
		}
	}
	for _, info := range s.Images().All() {
		if (info.Info().Flags() & VkImageCreateFlags(
			VkImageCreateFlagBits_VK_IMAGE_CREATE_SPARSE_BINDING_BIT)) ==
			VkImageCreateFlags(0) {

			continue
		}
		if err := sparseBindings.getImageSparseBindings(info); err != nil {
			return nil, err
		}
	}
	allocations := make([]*api.MemoryAllocation, 0, len(s.DeviceMemories().All()))
	// Serialize data on all allocations into protobufs
	for handle, info := range s.DeviceMemories().All() {
		device := info.Device()
		typ := info.MemoryTypeIndex()
		flags, err := s.getMemoryTypeFlags(device, typ)
		if err != nil {
			return nil, err
		}
		bindings, err := s.getAllocationBindings(info.Get(), st)
		if err != nil {
			return nil, err
		}

		if binds, ok := sparseBindings[handle]; ok {
			bindings = append(bindings, binds...)
		}

		mapping := api.MemoryMapping{
			Size:          uint64(info.MappedSize()),
			Offset:        uint64(info.MappedOffset()),
			MappedAddress: uint64(info.MappedLocation()),
		}

		alloc := api.MemoryAllocation{
			Device:     uint64(info.Device()),
			MemoryType: uint32(typ),
			Flags:      uint32(flags),
			Handle:     uint64(handle),
			Name:       strconv.FormatUint(uint64(handle), 10),
			Size:       uint64(info.AllocationSize()),
			Mapping:    &mapping,
			Bindings:   bindings,
		}

		allocations = append(allocations, &alloc)
	}
	return &api.MemoryBreakdown{
		API:                  path.NewAPI(id.ID(ID)),
		Allocations:          allocations,
		AllocationFlagsIndex: int32(VkMemoryPropertyFlagBitsConstants()),
	}, nil
}

func (s sparseBindingMap) getBufferSparseBindings(info BufferObjectʳ) error {
	handle := uint64(info.VulkanHandle())
	for _, bind := range info.SparseMemoryBindings().All() {
		if bind.Memory() == VkDeviceMemory(0) {
			continue
		}

		binding := api.MemoryBinding{
			Handle: handle,
			Name:   strconv.FormatUint(handle, 10),
			Size:   uint64(bind.Size()),
			Offset: uint64(bind.MemoryOffset()),
			Type: &api.MemoryBinding_SparseBufferBlock{
				&api.SparseBinding{
					Offset: uint64(bind.ResourceOffset()),
				},
			},
		}

		v, ok := s[bind.Memory()]
		if !ok {
			v = []*api.MemoryBinding{}
		}
		s[bind.Memory()] = append(v, &binding)
	}

	return nil
}

func getAspects(aspects uint32) []api.AspectType {
	aspectTypes := []api.AspectType{}
	if aspects&uint32(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT) != 0 {
		aspectTypes = append(aspectTypes, api.AspectType_COLOR)
	}
	if aspects&uint32(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) != 0 {
		aspectTypes = append(aspectTypes, api.AspectType_DEPTH)
	}
	if aspects&uint32(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) != 0 {
		aspectTypes = append(aspectTypes, api.AspectType_STENCIL)
	}
	return aspectTypes
}

func (s sparseBindingMap) getImageSparseBindings(img ImageObjectʳ) error {
	opaque := (img.Info().Flags() & VkImageCreateFlags(
		VkImageCreateFlagBits_VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT)) ==
		VkImageCreateFlags(0)
	handle := uint64(img.VulkanHandle())
	name := strconv.FormatUint(handle, 10)
	for _, bind := range img.OpaqueSparseMemoryBindings().All() {
		if bind.Memory() == VkDeviceMemory(0) {
			continue
		}

		binding := api.MemoryBinding{
			Handle: handle,
			Name:   name,
			Size:   uint64(bind.Size()),
			Offset: uint64(bind.MemoryOffset()),
		}
		if opaque {
			binding.Type = &api.MemoryBinding_SparseOpaqueImageBlock{
				&api.SparseBinding{
					Offset: uint64(bind.ResourceOffset()),
				},
			}
		} else {
			// Need to determine which type it is
			// Determine if it's in a mip tail
			checkMipTail := func(reqs VkSparseImageMemoryRequirements) (uint64, uint32, bool) {
				offset := bind.ResourceOffset()
				if offset < reqs.ImageMipTailOffset() {
					return 0, 0, false
				}
				offset -= reqs.ImageMipTailOffset()
				arrayLayer := uint32(0)
				if reqs.FormatProperties().Flags()&(VkSparseImageFormatFlags(
					VkSparseImageFormatFlagBits_VK_SPARSE_IMAGE_FORMAT_SINGLE_MIPTAIL_BIT)) ==
					VkSparseImageFormatFlags(0) {

					arrayLayer = uint32(offset / reqs.ImageMipTailStride())
					offset %= reqs.ImageMipTailStride()
				}
				if offset >= reqs.ImageMipTailSize() || arrayLayer >= img.Info().ArrayLayers() {
					return 0, 0, false
				}
				return uint64(offset), arrayLayer, true
			}
			if (bind.Flags() & VkSparseMemoryBindFlags(
				VkSparseMemoryBindFlagBits_VK_SPARSE_MEMORY_BIND_METADATA_BIT)) !=
				VkSparseMemoryBindFlags(0) {

				reqs, ok := img.SparseMemoryRequirements().Lookup(
					VkImageAspectFlagBits_VK_IMAGE_ASPECT_METADATA_BIT)
				if !ok {
					return fmt.Errorf("Metadata binding present but no metadata sparse memory requirements for image %v", handle)
				}

				offset, arrayLayer, ok := checkMipTail(reqs)

				if !ok {
					return fmt.Errorf("Binding has metadata flag setbut isn't in metadata mip tail")
				}
				binding.Type = &api.MemoryBinding_SparseImageMetadata{
					&api.SparseImageMetadataMipTail{
						ArrayLayer: arrayLayer,
						Offset:     offset,
					},
				}
			} else {
				inMip := false
				for aspects, reqs := range img.SparseMemoryRequirements().All() {
					offset, arrayLayer, ok := checkMipTail(reqs)
					if !ok {
						continue
					}
					inMip = true
					binding.Type = &api.MemoryBinding_SparseImageMipTail{
						&api.SparseImageMetadataMipTail{
							ArrayLayer: arrayLayer,
							Offset:     offset,
							Aspects:    getAspects(uint32(aspects)),
						},
					}
					break
				}
				if !inMip {
					binding.Type = &api.MemoryBinding_SparseOpaqueImageBlock{
						&api.SparseBinding{
							Offset: uint64(bind.ResourceOffset()),
						},
					}
				}
			}
		}
		v, ok := s[bind.Memory()]
		if !ok {
			v = []*api.MemoryBinding{}
		}
		s[bind.Memory()] = append(v, &binding)
	}
	for aspects, layers := range img.SparseImageMemoryBindings().All() {
		aspectTypes := getAspects(uint32(aspects))
		for layer, levels := range layers.Layers().All() {
			for level, blocks := range levels.Levels().All() {
				for _, block := range blocks.Blocks().All() {
					if block.Memory() == VkDeviceMemory(0) {
						continue
					}
					binding := api.MemoryBinding{
						Handle: handle,
						Name:   name,
						Size:   uint64(block.Size()),
						Offset: uint64(block.MemoryOffset()),
						Type: &api.MemoryBinding_SparseImageBlock{
							&api.SparseImageBlock{
								XOffset: block.Offset().X(),
								YOffset: block.Offset().Y(),
								Width:   block.Extent().Width(),
								Height:  block.Extent().Height(),

								MipLevel:   level,
								ArrayLayer: layer,
								Aspects:    aspectTypes,
							},
						},
					}
					v, ok := s[block.Memory()]
					if !ok {
						v = []*api.MemoryBinding{}
					}
					s[block.Memory()] = append(v, &binding)
				}
			}
		}
	}
	return nil
}

func (s *State) getMemoryTypeFlags(device VkDevice, typeIndex uint32) (VkMemoryPropertyFlags, error) {
	deviceObject := s.Devices().Get(device)
	if deviceObject.IsNil() {
		return VkMemoryPropertyFlags(0), fmt.Errorf("Failed to find device %v", device)
	}
	physicalDevice := deviceObject.PhysicalDevice()
	physicalDeviceObject := s.PhysicalDevices().Get(physicalDevice)
	if physicalDeviceObject.IsNil() {
		return VkMemoryPropertyFlags(0), fmt.Errorf("Failed to find physical device %v", physicalDevice)
	}
	props := physicalDeviceObject.MemoryProperties()
	if props.MemoryTypeCount() <= typeIndex {
		return VkMemoryPropertyFlags(0), fmt.Errorf("Memory type %v is larger than physical device %v's number of memory types (%v)",
			typeIndex, physicalDevice, props.MemoryTypeCount())
	}
	return props.MemoryTypes().Get(int(typeIndex)).PropertyFlags(), nil
}

func (s *State) getAllocationBindings(allocation DeviceMemoryObject, st *api.GlobalState) ([]*api.MemoryBinding, error) {
	bindings := []*api.MemoryBinding{}
	for handle, offset := range allocation.BoundObjects().All() {
		binding := api.MemoryBinding{
			Handle: uint64(handle),
			Name:   strconv.FormatUint(handle, 10),
			Offset: uint64(offset),
		}
		if buffer, ok := s.Buffers().Lookup(VkBuffer(handle)); ok {
			binding.Size = uint64(buffer.Info().Size())
			binding.Type = &api.MemoryBinding_Buffer{&api.NormalBinding{}}
		} else if image, ok := s.Images().Lookup(VkImage(handle)); ok {
			ctx := context.Background()
			memInfo, _ := subGetImagePlaneMemoryInfo(ctx, nil, api.CmdNoID, nil, st, s, 0, nil, nil, image, VkImageAspectFlagBits(0))
			memRequirement := memInfo.MemoryRequirements()
			binding.Size = uint64(memRequirement.Size())
			binding.Type = &api.MemoryBinding_Image{&api.NormalBinding{}}
		} else {
			return nil, fmt.Errorf("Bound object %v is not a buffer or an image", handle)
		}

		bindings = append(bindings, &binding)
	}
	return bindings, nil
}
