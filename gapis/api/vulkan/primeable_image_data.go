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
	"sync"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

// primeableImageData can be built by imagePrimer for a specific image, whose
// data needs to be primed. primeableImageData contains the data and logic
// to prime the data for the corresponding image.
type primeableImageData interface {
	// prime fills the corresponding image with the data held by this
	// primeableImageData
	prime(srcLayout, dstLayout ipLayoutInfo) error
	// free destroy any staging resources required for priming the data held by
	// this primeableImageData to the corresponding image.
	free()
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

// ipPrimeableByBufferCopy contains the data for priming through buffer image
// copy host data.
type ipPrimeableByBufferCopy struct {
	p           *imagePrimer
	img         VkImage
	queue       VkQueue
	copySession *ipBufferCopySession
}

func (pi *ipPrimeableByBufferCopy) prime(srcLayout, dstLayout ipLayoutInfo) error {
	err := pi.copySession.rolloutBufCopies(pi.queue, srcLayout, dstLayout)
	if err != nil {
		return log.Errf(pi.p.sb.ctx, err, "[Rolling out the buf->img copy commands for image: %v]", pi.img)
	}
	return nil
}

func (pi *ipPrimeableByBufferCopy) free() {}

func (pi *ipPrimeableByBufferCopy) primingQueue() VkQueue { return pi.queue }

// ipPrimeableByRendering contains the data for priming through rendering from
// staging images.
type ipPrimeableByRendering struct {
	p                    *imagePrimer
	img                  VkImage
	stagingImages        map[VkImageAspectFlagBits][]ImageObjectʳ
	freeCallbacks        []func()
	queue                VkQueue
	renderTaskCommitLock sync.Mutex
}

func (pi *ipPrimeableByRendering) free() {
	// staging images and memories will not be freed immediately, but wait until all the tasks on its queue are finished.
	freeTsk := pi.p.sb.newScratchTaskOnQueue(pi.queue)
	freeTsk.deferUntilExecuted(func() {
		for _, f := range pi.freeCallbacks {
			f()
		}
	})
	freeTsk.commit()
}

func (pi *ipPrimeableByRendering) primingQueue() VkQueue { return pi.queue }

func (pi *ipPrimeableByRendering) prime(srcLayout, dstLayout ipLayoutInfo) error {
	oldStateImgObj := GetState(pi.p.sb.oldState).Images().Get(pi.img)
	if oldStateImgObj.IsNil() {
		return log.Errf(pi.p.sb.ctx, fmt.Errorf("Nil Image in old state"), "[Priming by rendering, image: %v]", pi.img)
	}
	newStateImgObj := GetState(pi.p.sb.newState).Images().Get(pi.img)
	if newStateImgObj.IsNil() {
		return log.Errf(pi.p.sb.ctx, fmt.Errorf("Nil Image in new state"), "[Priming by rendering, image: %v]", pi.img)
	}
	renderTsk := pi.p.sb.newScratchTaskOnQueue(pi.queue)
	renderJobs := []*ipRenderJob{}
	for _, aspect := range pi.p.sb.imageAspectFlagBits(oldStateImgObj, oldStateImgObj.ImageAspect()) {
		for layer := uint32(0); layer < oldStateImgObj.Info().ArrayLayers(); layer++ {
			for level := uint32(0); level < oldStateImgObj.Info().MipLevels(); level++ {
				inputImageObjects := pi.stagingImages[aspect]
				inputImages := make([]ipRenderImage, len(inputImageObjects))
				for i, iimg := range inputImageObjects {
					inputImages[i] = ipRenderImage{
						image:         iimg,
						aspect:        VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
						layer:         layer,
						level:         level,
						initialLayout: VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
						finalLayout:   VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					}
				}
				renderJobs = append(renderJobs, &ipRenderJob{
					inputAttachmentImages: inputImages,
					renderTarget: ipRenderImage{
						image:         newStateImgObj,
						aspect:        aspect,
						layer:         layer,
						level:         level,
						initialLayout: srcLayout.layoutOf(aspect, layer, level),
						finalLayout:   dstLayout.layoutOf(aspect, layer, level),
					},
					inputFormat: newStateImgObj.Info().Fmt(),
				})
			}
		}
	}
	for _, renderJob := range renderJobs {
		err := pi.p.rh.render(renderJob, renderTsk)
		if err != nil {
			log.E(pi.p.sb.ctx, "[Priming image: %v, aspect: %v, layer: %v, level: %v data by rendering] %v",
				renderJob.renderTarget.image.VulkanHandle(),
				renderJob.renderTarget.aspect,
				renderJob.renderTarget.layer,
				renderJob.renderTarget.level, err)
		}
	}
	if err := renderTsk.commit(); err != nil {
		return log.Errf(pi.p.sb.ctx, err, "[Committing scratch task for priming image: %v data by rendering]", pi.img)
	}
	return nil
}

// ipPrimeableByBufferImageStore contains the data for priming through
// imageStore operations with buffer whose data from the host side.
type ipPrimeableByBufferImageStore struct {
	p                 *imagePrimer
	img               VkImage
	opaqueBoundRanges []VkImageSubresourceRange
	queue             VkQueue
}

func (pi *ipPrimeableByBufferImageStore) free() {}

func (pi *ipPrimeableByBufferImageStore) primingQueue() VkQueue { return pi.queue }

func (pi *ipPrimeableByBufferImageStore) prime(srcLayout, dstLayout ipLayoutInfo) error {
	oldStateImgObj := GetState(pi.p.sb.oldState).Images().Get(pi.img)
	if oldStateImgObj.IsNil() {
		return log.Errf(pi.p.sb.ctx, fmt.Errorf("Nil Image in old state"), "[Priming by buffer imageStore, img: %v]", pi.img)
	}
	newStateImgObj := GetState(pi.p.sb.newState).Images().Get(pi.img)
	if newStateImgObj.IsNil() {
		return log.Errf(pi.p.sb.ctx, fmt.Errorf("Nil Image in new state"), "[Priming by buffer imageStore, img: %v]", pi.img)
	}
	storeJobs := []*ipStoreJob{}
	for _, rng := range pi.opaqueBoundRanges {
		walkImageSubresourceRange(pi.p.sb, oldStateImgObj, rng,
			func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
				storeJobs = append(storeJobs, &ipStoreJob{
					// Use oldStateImgObj as target as later the store handler
					// will fetch the source data from the shadow memory of the
					// store target object. Note the Vulkan handle of the old
					// state image object is same as the handle of the new state
					// image object.
					storeTarget:       oldStateImgObj,
					aspect:            aspect,
					layer:             layer,
					level:             level,
					opaqueBlockOffset: MakeVkOffset3D(pi.p.sb.ta),
					opaqueBlockExtent: NewVkExtent3D(pi.p.sb.ta,
						uint32(levelSize.width),
						uint32(levelSize.height),
						uint32(levelSize.depth),
					),
				})
			})
	}

	if isSparseResidency(oldStateImgObj) {
		walkSparseImageMemoryBindings(pi.p.sb, oldStateImgObj,
			func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
				storeJobs = append(storeJobs, &ipStoreJob{
					storeTarget:       oldStateImgObj,
					aspect:            aspect,
					layer:             layer,
					level:             level,
					opaqueBlockOffset: blockData.Offset(),
					opaqueBlockExtent: blockData.Extent(),
				})
			})
	}
	whole := pi.p.sb.imageWholeSubresourceRange(newStateImgObj)
	transitionInfo := []imageSubRangeInfo{}
	finalLayouts := []VkImageLayout{}
	walkImageSubresourceRange(pi.p.sb, newStateImgObj, whole, func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
		transitionInfo = append(transitionInfo, imageSubRangeInfo{
			aspectMask:     VkImageAspectFlags(aspect),
			baseMipLevel:   level,
			levelCount:     1,
			baseArrayLayer: layer,
			layerCount:     1,
			oldLayout:      srcLayout.layoutOf(aspect, layer, level),
			newLayout:      VkImageLayout_VK_IMAGE_LAYOUT_GENERAL,
			oldQueue:       pi.queue,
			newQueue:       pi.queue,
		})
		finalLayouts = append(finalLayouts, dstLayout.layoutOf(aspect, layer, level))
	})
	pi.p.sb.changeImageSubRangeLayoutAndOwnership(newStateImgObj.VulkanHandle(), transitionInfo)

	for _, job := range storeJobs {
		err := pi.p.sh.store(job, pi.queue)
		if err != nil {
			log.E(pi.p.sb.ctx, "[Priming image: %v aspect: %v, layer: %v, level: %v, offset: %v, extent: %v data by buffer imageStore] %v", job.storeTarget.VulkanHandle(), job.aspect, job.layer, job.level, job.opaqueBlockOffset, job.opaqueBlockExtent, err)
		}
	}

	for i := range transitionInfo {
		transitionInfo[i].oldLayout = VkImageLayout_VK_IMAGE_LAYOUT_GENERAL
		transitionInfo[i].newLayout = finalLayouts[i]
	}
	pi.p.sb.changeImageSubRangeLayoutAndOwnership(newStateImgObj.VulkanHandle(), transitionInfo)

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

