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
	"bytes"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/shadertools"
)

func (t *ImageObject) IsResource() bool {
	// Since there are no good differentiating features for what is a "texture" and what
	// image is used for other things, we treat only images that can be used as SAMPLED
	// or STORAGE as resources. This may change in the future when we start doing
	// replays to get back gpu-generated image data.
	is_texture := 0 != (uint32(t.Info.Usage) & uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT|
		VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT))
	return t.VulkanHandle != 0 && is_texture
}

// ResourceName returns the UI name for the resource.
func (t *ImageObject) ResourceName() string {
	return fmt.Sprintf("Image<%d>", t.VulkanHandle)
}

// ResourceType returns the type of this resource.
func (t *ImageObject) ResourceType() gfxapi.ResourceType {
	if uint32(t.Info.Flags)&uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT) != 0 {
		return gfxapi.ResourceType_CubemapResource
	} else {
		return gfxapi.ResourceType_Texture2DResource
	}
}

type unsupportedVulkanFormatError struct {
	Format VkFormat
}

func (e *unsupportedVulkanFormatError) Error() string {
	return fmt.Sprintf("Unsupported Vulkan format: %d", e.Format)
}

func getImageFormatFromVulkanFormat(vkfmt VkFormat) (*image.Format, error) {
	switch vkfmt {
	case VkFormat_VK_FORMAT_R8G8B8A8_UNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_UNORM", fmts.RGBA_U8_NORM), nil
	case VkFormat_VK_FORMAT_BC1_RGB_SRGB_BLOCK:
		return image.NewS3_DXT1_RGB("VK_FORMAT_BC1_RGB_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC1_RGB_UNORM_BLOCK:
		return image.NewS3_DXT1_RGB("VK_FORMAT_BC1_RGB_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC1_RGBA_SRGB_BLOCK:
		return image.NewS3_DXT1_RGBA("VK_FORMAT_BC1_RGBA_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC1_RGBA_UNORM_BLOCK:
		return image.NewS3_DXT1_RGBA("VK_FORMAT_BC1_RGBA_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC2_UNORM_BLOCK:
		return image.NewS3_DXT3_RGBA("VK_FORMAT_BC2_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC3_UNORM_BLOCK:
		return image.NewS3_DXT5_RGBA("VK_FORMAT_BC3_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_SFLOAT", fmts.RGBA_F16), nil
	case VkFormat_VK_FORMAT_R8_UNORM:
		return image.NewUncompressed("VK_FORMAT_R8_UNORM", fmts.R_U8_NORM), nil
	case VkFormat_VK_FORMAT_R16_UNORM:
		return image.NewUncompressed("VK_FORMAT_R16_UNORM", fmts.R_U16_NORM), nil
	case VkFormat_VK_FORMAT_R16_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R16_SFLOAT", fmts.R_F16), nil
	case VkFormat_VK_FORMAT_R32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R32_SFLOAT", fmts.R_F32), nil
	case VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32A32_SFLOAT", fmts.RGBA_F32), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_UNORM:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_UNORM", fmts.BGRA_U8_NORM), nil
	case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT_S8_UINT", fmts.DS_F32U8), nil
	case VkFormat_VK_FORMAT_D32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT", fmts.D_F32), nil
	case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D24_UNORM_S8_UINT", fmts.DS_NU24U8), nil
	case VkFormat_VK_FORMAT_D16_UNORM:
		return image.NewUncompressed("VK_FORMAT_D16_UNORM", fmts.D_U16_NORM), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8_SRGB_BLOCK:
		return image.NewETC2_RGB8("VK_FORMAT_ETC2_R8G8B8_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK:
		return image.NewETC2_RGB8("VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8A8_SRGB_BLOCK:
		return image.NewETC2_RGBA8_EAC("VK_FORMAT_ETC2_R8G8B8A8_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK:
		return image.NewETC2_RGBA8_EAC("VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_R16G16_UNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16_UNORM", fmts.RG_U16_NORM), nil
	default:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	}
}

func getDepthImageFormatFromVulkanFormat(vkfmt VkFormat) (*image.Format, error) {
	switch vkfmt {
	case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT_S8_UINT", fmts.DS_F32U8), nil
	case VkFormat_VK_FORMAT_D32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT", fmts.D_F32), nil
	case VkFormat_VK_FORMAT_D16_UNORM:
		return image.NewUncompressed("VK_FORMAT_D16_UNORM", fmts.D_U16_NORM), nil
	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D16_UNORM_S8_UINT", fmts.DS_NU16U8), nil
	case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D24_UNORM_S8_UINT", fmts.DS_NU24U8), nil
	default:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	}
}

func setCubemapFace(img *image.Info2D, cubeMap *gfxapi.CubemapLevel, layerIndex uint32) (success bool) {
	if cubeMap == nil || img == nil {
		return false
	}
	switch layerIndex {
	case 0:
		cubeMap.PositiveX = img
	case 1:
		cubeMap.NegativeX = img
	case 2:
		cubeMap.PositiveY = img
	case 3:
		cubeMap.NegativeY = img
	case 4:
		cubeMap.PositiveZ = img
	case 5:
		cubeMap.NegativeZ = img
	default:
		return false
	}
	return true
}

// ResourceData returns the resource data given the current state.
func (t *ImageObject) ResourceData(ctx log.Context, s *gfxapi.State, resources gfxapi.ResourceMap) (interface{}, error) {
	ctx = ctx.Enter("ImageObject.Resource()")

	vkFmt := t.Info.Format
	format, err := getImageFormatFromVulkanFormat(vkFmt)
	if err != nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceName())}
	}
	switch t.Info.ImageType {
	case VkImageType_VK_IMAGE_TYPE_2D:
		// If this image has VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT set, it should have six layers to
		// represent a cubemap
		if uint32(t.Info.Flags)&uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT) != 0 {
			// Cubemap
			cubeMapLevels := make([]*gfxapi.CubemapLevel, len(t.Layers[0].Levels))
			for l := range cubeMapLevels {
				cubeMapLevels[l] = &gfxapi.CubemapLevel{}
			}
			for layerIndex, imageLayer := range t.Layers {
				for levelIndex, imageLevel := range imageLayer.Levels {
					img := &image.Info2D{
						Format: format,
						Width:  imageLevel.Width,
						Height: imageLevel.Height,
						Data:   image.NewID(imageLevel.Data.ResourceID(ctx, s)),
					}
					if !setCubemapFace(img, cubeMapLevels[levelIndex], layerIndex) {
						return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceName())}
					}
				}
			}
			return &gfxapi.Cubemap{Levels: cubeMapLevels}, nil
		} else {
			levels := make([]*image.Info2D, len(t.Layers[0].Levels))
			for i, level := range t.Layers[0].Levels {
				levels[i] = &image.Info2D{
					Format: format,
					Width:  level.Width,
					Height: level.Height,
					Data:   image.NewID(level.Data.ResourceID(ctx, s)),
				}
			}
			return &gfxapi.Texture2D{Levels: levels}, nil
		}
	default:
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceName())}
	}
}

