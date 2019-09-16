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
	"context"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
)

type stateWatcher struct {
	memoryWrites map[memory.PoolID]*interval.U64SpanList
}

func (b *stateWatcher) OnBeginCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {}
func (b *stateWatcher) OnEndCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd)   {}
func (b *stateWatcher) OnBeginSubCmd(ctx context.Context, subIdx api.SubCmdIdx, recordIdx api.RecordIdx) {
}
func (b *stateWatcher) OnRecordSubCmd(ctx context.Context, recordIdx api.RecordIdx) {}
func (b *stateWatcher) OnEndSubCmd(ctx context.Context)                             {}
func (b *stateWatcher) OnReadFrag(ctx context.Context, owner api.RefObject, frag api.Fragment, valueRef api.RefObject, track bool) {
}
func (b *stateWatcher) OnWriteFrag(ctx context.Context, owner api.RefObject, frag api.Fragment, oldValueRef api.RefObject, newValueRef api.RefObject, track bool) {
}

func (b *stateWatcher) OnWriteSlice(ctx context.Context, slice memory.Slice) {

	span := interval.U64Span{
		Start: slice.Base(),
		End:   slice.Base() + slice.Size(),
	}

	poolID := slice.Pool()

	if _, ok := b.memoryWrites[poolID]; !ok {
		b.memoryWrites[poolID] = &interval.U64SpanList{}
	}

	interval.Merge(b.memoryWrites[poolID], span, true)
}

func (b *stateWatcher) OnReadSlice(ctx context.Context, slice memory.Slice)                  {}
func (b *stateWatcher) OnWriteObs(ctx context.Context, observations []api.CmdObservation)    {}
func (b *stateWatcher) OnReadObs(ctx context.Context, observations []api.CmdObservation)     {}
func (b *stateWatcher) OpenForwardDependency(ctx context.Context, dependencyID interface{})  {}
func (b *stateWatcher) CloseForwardDependency(ctx context.Context, dependencyID interface{}) {}
func (b *stateWatcher) DropForwardDependency(ctx context.Context, dependencyID interface{})  {}

// Transfrom
type frameLoop struct {
	capture        *capture.GraphicsCapture
	cmds           []api.Cmd
	numInitialCmds int
	loopCount      int32
	loopStartIdx   int
	loopEndIdx     int
	backupState    *api.GlobalState
	watcher        *stateWatcher

	bufferCreated   map[VkBuffer]bool
	bufferChanged   map[VkBuffer]bool
	bufferDestroyed map[VkBuffer]bool
	bufferToBackup  map[VkBuffer]VkBuffer

	imageCreated   map[VkImage]bool
	imageChanged   map[VkImage]bool
	imageDestroyed map[VkImage]bool
	imageToBackup  map[VkImage]VkImage

	descriptorSetLayoutCreated map[VkDescriptorSetLayout]bool
	// descriptorSetLayoutChanged   map[VkDescriptorSetLayout]bool
	descriptorSetLayoutDestroyed map[VkDescriptorSetLayout]bool
	// descriptorSetLayoutToBackup  map[VkDescriptorSetLayout]VkDescriptorSetLayout

	loopCountPtr value.Pointer

	frameNum uint32
}