func (pi *ipPrimeableByPreinitialization) free() {}

func (pi *ipPrimeableByPreinitialization) primingQueue() VkQueue { return pi.queue }

func (pi *ipPrimeableByPreinitialization) prime(srcLayout, dstLayout ipLayoutInfo) error {
	oldStateImgObj := GetState(pi.p.sb.oldState).Images().Get(pi.img)
	if oldStateImgObj.IsNil() {
		return log.Errf(pi.p.sb.ctx, fmt.Errorf("Nil Image in old state"), "[Priming by preinitialization, image: %v]", pi.img)
	}
	newStateImgObj := GetState(pi.p.sb.newState).Images().Get(pi.img)
	if newStateImgObj.IsNil() {
		return log.Errf(pi.p.sb.ctx, fmt.Errorf("Nil Image in new state"), "[Priming by preinitialization, image: %v]", pi.img)
	}
	// TODO: Handle multi-planar images
	newImgPlaneMemInfo, _ := subGetImagePlaneMemoryInfo(pi.p.sb.ctx, nil, api.CmdNoID, nil, pi.p.sb.newState, GetState(pi.p.sb.newState), 0, nil, nil, newStateImgObj, VkImageAspectFlagBits(0))
	newMem := newImgPlaneMemInfo.BoundMemory()
	oldImgPlaneMemInfo, _ := subGetImagePlaneMemoryInfo(pi.p.sb.ctx, nil, api.CmdNoID, nil, pi.p.sb.oldState, GetState(pi.p.sb.oldState), 0, nil, nil, oldStateImgObj, VkImageAspectFlagBits(0))
	boundOffset := oldImgPlaneMemInfo.BoundMemoryOffset()
	planeMemRequirements := oldImgPlaneMemInfo.MemoryRequirements()
	boundSize := planeMemRequirements.Size()
	dat := pi.p.sb.MustReserve(uint64(boundSize))

	at := NewVoidᵖ(dat.Ptr())
	atdata := pi.p.sb.newState.AllocDataOrPanic(pi.p.sb.ctx, at)
	pi.p.sb.write(pi.p.sb.cb.VkMapMemory(
		newMem.Device(),
		newMem.VulkanHandle(),
		boundOffset,
		boundSize,
		VkMemoryMapFlags(0),
		atdata.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(atdata.Data()).AddWrite(atdata.Data()))
	atdata.Free()

	transitionInfo := []imageSubRangeInfo{}
	for _, rng := range pi.opaqueBoundRanges {
		walkImageSubresourceRange(pi.p.sb, oldStateImgObj, rng,
			func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
				origLevel := oldStateImgObj.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
				origDataSlice := origLevel.Data()
				linearLayout := origLevel.LinearLayout()

				pi.p.sb.ReadDataAt(origDataSlice.ResourceID(pi.p.sb.ctx, pi.p.sb.oldState), uint64(linearLayout.Offset())+dat.Address(), origDataSlice.Size())

				transitionInfo = append(transitionInfo, imageSubRangeInfo{
					aspectMask:     VkImageAspectFlags(aspect),
					baseMipLevel:   level,
					levelCount:     1,
					baseArrayLayer: layer,
					layerCount:     1,
					oldLayout:      VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED,
					newLayout:      dstLayout.layoutOf(aspect, layer, level),
					oldQueue:       pi.queue,
					newQueue:       pi.queue,
				})
			})
	}

	pi.p.sb.write(pi.p.sb.cb.VkFlushMappedMemoryRanges(
		newMem.Device(),
		1,
		pi.p.sb.MustAllocReadData(NewVkMappedMemoryRange(pi.p.sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
			0,                     // pNext
			newMem.VulkanHandle(), // memory
			0,                     // offset
			boundSize,             // size
		)).Ptr(),
		VkResult_VK_SUCCESS,
	))
	dat.Free()

	pi.p.sb.write(pi.p.sb.cb.VkUnmapMemory(
		newMem.Device(),
		newMem.VulkanHandle(),
	))

	pi.p.sb.changeImageSubRangeLayoutAndOwnership(pi.img, transitionInfo)

	return nil
}