func (t *ImageObject) SetResourceData(ctx log.Context, at *path.Command,
	data interface{}, resources gfxapi.ResourceMap, edits gfxapi.ReplaceCallback) error {
	return fmt.Errorf("SetResourceData is not supported for ImageObject")
}

// IsResource returns true if this instance should be considered as a resource.
func (s *ShaderModuleObject) IsResource() bool {
	return true
}

// ResourceName returns the UI name for the resource.
func (s *ShaderModuleObject) ResourceName() string {
	return fmt.Sprintf("Shader<0x%x>", s.VulkanHandle)
}

// ResourceType returns the type of this resource.
func (s *ShaderModuleObject) ResourceType() gfxapi.ResourceType {
	return gfxapi.ResourceType_ShaderResource
}

// ResourceData returns the resource data given the current state.
func (s *ShaderModuleObject) ResourceData(ctx log.Context, t *gfxapi.State, resources gfxapi.ResourceMap) (interface{}, error) {
	ctx = ctx.Enter("Shader.ResourceData()")
	words := s.Words.Read(ctx, nil, t, nil)
	source := shadertools.DisassembleSpirvBinary(words)
	return &gfxapi.Shader{Type: gfxapi.ShaderType_Spirv, Source: source}, nil
}

func (shader *ShaderModuleObject) SetResourceData(ctx log.Context, at *path.Command,
	data interface{}, resourceIDs gfxapi.ResourceMap, edits gfxapi.ReplaceCallback) error {
	ctx = ctx.Enter("ShaderModuleObject.SetResourceData()")
	// Dirty. TODO: Make separate type for getting info for a single resource.
	capturePath := at.Commands.Capture
	resources, err := resolve.Resources(ctx, capturePath)
	if err != nil {
		return err
	}
	resourceID := resourceIDs[shader]

	resource := resources.Find(shader.ResourceType(), resourceID)
	if resource == nil {
		return fmt.Errorf("Couldn't find resource")
	}

	c, err := capture.ResolveFromPath(ctx, capturePath)
	if err != nil {
		return err
	}

	list, err := c.Atoms(ctx)
	if err != nil {
		return err
	}

	index := len(resource.Accesses) - 1
	for resource.Accesses[index] > at.Index && index >= 0 {
		index--
	}
	for j := index; j >= 0; j-- {
		i := resource.Accesses[j]
		if a, ok := list.Atoms[i].(*VkCreateShaderModule); ok {
			edits(uint64(i), a.Replace(ctx, data))
			return nil
		}
	}
	return fmt.Errorf("No atom to set data in")
}