func newFrameLoop(ctx context.Context, graphicsCapture *capture.GraphicsCapture, numInitialCmds int, Cmds []api.Cmd, loopCount int32) *frameLoop {

	f := &frameLoop{

		capture:        graphicsCapture,
		cmds:           Cmds,
		numInitialCmds: numInitialCmds,
		loopCount:      loopCount,

		loopStartIdx: 0,
		loopEndIdx:   len(Cmds) - 1,

		watcher: &stateWatcher{
			memoryWrites: make(map[memory.PoolID]*interval.U64SpanList),
		},

		bufferCreated:   make(map[VkBuffer]bool),
		bufferChanged:   make(map[VkBuffer]bool),
		bufferDestroyed: make(map[VkBuffer]bool),
		bufferToBackup:  make(map[VkBuffer]VkBuffer),

		imageCreated:   make(map[VkImage]bool),
		imageChanged:   make(map[VkImage]bool),
		imageDestroyed: make(map[VkImage]bool),
		imageToBackup:  make(map[VkImage]VkImage),

		descriptorSetLayoutCreated: make(map[VkDescriptorSetLayout]bool),
		// descriptorSetLayoutChanged:   make(map[VkDescriptorSetLayout]bool),
		descriptorSetLayoutDestroyed: make(map[VkDescriptorSetLayout]bool),
		// descriptorSetLayoutToBackup:  make(map[VkDescriptorSetLayout]VkDescriptorSetLayout),
	}

	f.loopStartIdx, f.loopEndIdx = f.getLoopStartAndEndIndices(ctx, Cmds)
	return f
}

func (f *frameLoop) Transform(ctx context.Context, cmdId api.CmdID, cmd api.Cmd, out transform.Writer) {

	ctx = log.Enter(ctx, "FrameLoop Transform")

	if cmd == f.cmds[f.loopStartIdx] {

		log.D(ctx, "Loop: start loop at frame %v, cmdId %v, cmd %v.", f.frameNum, cmdId, cmd)
		f.detectChangedResource(ctx, out.State())

		apiState := GetState(out.State())

		stateBuilder := apiState.newStateBuilder(ctx, newTransformerOutput(out))
		defer stateBuilder.ta.Dispose()

		if err := f.backupChangedResources(ctx, stateBuilder); err != nil {
			log.E(ctx, "Failed to backup changed resources: %v", err)
			return
		}

		stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			f.loopCountPtr = b.AllocateMemory(4)
			b.Push(value.S32(f.loopCount))
			b.Store(f.loopCountPtr)
			b.JumpLabel(uint32(0x1))
			return nil
		}))

		out.NotifyPreLoop(ctx)

	} else if cmd == f.cmds[f.loopEndIdx] {

		log.D(ctx, "Loop: last frame is %v cmdId %v, cmd is %v.", f.frameNum, cmdId, cmd)

		apiState := GetState(out.State())

		stateBuilder := apiState.newStateBuilder(ctx, newTransformerOutput(out))
		defer stateBuilder.ta.Dispose()

		out.MutateAndWrite(ctx, cmdId, cmd)

		// Notify this is the end part of the loop to next transformer
		out.NotifyPostLoop(ctx)

		if err := f.resetResource(ctx, stateBuilder); err != nil {
			log.E(ctx, "Failed to reset changed resources %v.", err)
			return
		}

		// Add jump instruction
		stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			b.Load(protocol.Type_Int32, f.loopCountPtr)
			b.Sub(1)
			b.Clone(0)
			b.Store(f.loopCountPtr)
			b.JumpNZ(uint32(0x1))
			return nil
		}))

		return
	}

	if _, ok := cmd.(*VkQueuePresentKHR); ok {
		f.frameNum++
	}

	out.MutateAndWrite(ctx, cmdId, cmd)
}

func (f *frameLoop) Flush(ctx context.Context, out transform.Writer)    {}
func (f *frameLoop) PreLoop(ctx context.Context, out transform.Writer)  {}
func (f *frameLoop) PostLoop(ctx context.Context, out transform.Writer) {}

// TODO: Find out from which command are the start and the end the loop.
func (f *frameLoop) getLoopStartAndEndIndices(ctx context.Context, Cmds []api.Cmd) (startCmd, endCmd int) {

	startCmd = f.numInitialCmds
	endCmd = len(Cmds) - 1

	return startCmd, endCmd
}

