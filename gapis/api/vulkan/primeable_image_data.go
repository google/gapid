// Copyright (C) 2019 Google Inc.
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
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

// primeableImageData can be built by imagePrimer for a specific image, whose
// data needs to be primed. primeableImageData contains the data and logic
// to prime the data for the corresponding image.
type primeableImageData interface {
	// prime fills the corresponding image with the data held by this
	// primeableImageData
	prime(sb *stateBuilder, srcLayout, dstLayout ipLayoutInfo) error
	// free destroy any staging resources required for priming the data held by
	// this primeableImageData to the corresponding image.
	free(*stateBuilder)
	// primingQueue returns the queue will be used for priming.
	primingQueue() VkQueue
}

func getQueueForPriming(sb *stateBuilder, oldStateImgObj ImageObjectʳ, queueFlagBits VkQueueFlagBits) QueueObjectʳ {
	queueCandidates := []QueueObjectʳ{}
	for _, q := range sb.imageAllLastBoundQueues(oldStateImgObj) {
		if GetState(sb.newState).Queues().Contains(q) {
			queueCandidates = append(queueCandidates, GetState(sb.newState).Queues().Get(q))
		}
	}
	return sb.getQueueFor(queueFlagBits,
		queueFamilyIndicesToU32Slice(oldStateImgObj.Info().QueueFamilyIndices()),
		oldStateImgObj.Device(), queueCandidates...)
}

func deferUntilAllCommittedExecuted(sb *stateBuilder, queue VkQueue, f ...func()) {
	tsk := newQueueCommandBatch("")
	tsk.DeferToPostExecuted(func() {
		for _, ff := range f {
			ff()
		}
	})
	tsk.Commit(sb, sb.scratchRes.GetQueueCommandHandler(sb, queue))
}

type ipPrimeableHostCopy struct {
	queue VkQueue
	kits  []ipHostCopyKit
}

func (c ipPrimeableHostCopy) prime(sb *stateBuilder, srcLayout, dstLayout ipLayoutInfo) error {
	var err error
	if len(c.kits) == 0 {
		return fmt.Errorf("None host copy kit for priming by host copy")
	}
	dstImageObjs := []ImageObjectʳ{}
	for _, kit := range c.kits {
		dstImgObj := GetState(sb.newState).Images().Get(kit.dstImage)
		inList := false
		for _, dst := range dstImageObjs {
			if dst.VulkanHandle() == dstImgObj.VulkanHandle() {
				inList = true
				break
			}
		}
		if inList == false {
			dstImageObjs = append(dstImageObjs, dstImgObj)
		}
	}

	preCopyBarriers := []VkImageMemoryBarrier{}
	postCopyBarriers := []VkImageMemoryBarrier{}

	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, c.queue)
	for _, dstImgObj := range dstImageObjs {
		preCopyBarriers = append(preCopyBarriers, ipImageLayoutTransitionBarriers(sb, dstImgObj, srcLayout, useSpecifiedLayout(ipHostCopyImageLayout))...)
		postCopyBarriers = append(postCopyBarriers, ipImageLayoutTransitionBarriers(sb, dstImgObj, useSpecifiedLayout(ipHostCopyImageLayout), dstLayout)...)

	}
	err = ipRecordImageMemoryBarriers(sb, queueHandler, preCopyBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at pre host copy image layout transition")
	}
	for _, k := range c.kits {
		cmdBatch := k.BuildHostCopyCommands(sb)
		err := cmdBatch.Commit(sb, queueHandler)
		if err != nil {
			return log.Errf(sb.ctx, err, "failed at commit buffer image copy commands")
		}
	}

	err = ipRecordImageMemoryBarriers(sb, queueHandler, postCopyBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at post host copy image layout transition")
	}
	return nil
}

func (c ipPrimeableHostCopy) free(sb *stateBuilder) {
	// do nothing
}

func (c ipPrimeableHostCopy) primingQueue() VkQueue {
	return c.queue
}

type ipPrimeableDeviceCopy struct {
	queue VkQueue
	kits  []ipDeviceCopyKit
}