func (a *VkCreateShaderModule) Replace(ctx log.Context, data interface{}) gfxapi.ResourceAtom {
	ctx = ctx.Enter("VkCreateShaderModule.Replace()")
	state := capture.NewState(ctx)
	a.Mutate(ctx, state, nil)

	shader := data.(*gfxapi.Shader)
	codeSlice := shadertools.AssembleSpirvText(shader.Source)
	if codeSlice == nil {
		return nil
	}

	code := atom.Must(atom.AllocData(ctx, state, codeSlice))
	device := a.Device
	pAlloc := memory.Pointer(a.PAllocator)
	pShaderModule := memory.Pointer(a.PShaderModule)
	result := a.Result
	createInfo := a.PCreateInfo.Read(ctx, a, state, nil)

	createInfo.PCode = U32ᶜᵖ(code.Ptr())
	createInfo.CodeSize = uint64(len(codeSlice)) * 4
	// TODO(qining): The following is a hack to work around memory.Write().
	// In VkShaderModuleCreateInfo, CodeSize should be of type 'size', but
	// 'uint64' is used for now, and memory.Write() will always treat is as
	// a 8-byte type and causing padding issues so that won't encode the struct
	// correctly.
	// Possible solution: define another type 'size' and handle it correctly in
	// memory.Write().
	buf := &bytes.Buffer{}
	writer := endian.Writer(buf, state.MemoryLayout.GetEndian())
	VkShaderModuleCreateInfoEncodeRaw(state, writer, &createInfo)
	newCreateInfo := atom.Must(atom.AllocData(ctx, state, buf.Bytes()))
	newAtom := NewVkCreateShaderModule(device, newCreateInfo.Ptr(), pAlloc, pShaderModule, result)

	// Carry all non-observation extras through.
	for _, e := range a.Extras().All() {
		if _, ok := e.(*atom.Observations); !ok {
			newAtom.Extras().Add(e)
		}
	}

	// Add observations
	newAtom.AddRead(newCreateInfo.Data()).AddRead(code.Data())

	for _, w := range a.Extras().Observations().Writes {
		newAtom.AddWrite(w.Range, w.ID)
	}
	return newAtom
}