func (f *frameLoop) detectChangedResource(ctx context.Context, startState *api.GlobalState) {

	f.backupState = f.capture.NewUninitializedState(ctx)
	f.backupState.Memory = startState.Memory.Clone()

	for apiState, graphicsApi := range startState.APIs {

		clonedState := graphicsApi.Clone(f.backupState.Arena)
		clonedState.SetupInitialState(ctx)

		f.backupState.APIs[apiState] = clonedState
	}

	// Loop through each command mutating the shadow state and looking at what has been created/destroyed
	err := api.ForeachCmd(ctx, f.cmds[f.loopStartIdx:f.loopEndIdx], func(ctx context.Context, cmdId api.CmdID, cmd api.Cmd) error {

		cmd.Extras().Observations().ApplyWrites(f.backupState.Memory.ApplicationPool())

		switch cmd.(type) {

		// Buffers.
		case *VkCreateBuffer:
			vkCmd := cmd.(*VkCreateBuffer)
			buffer := vkCmd.PBuffer().MustRead(ctx, vkCmd, f.backupState, nil)
			log.D(ctx, "Buffer %v created.", buffer)
			f.bufferCreated[buffer] = true

		case *VkDestroyBuffer:
			vkCmd := cmd.(*VkDestroyBuffer)
			buffer := vkCmd.Buffer()
			log.D(ctx, "Buffer %v destroyed.", buffer)
			f.bufferDestroyed[buffer] = true

		// Images
		case *VkCreateImage:
			vkCmd := cmd.(*VkCreateImage)
			img := vkCmd.PImage().MustRead(ctx, vkCmd, f.backupState, nil)
			log.D(ctx, "Image %v created", img)
			f.imageCreated[img] = true

		case *VkDestroyImage:
			vkCmd := cmd.(*VkDestroyImage)
			img := vkCmd.Image()
			log.D(ctx, "Image %v destroyed", img)
			f.imageDestroyed[img] = true

		// DescriptionSetLayout(s)
		case *VkCreateDescriptorSetLayout:
			// type VkCreateDescriptorSetLayout struct {
			//         Thread               int64    `protobuf:"zigzag64,1,opt,name=thread,proto3" json:"thread,omitempty"`
			//         Device               int64    `protobuf:"zigzag64,8,opt,name=device,proto3" json:"device,omitempty"`
			//         PCreateInfo          int64    `protobuf:"zigzag64,9,opt,name=pCreateInfo,proto3" json:"pCreateInfo,omitempty"`
			//         PAllocator           int64    `protobuf:"zigzag64,10,opt,name=pAllocator,proto3" json:"pAllocator,omitempty"`
			//         PSetLayout           int64    `protobuf:"zigzag64,11,opt,name=pSetLayout,proto3" json:"pSetLayout,omitempty"`
			//         XXX_NoUnkeyedLiteral struct{} `json:"-"`
			//         XXX_unrecognized     []byte   `json:"-"`
			//         XXX_sizecache        int32    `json:"-"`
			// }
			vkCmd := cmd.(*VkCreateDescriptorSetLayout)
			descriptorSetLayout := vkCmd.PSetLayout().MustRead(ctx, vkCmd, f.backupState, nil)
			log.D(ctx, "DescriptorSetLayout %v created", descriptorSetLayout)
			f.descriptorSetLayoutCreated[descriptorSetLayout] = true

		case *VkDestroyDescriptorSetLayout:
			// type VkDestroyDescriptorSetLayout struct {
			//         Thread               int64    `protobuf:"zigzag64,1,opt,name=thread,proto3" json:"thread,omitempty"`
			//         Device               int64    `protobuf:"zigzag64,8,opt,name=device,proto3" json:"device,omitempty"`
			//         DescriptorSetLayout  int64    `protobuf:"zigzag64,9,opt,name=descriptorSetLayout,proto3" json:"descriptorSetLayout,omitempty"`
			//         PAllocator           int64    `protobuf:"zigzag64,10,opt,name=pAllocator,proto3" json:"pAllocator,omitempty"`
			//         XXX_NoUnkeyedLiteral struct{} `json:"-"`
			//         XXX_unrecognized     []byte   `json:"-"`
			//         XXX_sizecache        int32    `json:"-"`
			// }
			vkCmd := cmd.(*VkDestroyDescriptorSetLayout)
			descriptorSetLayout := vkCmd.DescriptorSetLayout()
			log.D(ctx, "DescriptorSetLayout %v created", descriptorSetLayout)
			f.descriptorSetLayoutDestroyed[descriptorSetLayout] = true

		// DescriptorPool(s)
		case *VkCreateDescriptorPool:
			// vkCmd := cmd.(*VkCreateDescriptorPool)

		case *VkDestroyDescriptorPool:
			// vkCmd := cmd.(*VkDestroyDescriptorPool)

		// DescriptorSet(s)
		case *VkAllocateDescriptorSets:
			// vkCmd := cmd.(*VkAllocateDescriptorSets)

		case *VkFreeDescriptorSets:
			// vkCmd := cmd.(*VkFreeDescriptorSets)

		case *VkUpdateDescriptorSets:
			// vkCmd := cmd.(*VkUpdateDescriptorSets)

			// TODO: Recreate destroyed resources.
		}

		if err := cmd.Mutate(ctx, cmdId, f.backupState, nil, f.watcher); err != nil {
			return fmt.Errorf("%v: %v: %v", cmdId, cmd, err)
		}

		return nil
	})

	if err != nil {
		log.E(ctx, "Mutate error: [%v].", err)
	}

	apiState := GetState(f.backupState)

	// Find out changed buffers.
	for bufferKey, buffer := range apiState.Buffers().All() {

		data := buffer.Memory().Data()
		span := interval.U64Span{data.Base(), data.Base() + data.Size()}
		poolID := data.Pool()

		// Did we see this buffer get written to during the loop? If we did, then we need to capture the values at the start of the loop.
		if writes, ok := f.watcher.memoryWrites[poolID]; ok {

			// We do this by comparing the buffer's memory extent with all the observed written areas.
			if _, count := interval.Intersect(writes, span); count != 0 {
				f.bufferChanged[bufferKey] = true
			}
		}
	}

	// Find out changed images.
	for imageKey, image := range apiState.Images().All() {

		// We exempt the frame buffer (swap chain) images from capture.
		if image.IsSwapchainImage() {
			continue
		}

		// Gotta remember to process all aspects, layers and levels of an image
		for _, imageAspect := range image.Aspects().All() {

			for _, layer := range imageAspect.Layers().All() {

				for _, level := range layer.Levels().All() {

					data := level.Data()
					span := interval.U64Span{data.Base(), data.Base() + data.Size()}
					poolID := data.Pool()

					// Did we see this part of this image get written to during the loop? If we did, then we need to capture the values at the start of the loop.
					if writes, ok := f.watcher.memoryWrites[poolID]; ok {

						// We do this by comparing the image's part's memory extent with all the observed written areas.
						if _, count := interval.Intersect(writes, span); count != 0 {
							f.imageChanged[imageKey] = true
							break
						}
					}
				}
			}
		}
	}
	// TODO: Find out other changed resources.
}