// newPrimeableImageData builds primeable image data for the given image with
// the specific opaque memory bound subresource ranges. The built primeable
// image data takes the data from the given image in the old state of the image
// primer's stateBuilder, and is able to prime the data to the image with the
// same Vulkan Handle in the new state of the stateBuilder. If fromHostData is
// true, the image data will be collected from the shadow memory of the old
// state image object, which is on the host accessible space. If fromHostData is
// false, the image data will be collected from the device memory.
func (p *imagePrimer) newPrimeableImageData(img VkImage, opaqueBoundRanges []VkImageSubresourceRange, fromHostData bool) (primeableImageData, error) {
	nilQueueErr := log.Err(p.sb.ctx, nil, "Nil Queue")
	notImplErr := log.Err(p.sb.ctx, nil, "Not Implemented")

	oldStateImgObj := GetState(p.sb.oldState).Images().Get(img)
	transDstBit := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT)
	attBits := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)
	storageBit := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT)

	isDepth := (oldStateImgObj.Info().Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) != 0

	primeByCopy := (oldStateImgObj.Info().Usage()&transDstBit) != 0 && (!isDepth)
	if primeByCopy {
		if fromHostData {
			queue := getQueueForPriming(p.sb, oldStateImgObj,
				VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT|VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
			if queue.IsNil() {
				return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by buffer -> image copy, image: %v]", img)
			}
			job := newImagePrimerBufCopyJob(oldStateImgObj)
			for _, aspect := range p.sb.imageAspectFlagBits(oldStateImgObj, oldStateImgObj.ImageAspect()) {
				job.addDst(p.sb.ctx, aspect, aspect, oldStateImgObj)
			}
			bcs := newImagePrimerBufferCopySession(p.sb, job)
			for _, rng := range opaqueBoundRanges {
				bcs.collectCopiesFromSubresourceRange(rng)
			}
			if isSparseResidency(oldStateImgObj) {
				bcs.collectCopiesFromSparseImageBindings()
			}
			return &ipPrimeableByBufferCopy{p: p, copySession: bcs, queue: queue.VulkanHandle()}, nil

		} else {
			return nil, log.Errf(p.sb.ctx, notImplErr, "[Building primeable image data that can be primed by image -> image copy, image: %v]", img)
		}
	}

	primeByRendering := (!primeByCopy) && ((oldStateImgObj.Info().Usage() & attBits) != 0)
	if primeByRendering {
		if fromHostData {
			queue := getQueueForPriming(p.sb, oldStateImgObj, VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)
			if queue.IsNil() {
				return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by rendering host data: %v]", img)
			}
			primeable := &ipPrimeableByRendering{p: p, img: img, stagingImages: map[VkImageAspectFlagBits][]ImageObjectʳ{}, queue: queue.VulkanHandle()}
			copyJob := newImagePrimerBufCopyJob(oldStateImgObj)
			for _, aspect := range p.sb.imageAspectFlagBits(oldStateImgObj, oldStateImgObj.ImageAspect()) {
				stagingImgs, freeStagingImgs, err := p.allocStagingImages(oldStateImgObj, aspect)
				if err != nil {
					// Free allocated staging images in case of error
					primeable.free()
					return nil, log.Errf(p.sb.ctx, err, "[Creating staging images for priming image data by rendering host data, image: %v, aspect: %v]", img, aspect)
				}
				copyJob.addDst(p.sb.ctx, aspect, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, stagingImgs...)
				primeable.stagingImages[aspect] = stagingImgs
				primeable.freeCallbacks = append(primeable.freeCallbacks, freeStagingImgs)
			}
			bcs := newImagePrimerBufferCopySession(p.sb, copyJob)
			for _, rng := range opaqueBoundRanges {
				bcs.collectCopiesFromSubresourceRange(rng)
			}
			if isSparseResidency(oldStateImgObj) {
				bcs.collectCopiesFromSparseImageBindings()
			}
			err := bcs.rolloutBufCopies(queue.VulkanHandle(), useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED), useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL))
			if err != nil {
				// Free allocated staging images in case of error.
				primeable.free()
				return nil, log.Errf(p.sb.ctx, err, "[Rolling out the buf->img copy commands for staging images, building primeable data for image: %v]", img)
			}
			return primeable, nil

		} else {
			return nil, log.Errf(p.sb.ctx, notImplErr, "[Building primeable image data that can be primed by rendering device data]")
		}
	}

	primeByImageStore := (!primeByCopy) && (!primeByRendering) && ((oldStateImgObj.Info().Usage() & storageBit) != 0)
	if primeByImageStore {
		if fromHostData {
			queue := getQueueForPriming(p.sb, oldStateImgObj, VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
			if queue.IsNil() {
				return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by host data imageStore operation, image: %v]", img)
			}
			return &ipPrimeableByBufferImageStore{p: p, img: img, opaqueBoundRanges: opaqueBoundRanges, queue: queue.VulkanHandle()}, nil
		} else {
			return nil, log.Errf(p.sb.ctx, notImplErr, "[Building primeable image data that can be primed by device data imageLoad/Store operation, image: %v]", img)
		}
	}

	primeByPreinitialization := (!primeByCopy) && (!primeByRendering) && (!primeByImageStore) && (oldStateImgObj.Info().Tiling() == VkImageTiling_VK_IMAGE_TILING_LINEAR) && (oldStateImgObj.Info().InitialLayout() == VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED)
	if primeByPreinitialization {
		if fromHostData {
			queue := getQueueForPriming(p.sb, oldStateImgObj, VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT|VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)
			if queue.IsNil() {
				return nil, log.Errf(p.sb.ctx, nilQueueErr, "[Building primeable image data that can be primed by preinitialization with host data, image: %v]", img)
			}
			return &ipPrimeableByPreinitialization{p: p, img: img, opaqueBoundRanges: opaqueBoundRanges, queue: queue.VulkanHandle()}, nil
		} else {
			return nil, log.Errf(p.sb.ctx, notImplErr, "[Building primeable image data that can be primed by preinitialization with device data, image: %v]", img)
		}
	}
	return nil, log.Errf(p.sb.ctx, nil, "No way build primeable image data for image: %v", img)
}