func (c ipPrimeableDeviceCopy) prime(sb *stateBuilder, srcLayout, dstLayout ipLayoutInfo) error {
	var err error
	if len(c.kits) == 0 {
		return fmt.Errorf("None device copy kit for priming by device copy")
	}
	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, c.queue)
	dstImageObjs := []ImageObjectʳ{}
	for _, kit := range c.kits {
		dstImgObj := GetState(sb.newState).Images().Get(kit.dstImage)
		inList := false
		for _, dst := range dstImageObjs {
			if dst.VulkanHandle() == dstImgObj.VulkanHandle() {
				inList = true
				break
			}
		}
		if inList == false {
			dstImageObjs = append(dstImageObjs, dstImgObj)
		}
	}
	preCopyBarriers := []VkImageMemoryBarrier{}
	postCopyBarriers := []VkImageMemoryBarrier{}
	for _, dstImgObj := range dstImageObjs {
		preCopyBarriers = append(preCopyBarriers, ipImageLayoutTransitionBarriers(sb, dstImgObj, srcLayout, useSpecifiedLayout(ipHostCopyImageLayout))...)
		postCopyBarriers = append(postCopyBarriers, ipImageLayoutTransitionBarriers(sb, dstImgObj, useSpecifiedLayout(ipHostCopyImageLayout), dstLayout)...)
	}
	err = ipRecordImageMemoryBarriers(sb, queueHandler, preCopyBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at pre device copy image layout transition")
	}

	for _, k := range c.kits {
		cmdBatch := k.BuildDeviceCopyCommands(sb)
		err := cmdBatch.Commit(sb, queueHandler)
		if err != nil {
			return log.Errf(sb.ctx, err, "failed at commit device image copy commands")
		}
	}

	err = ipRecordImageMemoryBarriers(sb, queueHandler, postCopyBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at post device copy image layout transition")
	}
	return nil
}

func (c ipPrimeableDeviceCopy) free(sb *stateBuilder) {
	// do nothing
}

func (c ipPrimeableDeviceCopy) primingQueue() VkQueue {
	return c.queue
}

type ipPrimeableRenderKits struct {
	img           VkImage
	queue         VkQueue
	kits          []ipRenderKit
	freeCallbacks []func()
}

func (pi *ipPrimeableRenderKits) free(sb *stateBuilder) {
	// staging images and memories will not be freed immediately, but wait until all the tasks on its queue are finished.
	if len(pi.freeCallbacks) > 0 {
		deferUntilAllCommittedExecuted(sb, pi.queue, pi.freeCallbacks...)
		// Avoid the double free causing issue.
		pi.freeCallbacks = nil
	}
}

func (pi *ipPrimeableRenderKits) primingQueue() VkQueue {
	return pi.queue
}

func (pi *ipPrimeableRenderKits) prime(sb *stateBuilder, srcLayout, dstLayout ipLayoutInfo) error {
	var err error
	newStateImgObj := GetState(sb.newState).Images().Get(pi.img)
	if newStateImgObj.IsNil() {
		return log.Errf(sb.ctx, fmt.Errorf("Nil Image in new state"), "[Priming by buffer imageStore, img: %v]", pi.img)
	}
	renderingLayout := ipRenderColorOutputLayout
	if (newStateImgObj.Info().Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) != 0 {
		renderingLayout = ipRenderDepthStencilOutputLayout
	}
	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, pi.queue)
	preRenderingBarriers := ipImageLayoutTransitionBarriers(sb, newStateImgObj, srcLayout, useSpecifiedLayout(renderingLayout))
	err = ipRecordImageMemoryBarriers(sb, queueHandler, preRenderingBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at pre rendering image layout transition")
	}
	for _, kit := range pi.kits {
		cmdBatch := kit.BuildRenderCommands(sb)
		if err = cmdBatch.Commit(sb, queueHandler); err != nil {
			return log.Errf(sb.ctx, err, "failed at committing render kit commands")
		}
	}
	postRenderingBarriers := ipImageLayoutTransitionBarriers(sb, newStateImgObj, useSpecifiedLayout(renderingLayout), dstLayout)
	err = ipRecordImageMemoryBarriers(sb, queueHandler, postRenderingBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at post rendering image layout transition")
	}
	return nil
}

type ipPrimeableStoreKits struct {
	img           VkImage
	queue         VkQueue
	kits          []ipStoreKit
	freeCallbacks []func()
}

func (pi *ipPrimeableStoreKits) free(sb *stateBuilder) {
	// staging images and memories will not be freed immediately, but wait until all the tasks on its queue are finished.
	if len(pi.freeCallbacks) > 0 {
		deferUntilAllCommittedExecuted(sb, pi.queue, pi.freeCallbacks...)
		// Avoid the double free causing issue.
		pi.freeCallbacks = nil
	}
}

func (pi *ipPrimeableStoreKits) primingQueue() VkQueue {
	return pi.queue
}