func (f *frameLoop) backupChangedResources(ctx context.Context, stateBuilder *stateBuilder) error {

	if err := f.backupChangedBuffers(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.backupChangedImages(ctx, stateBuilder); err != nil {
		return err
	}

	// TODO: Backup other resources.
	return nil
}

func (f *frameLoop) backupChangedBuffers(ctx context.Context, stateBuilder *stateBuilder) error {

	for buffer := range f.bufferChanged {

		if _, present := f.bufferCreated[buffer]; present {
			continue
		}

		if _, preset := f.bufferDestroyed[buffer]; preset {
			continue
		}

		log.D(ctx, "Buffer [%v] changed during loop.", buffer)

		bufferObj := GetState(stateBuilder.oldState).Buffers().Get(buffer)
		if bufferObj == NilBufferObjectʳ {
			return log.Err(ctx, nil, "Buffer is nil")
		}

		queue := stateBuilder.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			queueFamilyIndicesToU32Slice(bufferObj.Info().QueueFamilyIndices()),
			bufferObj.Device(),
			bufferObj.LastBoundQueue())

		if queue == NilQueueObjectʳ {
			return log.Err(ctx, nil, "Queue is nil")
		}

		stagingBuffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
			return GetState(stateBuilder.oldState).Buffers().Contains(VkBuffer(x))
		}))

		err := stateBuilder.createSameBuffer(bufferObj, stagingBuffer)
		if err != nil {
			return log.Errf(ctx, err, "Create staging buffer for buffer %v failed: %v", buffer)
		}

		task := newQueueCommandBatch(
			fmt.Sprintf("Copy buffer: %v", stagingBuffer),
		)

		stateBuilder.copyBuffer(buffer, stagingBuffer, queue, task)

		if err := task.Commit(stateBuilder, stateBuilder.scratchRes.GetQueueCommandHandler(stateBuilder, queue.VulkanHandle())); err != nil {
			return log.Errf(ctx, err, "Copy from buffer %v to %v failed", buffer, stagingBuffer)
		}

		f.bufferToBackup[buffer] = stagingBuffer
	}

	stateBuilder.scratchRes.Free(stateBuilder)
	return nil
}

func (f *frameLoop) backupChangedImages(ctx context.Context, stateBuilder *stateBuilder) error {

	apiState := GetState(stateBuilder.oldState)

	imgPrimer := newImagePrimer(stateBuilder)
	defer imgPrimer.Free()

	for img := range f.imageChanged {

		if _, present := f.imageCreated[img]; present {
			continue
		}

		log.D(ctx, "Image [%v] changed during loop.", img)

		// Create staging Image which is used to backup the changed images
		imgObj := apiState.Images().Get(img)
		stagingImage, _, err := imgPrimer.CreateSameStagingImage(imgObj)

		if err != nil {
			return log.Err(ctx, err, "Create staging image failed.")
		}

		f.imageToBackup[img] = stagingImage.VulkanHandle()

		if err := f.copyImage(ctx, imgObj, stagingImage, stateBuilder); err != nil {
			return log.Err(ctx, err, "Copy image failed")
		}
	}

	return nil
}

func (f *frameLoop) resetResource(ctx context.Context, stateBuilder *stateBuilder) error {

	if err := f.resetBuffers(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetImages(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetDescriptorSetLayouts(ctx, stateBuilder); err != nil {
		return err
	}

	//TODO: Reset other resources.
	return nil
}

func (f *frameLoop) resetBuffers(ctx context.Context, stateBuilder *stateBuilder) error {

	if len(f.bufferToBackup) == 0 {
		return nil
	}

	for dst, src := range f.bufferToBackup {

		bufferObj := GetState(stateBuilder.newState).Buffers().Get(src)

		queue := stateBuilder.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			queueFamilyIndicesToU32Slice(bufferObj.Info().QueueFamilyIndices()),
			bufferObj.Device(),
			bufferObj.LastBoundQueue())

		task := newQueueCommandBatch(
			fmt.Sprintf("Reset buffer %v", dst),
		)

		stateBuilder.copyBuffer(src, dst, queue, task)

		if err := task.Commit(stateBuilder, stateBuilder.scratchRes.GetQueueCommandHandler(stateBuilder, queue.VulkanHandle())); err != nil {
			return log.Errf(ctx, err, "Reset buffer [%v] with buffer [%v] failed", dst, src)
		}

		log.D(ctx, "Reset buffer [%v] with buffer [%v] succeed", dst, src)
	}

	stateBuilder.scratchRes.Free(stateBuilder)
	return nil
}

func (f *frameLoop) resetImages(ctx context.Context, stateBuilder *stateBuilder) error {

	if len(f.imageToBackup) == 0 {
		return nil
	}

	apiState := GetState(stateBuilder.newState)

	imgPrimer := newImagePrimer(stateBuilder)
	defer imgPrimer.Free()

	for dst, src := range f.imageToBackup {

		dstObj := apiState.Images().Get(dst)

		primeable, err := imgPrimer.newPrimeableImageDataFromDevice(src, dst)
		if err != nil {
			return log.Errf(ctx, err, "Create primeable image data for image %v", dst)
		}
		defer primeable.free(stateBuilder)

		err = primeable.prime(stateBuilder, useSpecifiedLayout(dstObj.Info().InitialLayout()), sameLayoutsOfImage(dstObj))
		if err != nil {
			return log.Errf(ctx, err, "Priming image %v with data", dst)
		}

		log.D(ctx, "Prime image from [%v] to [%v] succeed", src, dst)
	}

	return nil
}

func (f *frameLoop) copyImage(ctx context.Context, srcImg, dstImg ImageObjectʳ, stateBuilder *stateBuilder) error {

	deviceCopyKit, err := ipBuildDeviceCopyKit(stateBuilder, srcImg.VulkanHandle(), dstImg.VulkanHandle())
	if err != nil {
		return log.Err(ctx, err, "create ipBuildDeviceCopyKit failed")
	}

	queue := getQueueForPriming(stateBuilder, srcImg, VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)

	queueHandler := stateBuilder.scratchRes.GetQueueCommandHandler(stateBuilder, queue.VulkanHandle())
	preCopyBarriers := ipImageLayoutTransitionBarriers(stateBuilder, dstImg, useSpecifiedLayout(srcImg.Info().InitialLayout()), useSpecifiedLayout(ipHostCopyImageLayout))

	if err = ipRecordImageMemoryBarriers(stateBuilder, queueHandler, preCopyBarriers...); err != nil {
		return log.Err(ctx, err, "Failed at pre device copy image layout transition")
	}

	cmdBatch := deviceCopyKit.BuildDeviceCopyCommands(stateBuilder)

	if err = cmdBatch.Commit(stateBuilder, queueHandler); err != nil {
		return log.Err(ctx, err, "Failed at commit buffer image copy commands")
	}

	postCopyBarriers := ipImageLayoutTransitionBarriers(stateBuilder, dstImg, useSpecifiedLayout(ipHostCopyImageLayout), sameLayoutsOfImage(dstImg))
	if err = ipRecordImageMemoryBarriers(stateBuilder, queueHandler, postCopyBarriers...); err != nil {
		return log.Err(ctx, err, "Failed at post device copy image layout transition")
	}

	return nil
}

func (f *frameLoop) resetDescriptorSetLayouts(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every DescriptorSetLayout that was created during the loop...
	for created, _ := range f.descriptorSetLayoutCreated {

		// If that DescriptorSetLayout was not destroyed before the end of the loop, we need to destroy it, to put the state back to where it should be.
		if _, ok := f.descriptorSetLayoutDestroyed[created]; !ok {

			// Write the command
			// stateBuilder.write(stateBuilder.cb.VkDestroyDescriptorSetLayout(created.Device(), descriptorSetLayout, allocator));
		}
	}

	// For every DescriptorSetLayout that was destroyed during the loop...
	for destroyed, _ := range f.descriptorSetLayoutDestroyed {

		// If that DescriptorSetLayout was not created after the start of the loop, we need to re-create it, to put the state back to where it should be.
		if _, ok := f.descriptorSetLayoutCreated[destroyed]; !ok {

			// Write the command
			// stateBuilder.write(stateBuilder.cb.VkCreateDescriptorSetLayout(destroyed.Device(), pCreateInfo, pAllocator, pSetLayout, result));
		}
	}

	return nil
}