func (pi *ipPrimeableStoreKits) prime(sb *stateBuilder, srcLayout, dstLayout ipLayoutInfo) error {
	var err error
	newStateImgObj := GetState(sb.newState).Images().Get(pi.img)
	if newStateImgObj.IsNil() {
		return log.Errf(sb.ctx, fmt.Errorf("Nil Image in new state"), "[Priming by buffer imageStore, img: %v]", pi.img)
	}
	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, pi.queue)
	preStoreBarriers := ipImageLayoutTransitionBarriers(sb, newStateImgObj, srcLayout, useSpecifiedLayout(ipStoreImageLayout))
	err = ipRecordImageMemoryBarriers(sb, queueHandler, preStoreBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at recording pre image store layout transition")
	}
	for _, kit := range pi.kits {
		cmdBatch := kit.BuildStoreCommands(sb)
		if err := cmdBatch.Commit(sb, queueHandler); err != nil {
			return log.Errf(sb.ctx, err, "failed at committing store kit commands")
		}
	}
	postStoreBarriers := ipImageLayoutTransitionBarriers(sb, newStateImgObj, useSpecifiedLayout(ipStoreImageLayout), dstLayout)
	err = ipRecordImageMemoryBarriers(sb, queueHandler, postStoreBarriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at recording post image store layout transition")
	}
	return nil
}

// ipPrimeableByPreinitialization contains the data for priming through mapping
// host data to the underlying memory.
type ipPrimeableByPreinitialization struct {
	p                 *imagePrimer
	img               VkImage
	opaqueBoundRanges []VkImageSubresourceRange
	queue             VkQueue
}

func (pi *ipPrimeableByPreinitialization) free(sb *stateBuilder) {}

func (pi *ipPrimeableByPreinitialization) primingQueue() VkQueue { return pi.queue }

func (pi *ipPrimeableByPreinitialization) prime(sb *stateBuilder, srcLayout, dstLayout ipLayoutInfo) error {
	oldStateImgObj := GetState(sb.oldState).Images().Get(pi.img)
	if oldStateImgObj.IsNil() {
		return log.Errf(sb.ctx, fmt.Errorf("Nil Image in old state"), "[Priming by preinitialization, image: %v]", pi.img)
	}
	newStateImgObj := GetState(sb.newState).Images().Get(pi.img)
	if newStateImgObj.IsNil() {
		return log.Errf(sb.ctx, fmt.Errorf("Nil Image in new state"), "[Priming by preinitialization, image: %v]", pi.img)
	}
	// TODO: Handle multi-planar images
	newImgPlaneMemInfo, _ := subGetImagePlaneMemoryInfo(sb.ctx, nil, api.CmdNoID, nil, sb.newState, GetState(sb.newState), 0, nil, nil, newStateImgObj, VkImageAspectFlagBits(0))
	newMem := newImgPlaneMemInfo.BoundMemory()
	oldImgPlaneMemInfo, _ := subGetImagePlaneMemoryInfo(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, GetState(sb.oldState), 0, nil, nil, oldStateImgObj, VkImageAspectFlagBits(0))
	boundOffset := oldImgPlaneMemInfo.BoundMemoryOffset()
	dataAndSlices := []hashedDataAndOffset{}

	for _, rng := range pi.opaqueBoundRanges {
		walkImageSubresourceRange(sb, oldStateImgObj, rng,
			func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
				origLevel := oldStateImgObj.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
				origDataSlice := origLevel.Data()
				linearLayout := origLevel.LinearLayout()
				hashed := newHashedDataFromSlice(sb.ctx, sb.oldState, origDataSlice)
				dataAndSlices = append(dataAndSlices, newHashedDataAndOffset(hashed, uint64(boundOffset+linearLayout.Offset())))
				if srcLayout.layoutOf(aspect, layer, level) != VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED {
					log.E(sb.ctx, "Error: Priming image data by preinitialization, image source layout is not VK_IMAGE_LAYOUT_PREINITIALIZED, img: %v, aspect: %v, layer: %v, level: %v", newStateImgObj.VulkanHandle(), aspect, layer, level)
				}
			})
	}

	err := flushDataToMemory(sb, newMem.VulkanHandle(), 1, dataAndSlices...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at flushing data to the backing memory")
	}

	barriers := ipImageLayoutTransitionBarriers(sb, newStateImgObj, useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED), dstLayout)
	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, pi.queue)
	err = ipRecordImageMemoryBarriers(sb, queueHandler, barriers...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at post memory mapping image layout transition")
	}
	return nil
}

// newPrimeableImageDataFromHost builds primeable image data for the given image with
// the specific opaque memory bound subresource ranges. The built primeable
// image data takes the data from the given image in the old state of the image
// primer's stateBuilder, and is able to prime the data to the image with the
// same Vulkan Handle in the new state of the stateBuilder. The image data will
// be collected from the device memory.
func (p *imagePrimer) newPrimeableImageDataFromHost(img VkImage, opaqueBoundRanges []VkImageSubresourceRange) (primeableImageData, error) {
	nilQueueErr := fmt.Errorf("Nil Queue")
	queueNotExistInNewState := func(q VkQueue) error { return fmt.Errorf("Queue: %v does not exist in new state", q) }

	oldStateImgObj := GetState(p.sb.oldState).Images().Get(img)
	transDstBit := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT)
	attBits := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)
	storageBit := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT)

	isDepth := (oldStateImgObj.Info().Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) != 0

	primeByCopy := (oldStateImgObj.Info().Usage()&transDstBit) != 0 && (!isDepth)
	if primeByCopy {
		queue := getQueueForPriming(p.sb, oldStateImgObj,
			VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT|VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by buffer -> image copy, image: %v]", img)
		}
		recipes := map[VkImageAspectFlagBits]*ipHostCopyRecipe{}
		for _, rng := range opaqueBoundRanges {
			walkImageSubresourceRange(p.sb, oldStateImgObj, rng,
				func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
					if _, ok := recipes[aspect]; !ok {
						recipes[aspect] = &ipHostCopyRecipe{
							srcImageInOldState: img,
							srcAspect:          aspect,
							dstImageInNewState: img,
							dstAspect:          aspect,
							wordIndex:          uint32(0),
							subAspectPieces:    []ipHostCopyRecipeSubAspectPiece{},
						}
					}
					recipe := recipes[aspect]
					recipe.subAspectPieces = append(recipe.subAspectPieces, ipHostCopyRecipeSubAspectPiece{
						layer:        layer,
						level:        level,
						offsetX:      0,
						offsetY:      0,
						offsetZ:      0,
						extentWidth:  uint32(levelSize.width),
						extentHeight: uint32(levelSize.height),
						extentDepth:  uint32(levelSize.depth),
					})
				})
		}
		if isSparseResidency(oldStateImgObj) {
			walkSparseImageMemoryBindings(p.sb, oldStateImgObj, func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
				if _, ok := recipes[aspect]; !ok {
					recipes[aspect] = &ipHostCopyRecipe{
						srcImageInOldState: img,
						srcAspect:          aspect,
						dstImageInNewState: img,
						dstAspect:          aspect,
						wordIndex:          uint32(0),
						subAspectPieces:    []ipHostCopyRecipeSubAspectPiece{},
					}
				}
				recipe := recipes[aspect]
				recipe.subAspectPieces = append(recipe.subAspectPieces, ipHostCopyRecipeSubAspectPiece{
					layer:        layer,
					level:        level,
					offsetX:      uint32(blockData.Offset().X()),
					offsetY:      uint32(blockData.Offset().Y()),
					offsetZ:      uint32(blockData.Offset().Z()),
					extentWidth:  blockData.Extent().Width(),
					extentHeight: blockData.Extent().Height(),
					extentDepth:  blockData.Extent().Depth(),
				})
			})
		}
		recipeList := []ipHostCopyRecipe{}
		for _, r := range recipes {
			recipeList = append(recipeList, *r)
		}
		dev := queue.Device()
		kb := p.GetHostCopyKitBuilder(dev)
		kits, err := kb.BuildHostCopyKits(p.sb, recipeList...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at building host copy kits for host copy")
		}
		return &ipPrimeableHostCopy{queue: queue.VulkanHandle(), kits: kits}, nil
	}

	primeByRendering := (!primeByCopy) && ((oldStateImgObj.Info().Usage() & attBits) != 0)
	if primeByRendering {
		queue := getQueueForPriming(p.sb, oldStateImgObj, VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by rendering host data: %v]", img)
		}
		dev := queue.Device()
		primeable := &ipPrimeableRenderKits{img: img, queue: queue.VulkanHandle(), kits: []ipRenderKit{}}
		stagingImages := map[VkImageAspectFlagBits][]ImageObjectʳ{}

		hostCopyRecipes := map[VkImageAspectFlagBits][]*ipHostCopyRecipe{}
		for _, aspect := range p.sb.imageAspectFlagBits(oldStateImgObj, oldStateImgObj.ImageAspect()) {
			stagingImgs, freeStagingImgs, err := p.Create32BitUintColorStagingImagesForAspect(
				oldStateImgObj, aspect, VkImageUsageFlags(
					VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT|
						VkImageUsageFlagBits_VK_IMAGE_USAGE_INPUT_ATTACHMENT_BIT|
						VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT))
			if err != nil {
				primeable.free(p.sb)
				return nil, log.Errf(p.sb.ctx, err, "[Creating staging images for priming image data by rendering host data, image: %v, aspect: %v]", img, aspect)
			}
			stagingImages[aspect] = stagingImgs
			hostCopyRecipes[aspect] = make([]*ipHostCopyRecipe, len(stagingImgs))
			primeable.freeCallbacks = append(primeable.freeCallbacks, freeStagingImgs)

		}
		for _, rng := range opaqueBoundRanges {
			walkImageSubresourceRange(p.sb, oldStateImgObj, rng,
				func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
					for i, simg := range stagingImages[aspect] {
						if hostCopyRecipes[aspect][i] == nil {
							hostCopyRecipes[aspect][i] = &ipHostCopyRecipe{
								srcImageInOldState: img,
								srcAspect:          aspect,
								dstImageInNewState: simg.VulkanHandle(),
								dstAspect:          VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
								wordIndex:          uint32(i),
								subAspectPieces:    []ipHostCopyRecipeSubAspectPiece{},
							}
						}
						copyRecipe := hostCopyRecipes[aspect][i]
						copyRecipe.subAspectPieces = append(copyRecipe.subAspectPieces, ipHostCopyRecipeSubAspectPiece{
							layer:        layer,
							level:        level,
							offsetX:      0,
							offsetY:      0,
							offsetZ:      0,
							extentWidth:  uint32(levelSize.width),
							extentHeight: uint32(levelSize.height),
							extentDepth:  uint32(levelSize.depth),
						})
					}
				})
		}
		if isSparseResidency(oldStateImgObj) {
			walkSparseImageMemoryBindings(p.sb, oldStateImgObj, func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
				for i, simg := range stagingImages[aspect] {
					if hostCopyRecipes[aspect][i] == nil {
						hostCopyRecipes[aspect][i] = &ipHostCopyRecipe{
							srcImageInOldState: img,
							srcAspect:          aspect,
							dstImageInNewState: simg.VulkanHandle(),
							dstAspect:          VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
							wordIndex:          uint32(i),
							subAspectPieces:    []ipHostCopyRecipeSubAspectPiece{},
						}
					}
					copyRecipe := hostCopyRecipes[aspect][i]
					copyRecipe.subAspectPieces = append(copyRecipe.subAspectPieces, ipHostCopyRecipeSubAspectPiece{
						layer:        layer,
						level:        level,
						offsetX:      uint32(blockData.Offset().X()),
						offsetY:      uint32(blockData.Offset().Y()),
						offsetZ:      uint32(blockData.Offset().Z()),
						extentWidth:  blockData.Extent().Width(),
						extentHeight: blockData.Extent().Height(),
						extentDepth:  blockData.Extent().Depth(),
					})
				}
			})
		}
		copyList := make([]ipHostCopyRecipe, 0, len(hostCopyRecipes)*2)
		for _, rs := range hostCopyRecipes {
			for _, r := range rs {
				// recipe pointer can be nil if the aspect has no real data
				// e.g. all layers and levels have UNDEFINED layout.
				if r != nil {
					copyList = append(copyList, *r)
				}
			}
		}
		copyKitBuilder := p.GetHostCopyKitBuilder(dev)
		copyKits, err := copyKitBuilder.BuildHostCopyKits(p.sb, copyList...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at build host data copy kits for staging images")
		}
		copy := &ipPrimeableHostCopy{queue: queue.VulkanHandle(), kits: copyKits}
		err = copy.prime(p.sb,
			useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED),
			useSpecifiedLayout(ipRenderInputAttachmentLayout))
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at roll out the host data copy to staging images")
		}

		newStateImgObj := GetState(p.sb.newState).Images().Get(img)
		kb := p.GetRenderKitBuilder(dev)
		recipes := []ipRenderRecipe{}
		for _, copy := range copyList {
			for _, piece := range copy.subAspectPieces {
				sizes := p.sb.levelSize(newStateImgObj.Info().Extent(), newStateImgObj.Info().Fmt(), piece.level, copy.srcAspect)
				r := ipRenderRecipe{
					inputAttachmentImage:  copy.dstImageInNewState,
					inputAttachmentAspect: VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
					renderImage:           newStateImgObj.VulkanHandle(),
					renderAspect:          copy.srcAspect,
					layer:                 piece.layer,
					level:                 piece.level,
					renderRectX:           int32(piece.offsetX),
					renderRectY:           int32(piece.offsetY),
					renderRectWidth:       piece.extentWidth,
					renderRectHeight:      piece.extentHeight,
					wordIndex:             copy.wordIndex,
					framebufferWidth:      uint32(sizes.width),
					framebufferHeight:     uint32(sizes.height),
				}
				recipes = append(recipes, r)
			}
		}

		kits, err := kb.BuildRenderKits(p.sb, recipes...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed to build render kits from recipes")
		}
		primeable.kits = kits

		return primeable, nil
	}

	primeByImageStore := (!primeByCopy) && (!primeByRendering) && ((oldStateImgObj.Info().Usage() & storageBit) != 0)
	if primeByImageStore {
		queue := getQueueForPriming(p.sb, oldStateImgObj, VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by host data imageStore operation, image: %v]", img)
		}
		if !GetState(p.sb.newState).Queues().Contains(queue.VulkanHandle()) {
			return nil, log.Errf(p.sb.ctx, queueNotExistInNewState(queue.VulkanHandle()), "[Building primeable image data that can be primed by host data imageStore operation, image: %v]", img)
		}

		dev := queue.Device()
		primeable := &ipPrimeableStoreKits{img: img, queue: queue.VulkanHandle(), kits: []ipStoreKit{}}
		stagingImages := map[VkImageAspectFlagBits][]ImageObjectʳ{}
		hostCopyRecipes := map[VkImageAspectFlagBits][]*ipHostCopyRecipe{}
		for _, aspect := range p.sb.imageAspectFlagBits(oldStateImgObj, oldStateImgObj.ImageAspect()) {
			stagingImgs, freeStagingImgs, err := p.Create32BitUintColorStagingImagesForAspect(
				oldStateImgObj, aspect, VkImageUsageFlags(
					VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT|
						VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT))
			if err != nil {
				primeable.free(p.sb)
				return nil, log.Errf(p.sb.ctx, err, "[Creating staging images for priming image data by imageStore host data, image: %v, aspect: %v]", img, aspect)
			}
			stagingImages[aspect] = stagingImgs
			hostCopyRecipes[aspect] = make([]*ipHostCopyRecipe, len(stagingImgs))
			primeable.freeCallbacks = append(primeable.freeCallbacks, freeStagingImgs)

		}
		for _, rng := range opaqueBoundRanges {
			walkImageSubresourceRange(p.sb, oldStateImgObj, rng,
				func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
					for i, simg := range stagingImages[aspect] {
						if hostCopyRecipes[aspect][i] == nil {
							hostCopyRecipes[aspect][i] = &ipHostCopyRecipe{
								srcImageInOldState: img,
								srcAspect:          aspect,
								dstImageInNewState: simg.VulkanHandle(),
								dstAspect:          VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
								wordIndex:          uint32(i),
								subAspectPieces:    []ipHostCopyRecipeSubAspectPiece{},
							}
						}
						copyRecipe := hostCopyRecipes[aspect][i]
						copyRecipe.subAspectPieces = append(copyRecipe.subAspectPieces, ipHostCopyRecipeSubAspectPiece{
							layer:        layer,
							level:        level,
							offsetX:      0,
							offsetY:      0,
							offsetZ:      0,
							extentWidth:  uint32(levelSize.width),
							extentHeight: uint32(levelSize.height),
							extentDepth:  uint32(levelSize.depth),
						})
					}
				})
		}
		if isSparseResidency(oldStateImgObj) {
			walkSparseImageMemoryBindings(p.sb, oldStateImgObj, func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
				for i, simg := range stagingImages[aspect] {
					if hostCopyRecipes[aspect][i] == nil {
						hostCopyRecipes[aspect][i] = &ipHostCopyRecipe{
							srcImageInOldState: img,
							srcAspect:          aspect,
							dstImageInNewState: simg.VulkanHandle(),
							dstAspect:          VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
							wordIndex:          uint32(i),
							subAspectPieces:    []ipHostCopyRecipeSubAspectPiece{},
						}
					}
					copyRecipe := hostCopyRecipes[aspect][i]
					copyRecipe.subAspectPieces = append(copyRecipe.subAspectPieces, ipHostCopyRecipeSubAspectPiece{
						layer:        layer,
						level:        level,
						offsetX:      uint32(blockData.Offset().X()),
						offsetY:      uint32(blockData.Offset().Y()),
						offsetZ:      uint32(blockData.Offset().Z()),
						extentWidth:  blockData.Extent().Width(),
						extentHeight: blockData.Extent().Height(),
						extentDepth:  blockData.Extent().Depth(),
					})
				}
			})
		}
		copyList := make([]ipHostCopyRecipe, 0, len(hostCopyRecipes)*2)
		for _, rs := range hostCopyRecipes {
			for _, r := range rs {
				// recipe pointer can be nil if the aspect has no real data
				// e.g. all layers and levels have UNDEFINED layout.
				if r != nil {
					copyList = append(copyList, *r)
				}
			}
		}
		copyKitBuilder := p.GetHostCopyKitBuilder(dev)
		copyKits, err := copyKitBuilder.BuildHostCopyKits(p.sb, copyList...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at build host data copy kits for staging images")
		}
		copy := &ipPrimeableHostCopy{queue: queue.VulkanHandle(), kits: copyKits}
		err = copy.prime(p.sb,
			useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED),
			useSpecifiedLayout(ipStoreImageLayout))
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at roll out the host data copy to staging images")
		}

		newStateImgObj := GetState(p.sb.newState).Images().Get(img)
		kb := p.GetStoreKitBuilder(dev)
		recipes := []ipStoreRecipe{}
		for _, copy := range copyList {
			for _, piece := range copy.subAspectPieces {
				r := ipStoreRecipe{
					inputImage:   copy.dstImageInNewState,
					inputAspect:  VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
					outputImage:  newStateImgObj.VulkanHandle(),
					outputAspect: copy.srcAspect,
					layer:        piece.layer,
					level:        piece.level,
					wordIndex:    copy.wordIndex,
					extentWidth:  piece.extentWidth,
					extentHeight: piece.extentHeight,
					extentDepth:  piece.extentDepth,
					offsetX:      int32(piece.offsetX),
					offsetY:      int32(piece.offsetY),
					offsetZ:      int32(piece.offsetZ),
				}
				recipes = append(recipes, r)
			}
		}
		kits, err := kb.BuildStoreKits(p.sb, recipes...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed to build store kits from recipes")
		}
		primeable.kits = kits
		return primeable, nil
	}

	primeByPreinitialization := (!primeByCopy) && (!primeByRendering) && (!primeByImageStore) && (oldStateImgObj.Info().Tiling() == VkImageTiling_VK_IMAGE_TILING_LINEAR) && (oldStateImgObj.Info().InitialLayout() == VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED)
	if primeByPreinitialization {
		queue := getQueueForPriming(p.sb, oldStateImgObj, VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT|VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by preinitialization with host data, image: %v]", img)
		}
		return &ipPrimeableByPreinitialization{p: p, img: img, opaqueBoundRanges: opaqueBoundRanges, queue: queue.VulkanHandle()}, nil

	}
	return nil, log.Errf(p.sb.ctx, nil, "No way build primeable image data for image: %v", img)
}

// newPrimeableImageDataFromDevice builds primeable image data from the on device source image.
func (p *imagePrimer) newPrimeableImageDataFromDevice(srcImg, dstImg VkImage) (primeableImageData, error) {
	nilQueueErr := fmt.Errorf("Nil Queue")
	notImplErr := fmt.Errorf("Not Implemented")
	srcImgObj := GetState(p.sb.newState).Images().Get(srcImg)
	dstImgObj := GetState(p.sb.newState).Images().Get(dstImg)

	transDstBit := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT)
	attBits := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)
	storageBit := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT)
	isDepth := (dstImgObj.Info().Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) != 0

	if isSparseResidency(srcImgObj) != isSparseResidency(dstImgObj) {
		return nil, fmt.Errorf("src image residency does not match with dst image residency")
	}
	primeByCopy := (dstImgObj.Info().Usage()&transDstBit) != 0 && (!isDepth)
	if primeByCopy {
		queue := getQueueForPriming(p.sb, dstImgObj,
			VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT|VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by copy from on device image: %v]", srcImg)
		}
		kit, err := ipBuildDeviceCopyKit(p.sb, srcImg, dstImg)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at building device copy kits.")
		}
		return &ipPrimeableDeviceCopy{queue: queue.VulkanHandle(), kits: []ipDeviceCopyKit{kit}}, nil
	}

	primeByRendering := (!primeByCopy) && ((dstImgObj.Info().Usage() & attBits) != 0)
	if primeByRendering {
		queue := getQueueForPriming(p.sb, dstImgObj, VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by rendering from on device image: %v]", srcImg)
		}
		dev := queue.Device()
		primeable := &ipPrimeableRenderKits{img: srcImg, queue: queue.VulkanHandle(), kits: []ipRenderKit{}}

		kb := p.GetRenderKitBuilder(dev)
		recipes := []ipRenderRecipe{}

		if isSparseResidency(dstImgObj) {
			walkSparseImageMemoryBindings(p.sb, dstImgObj, func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
				sizes := p.sb.levelSize(dstImgObj.Info().Extent(), dstImgObj.Info().Fmt(), level, aspect)
				r := ipRenderRecipe{
					inputAttachmentImage:  srcImg,
					inputAttachmentAspect: VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
					renderImage:           dstImg,
					renderAspect:          aspect,
					layer:                 layer,
					level:                 level,
					renderRectX:           blockData.Offset().X(),
					renderRectY:           blockData.Offset().Y(),
					renderRectWidth:       blockData.Extent().Width(),
					renderRectHeight:      blockData.Extent().Height(),
					wordIndex:             uint32(0),
					framebufferWidth:      uint32(sizes.width),
					framebufferHeight:     uint32(sizes.height),
				}
				recipes = append(recipes, r)
			})
		} else {
			walkImageSubresourceRange(p.sb, dstImgObj, p.sb.imageWholeSubresourceRange(dstImgObj),
				func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
					sizes := p.sb.levelSize(dstImgObj.Info().Extent(), dstImgObj.Info().Fmt(), level, aspect)
					r := ipRenderRecipe{
						inputAttachmentImage:  srcImg,
						inputAttachmentAspect: VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
						renderImage:           dstImg,
						renderAspect:          aspect,
						layer:                 layer,
						level:                 level,
						renderRectX:           int32(0),
						renderRectY:           int32(0),
						renderRectWidth:       uint32(levelSize.width),
						renderRectHeight:      uint32(levelSize.height),
						wordIndex:             uint32(0),
						framebufferWidth:      uint32(sizes.width),
						framebufferHeight:     uint32(sizes.height),
					}
					recipes = append(recipes, r)
				})
		}
		kits, err := kb.BuildRenderKits(p.sb, recipes...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed to build render kits from recipes")
		}
		primeable.kits = kits

		return primeable, nil
	}

	primeByImageStore := (!primeByCopy) && (!primeByRendering) && ((dstImgObj.Info().Usage() & storageBit) != 0)
	if primeByImageStore {
		queue := getQueueForPriming(p.sb, dstImgObj, VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
		if queue.IsNil() {
			return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by imageStore operation from device image: %v]", srcImg)
		}
		if !GetState(p.sb.newState).Queues().Contains(queue.VulkanHandle()) {
			return nil, fmt.Errorf("Queue: %v does not exist", queue)
		}

		dev := queue.Device()
		primeable := &ipPrimeableStoreKits{img: srcImg, queue: queue.VulkanHandle(), kits: []ipStoreKit{}}

		recipes := []ipStoreRecipe{}
		if isSparseResidency(dstImgObj) {
			walkSparseImageMemoryBindings(p.sb, dstImgObj, func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
				r := ipStoreRecipe{
					inputImage:   srcImg,
					inputAspect:  aspect,
					outputImage:  dstImg,
					outputAspect: aspect,
					layer:        layer,
					level:        level,
					wordIndex:    uint32(0),
					extentWidth:  blockData.Extent().Width(),
					extentHeight: blockData.Extent().Height(),
					extentDepth:  blockData.Extent().Depth(),
					offsetX:      blockData.Offset().X(),
					offsetY:      blockData.Offset().Y(),
					offsetZ:      blockData.Offset().Z(),
				}
				recipes = append(recipes, r)
			})
		} else {
			walkImageSubresourceRange(p.sb, dstImgObj, p.sb.imageWholeSubresourceRange(dstImgObj),
				func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
					r := ipStoreRecipe{
						inputImage:   srcImg,
						inputAspect:  aspect,
						outputImage:  dstImg,
						outputAspect: aspect,
						layer:        layer,
						level:        level,
						wordIndex:    uint32(0),
						extentWidth:  uint32(levelSize.width),
						extentHeight: uint32(levelSize.height),
						extentDepth:  uint32(levelSize.depth),
						offsetX:      int32(0),
						offsetY:      int32(0),
						offsetZ:      int32(0),
					}
					recipes = append(recipes, r)
				})
		}
		kb := p.GetStoreKitBuilder(dev)
		kits, err := kb.BuildStoreKits(p.sb, recipes...)
		if err != nil {
			return nil, log.Errf(p.sb.ctx, err, "failed at build store kits for priming storage image")
		}
		primeable.kits = kits
		return primeable, nil
	}
	primeByPreinitialization := (!primeByCopy) && (!primeByRendering) && (!primeByImageStore) && (dstImgObj.Info().Tiling() == VkImageTiling_VK_IMAGE_TILING_LINEAR) && (dstImgObj.Info().InitialLayout() == VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED)
	if primeByPreinitialization {
		return nil, notImplErr
	}
	return nil, log.Errf(p.sb.ctx, nil, "No way build primeable image data for image: %v", dstImg)
}
