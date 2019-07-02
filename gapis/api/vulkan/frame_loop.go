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
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapir"
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

func (b *stateWatcher) OnBeginCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
}

func (b *stateWatcher) OnEndCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
}
func (b *stateWatcher) OnBeginSubCmd(ctx context.Context, subIdx api.SubCmdIdx, recordIdx api.RecordIdx) {
}
func (b *stateWatcher) OnRecordSubCmd(ctx context.Context, recordIdx api.RecordIdx) {
}
func (b *stateWatcher) OnEndSubCmd(ctx context.Context) {
}
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

func (b *stateWatcher) OnReadSlice(ctx context.Context, slice memory.Slice) {
}

func (b *stateWatcher) OnWriteObs(ctx context.Context, obs []api.CmdObservation) {
}

func (b *stateWatcher) OnReadObs(ctx context.Context, obs []api.CmdObservation) {
}

func (b *stateWatcher) OpenForwardDependency(ctx context.Context, dependencyID interface{}) {
}

func (b *stateWatcher) CloseForwardDependency(ctx context.Context, dependencyID interface{}) {
}

func (b *stateWatcher) DropForwardDependency(ctx context.Context, dependencyID interface{}) {
}

// Transfrom
type frameLoop struct {
	capture        *capture.GraphicsCapture
	cmds           []api.Cmd
	numInitialCmds int
	loopCount      int32
	loopStartCmd   api.Cmd
	loopEndCmd     api.Cmd
	startState     *api.GlobalState
	watcher        *stateWatcher

	bufferCreated   map[VkBuffer]bool
	bufferChanged   map[VkBuffer]bool
	bufferDestroyed map[VkBuffer]bool
	bufferToBackup  map[VkBuffer]VkBuffer

	imageCreated   map[VkImage]bool
	imageChanged   map[VkImage]bool
	imageDestroyed map[VkImage]bool
	imageToBackup  map[ImageObjectʳ]ImageObjectʳ
	imageToBuffer  map[VkImage]VkBuffer

	loopCountPtr value.Pointer

	frameNum uint32
}

func newFrameLoop(ctx context.Context, c *capture.GraphicsCapture, numInitialCmds int, Cmds []api.Cmd, loopCount int32) *frameLoop {
	f := &frameLoop{
		capture:        c,
		cmds:           Cmds,
		numInitialCmds: numInitialCmds,
		loopCount:      loopCount,
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
		imageToBackup:  make(map[ImageObjectʳ]ImageObjectʳ),
		imageToBuffer:  make(map[VkImage]VkBuffer),
	}

	f.loopStartCmd, f.loopEndCmd = f.getLoopStartAndEndCmd(ctx, Cmds)

	return f
}

func (f *frameLoop) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	ctx = log.Enter(ctx, "frameLoop Transform")

	if cmd == f.loopStartCmd {
		log.I(ctx, "Loop: start loop at frame %v, id %v, cmd %v", f.frameNum, id, cmd)
		f.detectChangedResource(ctx, out)
		st := GetState(f.startState)
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		defer sb.ta.Dispose()

		if err := f.backupChangedResources(ctx, st, sb); err != nil {
			log.Err(ctx, err, "Failed to backup changed resources")
			return
		}
		// Add jump label
		sb.write(sb.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			f.loopCountPtr = b.AllocateMemory(4)
			b.Push(value.S32(f.loopCount))
			b.Store(f.loopCountPtr)
			b.JumpLabel(uint32(0x1))
			return nil
		}))
		out.NotifyPreLoop(ctx)

	} else if cmd == f.loopEndCmd {
		log.I(ctx, "Loop: last frame is %v cmd is %v", f.frameNum, cmd)
		st := GetState(f.startState)
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		defer sb.ta.Dispose()
		out.MutateAndWrite(ctx, id, cmd)
		// Notify this is the end part of the loop to next transformer
		out.NotifyPostLoop(ctx)

		if err := f.resetResource(ctx, st, sb); err != nil {
			log.Err(ctx, err, "Failed to reset changed resources")
			return
		}

		// Add jump instruction
		sb.write(sb.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
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

	out.MutateAndWrite(ctx, id, cmd)

}

func (f *frameLoop) Flush(ctx context.Context, out transform.Writer)    {}
func (f *frameLoop) PreLoop(ctx context.Context, out transform.Writer)  {}
func (f *frameLoop) PostLoop(ctx context.Context, out transform.Writer) {}

// TODO: find out from which command are the start and the end the loop.
func (f *frameLoop) getLoopStartAndEndCmd(ctx context.Context, Cmds []api.Cmd) (startCmd, endCmd api.Cmd) {
	startCmd = Cmds[f.numInitialCmds]
	endCmd = Cmds[len(Cmds)-1]

	for i, c := range Cmds[f.numInitialCmds:] {
		if _, ok := c.(*VkQueuePresentKHR); !ok {
			continue
		}
		// use the start cmd of the second frame as the startCmd
		if i+1 < len(Cmds) {
			startCmd = Cmds[i+1]
			break
		}
	}

	return startCmd, endCmd
}

func (f *frameLoop) detectChangedResource(ctx context.Context, out transform.Writer) {
	ctx = log.Enter(ctx, "detectChangedResource")
	gs := out.State()

	f.startState = f.capture.NewUninitializedState(ctx)
	f.startState.Memory = gs.Memory.Clone()
	for k, v := range gs.APIs {
		s := v.Clone(f.startState.Arena)
		s.SetupInitialState(ctx)
		f.startState.APIs[k] = s
	}

	s := f.startState

	err := api.ForeachCmd(ctx, f.cmds[f.numInitialCmds:], func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		switch c := cmd.(type) {
		// Images
		case *VkCreateImage:
			vkCmd := cmd.(*VkCreateImage)
			vkCmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
			img := vkCmd.PImage().MustRead(ctx, vkCmd, s, nil)
			f.imageCreated[img] = true
			cmd.Mutate(ctx, id, f.startState, nil, f.watcher)
		case *VkDestroyImage:
			vkCmd := cmd.(*VkDestroyImage)
			vkCmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
			img := vkCmd.Image()
			log.I(ctx, "Destroy image %v ", c)
			f.imageDestroyed[img] = true

		// Buffers
		case *VkCreateBuffer:
			vkCmd := cmd.(*VkCreateBuffer)
			vkCmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
			buffer := vkCmd.PBuffer().MustRead(ctx, vkCmd, s, nil)
			f.bufferCreated[buffer] = true
			cmd.Mutate(ctx, id, f.startState, nil, f.watcher)
		case *VkDestroyBuffer:
			vkCmd := cmd.(*VkDestroyBuffer)
			vkCmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
			buffer := vkCmd.Buffer()
			log.I(ctx, "Buffer %x destroyed", buffer)
			f.bufferDestroyed[buffer] = true

		// Descriptor sets
		case *VkAllocateDescriptorSets:
			vkCmd := cmd.(*VkAllocateDescriptorSets)
			log.I(ctx, "VkAllocateDescriptorSets %v", vkCmd)
			cmd.Mutate(ctx, id, f.startState, nil, f.watcher)
		case *VkUpdateDescriptorSets:
			vkCmd := cmd.(*VkUpdateDescriptorSets)
			log.I(ctx, "Update descriptorset %v", vkCmd)
			cmd.Mutate(ctx, id, f.startState, nil, f.watcher)
		case *VkFreeDescriptorSets:
			vkCmd := cmd.(*VkFreeDescriptorSets)
			log.I(ctx, "VkFreeDescriptorSets %v", vkCmd)

		// TODO: recreate destroyed resources
		case *VkDestroyInstance:
		case *VkDestroyDevice:
		case *VkDestroyCommandPool:
		case *VkFreeCommandBuffers:
		case *VkDestroyDescriptorPool:
		case *VkDestroyQueryPool:
		case *VkDestroyBufferView:
		case *VkDestroyImageView:
		case *VkDestroyShaderModule:
		case *VkDestroyPipelineCache:
		case *VkDestroyPipeline:
		case *VkFreeMemory:
		case *VkDestroyFence:
		case *VkDestroySemaphore:
		case *VkDestroyEvent:
			log.I(ctx, "ignore destroy cmd %v", cmd)

		default:
			if err := cmd.Mutate(ctx, id, f.startState, nil, f.watcher); err != nil {
				return fmt.Errorf("%v: %v: %v", id, cmd, err)
			}
		}

		return nil
	})

	if err != nil {
		log.I(ctx, "mutate error [%v]", err)
	}

	st := GetState(f.startState)

	// Find changed buffers
	vkBuffers := st.Buffers().All()
	for k, buffer := range vkBuffers {
		data := buffer.Memory().Data()
		span := interval.U64Span{data.Base(), data.Base() + data.Size()}
		poolID := data.Pool()
		if l, ok := f.watcher.memoryWrites[poolID]; ok {
			if _, count := interval.Intersect(l, span); count != 0 {
				f.bufferChanged[k] = true
			}
		}
	}

	// Find changed images
	imgs := st.Images().All()
	for k, v := range imgs {
		if v.IsSwapchainImage() {
			continue
		}
		for _, imageAspect := range v.Aspects().All() {
			for _, layer := range imageAspect.Layers().All() {
				for _, level := range layer.Levels().All() {
					data := level.Data()
					span := interval.U64Span{data.Base(), data.Base() + data.Size()}
					poolID := data.Pool()
					if l, ok := f.watcher.memoryWrites[poolID]; ok {
						if _, count := interval.Intersect(l, span); count != 0 {
							f.imageChanged[k] = true
							log.I(ctx, "image %v changed", k)
						}
					}
				}
			}
		}
	}

	// TODO: Observe other changes beside images and buffers.
}

func (f *frameLoop) backupChangedResources(ctx context.Context, st *State, sb *stateBuilder) error {
	imgPrimer := newImagePrimer(sb)
	defer imgPrimer.Free()
	if err := f.backupChangedBuffers(ctx, st, sb); err != nil {
		return err
	}

	if err := f.backupChangedImages(ctx, st, sb, imgPrimer); err != nil {
		return err
	}

	// TODO: backup other resource.
	return nil
}

func (f *frameLoop) backupChangedBuffers(ctx context.Context, st *State, sb *stateBuilder) error {
	for buffer := range f.bufferChanged {
		if _, present := f.bufferCreated[buffer]; present {
			log.I(ctx, "Buffer [%v] is created in between", buffer)
			continue
		}
		log.I(ctx, "buffer [%v] changed during loop", buffer)

		stagingBuffer := f.allocateStagingBuffer(ctx, st.Buffers().Get(buffer), sb)
		if err := f.copyBuffer(ctx, buffer, stagingBuffer, sb); err != nil {
			return fmt.Errorf("copy buffer [%v] to staging buffer [%v] failed", buffer, stagingBuffer)
		}
		log.I(ctx, "copy buffer [%v] to staging buffer [%v] succeed", buffer, stagingBuffer)
		f.bufferToBackup[buffer] = stagingBuffer
	}
	return nil
}

func (f *frameLoop) backupChangedImages(ctx context.Context, st *State, sb *stateBuilder, imgPrimer *imagePrimer) error {
	for img := range f.imageChanged {
		if _, present := f.imageCreated[img]; present {
			log.I(ctx, "Image [%v] is created in between", img)
			continue
		}
		imgObj := st.Images().Get(img)

		// Create staging Image which is used to backup the changed images
		stagingImage, _, err := imgPrimer.CreateSameStagingImage(imgObj)
		if err != nil {
			log.Err(ctx, err, "create staging image failed for backup.")
			return err
		}
		f.imageToBackup[imgObj] = stagingImage

		if err := f.copyImage(ctx, imgObj, stagingImage, st, sb); err != nil {
			log.Err(ctx, err, "copy image failed")
			return err
		}
		log.I(ctx, "backup image %v succeed", imgObj)
	}
	return nil
}

func (f *frameLoop) resetResource(ctx context.Context, st *State, sb *stateBuilder) error {
	// Reset buffers
	if err := f.resetBuffers(ctx, st, sb); err != nil {
		return err
	}

	// Reset Images
	if err := f.resetImages(ctx, st, sb); err != nil {
		return err
	}

	// TODO: reset other resource.
	return nil
}

func (f *frameLoop) resetBuffers(ctx context.Context, st *State, sb *stateBuilder) error {
	for dst, src := range f.bufferToBackup {
		if err := f.copyBuffer(ctx, src, dst, sb); err != nil {
			log.I(ctx, "reset buffer [%v] with buffer [%v] failed", src, dst)
			return err
		}
		log.I(ctx, "reset buffer [%v] with buffer [%v] succeed", src, dst)
	}
	return nil
}

func (f *frameLoop) resetImages(ctx context.Context, st *State, sb *stateBuilder) error {
	imgPrimer := newImagePrimer(sb)
	defer imgPrimer.Free()
	for dst, src := range f.imageToBackup {
		primeable, err := imgPrimer.newPrimeableImageDataFromDevice(src.VulkanHandle(), dst.VulkanHandle())
		if err != nil {
			log.E(sb.ctx, "Create primeable image data: %v", err)
			return err
		}
		defer primeable.free(sb)
		err = primeable.prime(sb, useSpecifiedLayout(dst.Info().InitialLayout()), sameLayoutsOfImage(dst))
		if err != nil {
			log.E(sb.ctx, "Priming image data: %v", err)
			return err
		}
		log.I(ctx, "prime image from [%v] to [%v] succeed", src, dst)
	}

	return nil
}

func (f *frameLoop) copyBuffer(ctx context.Context, src VkBuffer, dst VkBuffer, sb *stateBuilder) error {
	s := sb.newState
	bufferObj := GetState(s).Buffers().Get(dst)
	queue := sb.getQueueFor(
		VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
		queueFamilyIndicesToU32Slice(bufferObj.Info().QueueFamilyIndices()),
		bufferObj.Device(),
		bufferObj.LastBoundQueue())
	if queue.IsNil() {
		log.I(ctx, "last bound queue for buffer [%v] is nil", bufferObj)
		return fmt.Errorf("last bound queue for buffer [%v] is nil", bufferObj)
	}

	bufferCopy := []VkBufferCopy{}
	offset := VkDeviceSize(0)

	srcObj := GetState(sb.newState).Buffers().Get(src)
	size := srcObj.Info().Size()

	bufferCopy = append(bufferCopy, NewVkBufferCopy(sb.ta,
		offset, // srcOffset
		0,      // dstOffset
		size,   // size
	))

	queueFamily := queue.Family()
	vkDevice := queue.Device()

	// TODO: deal with sparse memory bindings
	commandPoolID := VkCommandPool(newUnusedID(false, func(x uint64) bool { ok := GetState(s).CommandPools().Contains(VkCommandPool(x)); return ok }))
	commandBufferID := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { ok := GetState(s).CommandBuffers().Contains(VkCommandBuffer(x)); return ok }))

	sb.write(sb.cb.VkCreateCommandPool(
		vkDevice,
		sb.MustAllocReadData(
			NewVkCommandPoolCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
				NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
				VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
				queueFamily, // queueFamilyIndex
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(commandPoolID).Ptr(),
		VkResult_VK_SUCCESS,
	))

	commandBufferData := sb.MustAllocWriteData(commandBufferID)

	sb.write(sb.cb.VkAllocateCommandBuffers(
		vkDevice,
		sb.MustAllocReadData(NewVkCommandBufferAllocateInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
			NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
			commandPoolID,                                                  // commandPool
			VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
			1, // commandBufferCount
		)).Ptr(),
		commandBufferData.Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkBeginCommandBuffer(
		commandBufferID,
		sb.MustAllocReadData(
			NewVkCommandBufferBeginInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
				0, // pNext
				VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
				0, // pInheritanceInfo
			)).Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkCmdPipelineBarrier(
		commandBufferID,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		0,
		memory.Nullptr,
		1,
		sb.MustAllocReadData(
			NewVkBufferMemoryBarrier(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
				0, // pNext
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
				queueFamilyIgnore,                // srcQueueFamilyIndex
				queueFamilyIgnore,                // dstQueueFamilyIndex
				src,                              // buffer
				0,                                // offset
				VkDeviceSize(0xFFFFFFFFFFFFFFFF), // size
			)).Ptr(),
		0,
		memory.Nullptr,
	))
	sb.write(sb.cb.VkCmdCopyBuffer(
		commandBufferID,
		src,
		dst,
		uint32(len(bufferCopy)),
		sb.MustAllocReadData(bufferCopy).Ptr(),
	))

	sb.write(sb.cb.VkCmdPipelineBarrier(
		commandBufferID,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		0,
		memory.Nullptr,
		1,
		sb.MustAllocReadData(
			NewVkBufferMemoryBarrier(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
				0, // pNext
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
				queueFamilyIgnore,                // srcQueueFamilyIndex
				queueFamilyIgnore,                // dstQueueFamilyIndex
				src,                              // buffer
				0,                                // offset
				VkDeviceSize(0xFFFFFFFFFFFFFFFF), // size
			)).Ptr(),
		0,
		memory.Nullptr,
	))

	sb.write(sb.cb.VkEndCommandBuffer(
		commandBufferID,
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkQueueSubmit(
		queue.VulkanHandle(),
		1,
		sb.MustAllocReadData(NewVkSubmitInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
			0, // pNext
			0, // waitSemaphoreCount
			0, // pWaitSemaphores
			0, // pWaitDstStageMask
			1, // commandBufferCount
			NewVkCommandBufferᶜᵖ(sb.MustAllocReadData(commandBufferID).Ptr()), // pCommandBuffers
			0, // signalSemaphoreCount
			0, // pSignalSemaphores
		)).Ptr(),
		VkFence(0),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkQueueWaitIdle(queue.VulkanHandle(), VkResult_VK_SUCCESS))

	// Used for debug: copy dst buffer back
	// dstObj := GetState(sb.newState).Buffers().Get(dst)
	// buflen := uint64(size)
	// dstData := sb.MustReserve(buflen)
	// ptrDstData := sb.newState.AllocDataOrPanic(sb.ctx, NewVoidᵖ(dstData.Ptr()))
	// sb.write(sb.cb.VkMapMemory(
	// 	dstObj.Device(), dstObj.Memory().VulkanHandle(), VkDeviceSize(0), VkDeviceSize(size),
	// 	VkMemoryMapFlags(0), ptrDstData.Ptr(), VkResult_VK_SUCCESS,
	// ).AddRead(ptrDstData.Data()).AddWrite(ptrDstData.Data()))

	// sb.write(sb.cb.VkInvalidateMappedMemoryRanges(
	// 	dstObj.Device(),
	// 	1,
	// 	sb.MustAllocReadData(
	// 		NewVkMappedMemoryRange(sb.ta,
	// 			VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
	// 			0,                              // pNext
	// 			dstObj.Memory().VulkanHandle(), // memory
	// 			VkDeviceSize(0),                // offset
	// 			// VkDeviceSize(0xFFFFFFFFFFFFFFFF), // size
	// 			size,
	// 		)).Ptr(),
	// 	VkResult_VK_SUCCESS,
	// ))

	// sb.write(sb.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
	// 	notificationID := b.GetNotificationID()
	// 	log.I(ctx, "register notification id %v, dstData.Address() %v", notificationID, dstData.Address())
	// 	b.Notification(notificationID, value.ObservedPointer(dstData.Address()), buflen)
	// 	err := b.RegisterNotificationReader(notificationID, func(n gapir.Notification) {
	// 		d := n.GetData()
	// 		id := n.GetId()
	// 		if id != notificationID {
	// 			log.I(ctx, "received incorrect notification id %, expect %d", id, notificationID)
	// 		}
	// 		data := d.GetData()
	// 		name := fmt.Sprintf("VkBuffer_src_%v_dst_%v_ts_%v", src, dst, time.Now().UnixNano())
	// 		file, err := os.Create(name)
	// 		defer file.Close()
	// 		if err != nil {
	// 			log.E(ctx, "%v", err)
	// 		}

	// 		byteOrder := s.MemoryLayout.GetEndian()
	// 		r := endian.Reader(bytes.NewReader(data), byteOrder)
	// 		bytes := make([]byte, buflen)
	// 		r.Data(bytes)
	// 		r.Error()
	// 		_, err = file.Write(bytes)
	// 		if err != nil {
	// 			log.E(ctx, "err write file %v", err)
	// 		}
	// 	})

	// 	if err != nil {
	// 		log.I(ctx, "RegisterNotificationReader failed %v", err)
	// 	}
	// 	return err
	// }))
	// ptrDstData.Free()
	// sb.write(sb.cb.VkUnmapMemory(
	// 	dstObj.Device(),
	// 	dstObj.Memory().VulkanHandle(),
	// ))

	// sb.write(sb.cb.VkDestroyCommandPool(
	// 	vkDevice,
	// 	commandPoolID,
	// 	memory.Nullptr,
	// ))

	return nil
}

func (f *frameLoop) copyImage(ctx context.Context, srcImg, dstImg ImageObjectʳ, st *State, sb *stateBuilder) error {

	dck, err := ipBuildDeviceCopyKit(sb, srcImg.VulkanHandle(), dstImg.VulkanHandle())
	if err != nil {
		log.Err(ctx, err, "create ipBuildDeviceCopyKit failed")
		return err
	}

	queue := getQueueForPriming(sb, srcImg, VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)

	// TODO: check the srs/dst layout
	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, queue.VulkanHandle())
	preCopyBarriers := ipImageLayoutTransitionBarriers(sb, dstImg, useSpecifiedLayout(srcImg.Info().InitialLayout()), useSpecifiedLayout(ipHostCopyImageLayout))
	if err = ipRecordImageMemoryBarriers(sb, queueHandler, preCopyBarriers...); err != nil {
		return log.Err(sb.ctx, err, "failed at pre device copy image layout transition")
	}

	cmdBatch := dck.BuildDeviceCopyCommands(sb)
	if err = cmdBatch.Commit(sb, queueHandler); err != nil {
		return log.Err(sb.ctx, err, "failed at commit buffer image copy commands")
	}
	postCopyBarriers := ipImageLayoutTransitionBarriers(sb, dstImg, useSpecifiedLayout(ipHostCopyImageLayout), sameLayoutsOfImage(dstImg))
	if err = ipRecordImageMemoryBarriers(sb, queueHandler, postCopyBarriers...); err != nil {
		return log.Err(sb.ctx, err, "failed at post device copy image layout transition")
	}

	// Debug: dump image data to file
	// f.debugCopyImageToBuffer(ctx, srcImg.VulkanHandle(), queue.VulkanHandle(), sb)
	return nil
}

func (f *frameLoop) getBufferMemoryIndex(ctx context.Context, device DeviceObjectʳ) uint32 {
	s := GetState(f.startState)
	physicalDeviceObject := s.PhysicalDevices().Get(device.PhysicalDevice())

	typeBits := uint32((uint64(1) << uint64(physicalDeviceObject.MemoryProperties().MemoryTypeCount())) - 1)
	if s.TransferBufferMemoryRequirements().Contains(device.VulkanHandle()) {
		typeBits = s.TransferBufferMemoryRequirements().Get(device.VulkanHandle()).MemoryTypeBits()
	}
	index := memoryTypeIndexFor(typeBits, physicalDeviceObject.MemoryProperties(), VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT))

	if index >= 0 {
		return uint32(index)
	}
	log.E(ctx, "cannot get the memory type index for host visible memory to create scratch buffer, fallback to use index 0")
	return 0
}

func (f *frameLoop) allocateStagingBuffer(ctx context.Context, src BufferObjectʳ, sb *stateBuilder) VkBuffer {
	s := f.startState
	queue := sb.getQueueFor(
		VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
		queueFamilyIndicesToU32Slice(src.Info().QueueFamilyIndices()),
		src.Device(),
		src.LastBoundQueue())
	if queue.IsNil() {
		log.I(ctx, "last bound queue for buffer [%v] is nil", src)
		return VkBuffer(0)
	}

	dev := GetState(s).Queues().Get(queue.VulkanHandle()).Device()
	buffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
		return GetState(s).Buffers().Contains(VkBuffer(x))
	}))
	usageFlags := VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT | VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT)
	deviceMemory := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
		return GetState(s).DeviceMemories().Contains(VkDeviceMemory(x))
	}))
	memoryTypeIndex := f.getBufferMemoryIndex(ctx, GetState(s).Devices().Get(dev))

	sb.write(sb.cb.VkCreateBuffer(
		dev,
		sb.MustAllocReadData(
			NewVkBufferCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
				0,                                       // pNext
				0,                                       // flags
				src.Info().Size(),                       // size
				usageFlags,                              // usage
				VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
				uint32(src.Info().QueueFamilyIndices().Len()),                               // queueFamilyIndexCount
				NewU32ᶜᵖ(sb.MustUnpackReadMap(src.Info().QueueFamilyIndices().All()).Ptr()), // pQueueFamilyIndices
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(buffer).Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkGetBufferMemoryRequirements(
		dev,
		buffer,
		sb.MustAllocWriteData(NewVkMemoryRequirements(sb.ta,
			src.MemoryRequirements().Size(), src.MemoryRequirements().Alignment(), src.MemoryRequirements().MemoryTypeBits())).Ptr(),
	))

	sb.write(sb.cb.VkAllocateMemory(
		dev,
		sb.MustAllocReadData(
			NewVkMemoryAllocateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
				0,                               // pNext
				src.MemoryRequirements().Size(), // allocationSize
				memoryTypeIndex,                 // memoryTypeIndex
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(deviceMemory).Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(
		sb.cb.VkBindBufferMemory(
			dev,
			buffer,
			deviceMemory,
			0,
			VkResult_VK_SUCCESS,
		))
	return buffer
}

func (f *frameLoop) debugCopyImageToBuffer(ctx context.Context, img VkImage, queue VkQueue, sb *stateBuilder) error {
	s := sb.newState

	bufSize := f.getImageSizeforBuffer(img, sb)
	dev := GetState(s).Queues().Get(queue).Device()
	size := nextMultipleOf(bufSize, 256)
	allocSize := bufferAllocationSize(size, 256)
	buffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
		return GetState(s).Buffers().Contains(VkBuffer(x))
	}))
	usageFlags := VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT | VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT)
	deviceMemory := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
		return GetState(s).DeviceMemories().Contains(VkDeviceMemory(x))
	}))
	memoryTypeIndex := f.getBufferMemoryIndex(ctx, GetState(s).Devices().Get(dev))

	sb.write(sb.cb.VkCreateBuffer(
		dev,
		sb.MustAllocReadData(
			NewVkBufferCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
				0,                                       // pNext
				0,                                       // flags
				VkDeviceSize(size),                      // size
				usageFlags,                              // usage
				VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
				0,                                       // queueFamilyIndexCount
				0,                                       // pQueueFamilyIndices
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(buffer).Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkGetBufferMemoryRequirements(
		dev,
		buffer,
		sb.MustAllocWriteData(NewVkMemoryRequirements(sb.ta,
			VkDeviceSize(allocSize), VkDeviceSize(256), 0xFFFFFFFF)).Ptr(),
	))

	sb.write(sb.cb.VkAllocateMemory(
		dev,
		sb.MustAllocReadData(
			NewVkMemoryAllocateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
				0,                  // pNext
				VkDeviceSize(size), // allocationSize
				memoryTypeIndex,    // memoryTypeIndex
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(deviceMemory).Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(
		sb.cb.VkBindBufferMemory(
			dev,
			buffer,
			deviceMemory,
			0,
			VkResult_VK_SUCCESS,
		))

	bufferOffset := VkDeviceSize(0)
	bufImgCopies := []VkBufferImageCopy{}
	imgObj := GetState(f.startState).Images().Get(img)

	preCopyImgBarriers := []VkImageMemoryBarrier{}
	postCopyImgBarriers := []VkImageMemoryBarrier{}

	queueFamily := GetState(s).Queues().Get(queue).Family()
	vkDevice := GetState(s).Queues().Get(queue).Device()
	commandPoolID := VkCommandPool(newUnusedID(false, func(x uint64) bool { ok := GetState(s).CommandPools().Contains(VkCommandPool(x)); return ok }))
	commandBufferID := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { ok := GetState(s).CommandBuffers().Contains(VkCommandBuffer(x)); return ok }))

	sb.write(sb.cb.VkCreateCommandPool(
		vkDevice,
		sb.MustAllocReadData(
			NewVkCommandPoolCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
				NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
				VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
				queueFamily, // queueFamilyIndex
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(commandPoolID).Ptr(),
		VkResult_VK_SUCCESS,
	))

	commandBufferData := sb.MustAllocWriteData(commandBufferID)

	sb.write(sb.cb.VkAllocateCommandBuffers(
		vkDevice,
		sb.MustAllocReadData(NewVkCommandBufferAllocateInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
			NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
			commandPoolID,                                                  // commandPool
			VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
			1, // commandBufferCount
		)).Ptr(),
		commandBufferData.Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkBeginCommandBuffer(
		commandBufferID,
		sb.MustAllocReadData(
			NewVkCommandBufferBeginInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
				0, // pNext
				VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
				0, // pInheritanceInfo
			)).Ptr(),
		VkResult_VK_SUCCESS,
	))

	for aspect, aspectData := range imgObj.Aspects().All() {
		for layer, layerData := range aspectData.Layers().All() {
			for level, levelData := range layerData.Levels().All() {

				levelSize := sb.levelSize(imgObj.Info().Extent(), imgObj.Info().Fmt(), level, aspect)
				bufImgCopy := NewVkBufferImageCopy(sb.ta,
					bufferOffset, // bufferOffset
					0,            // bufferRowLength
					0,            // bufferImageHeight
					NewVkImageSubresourceLayers(sb.ta, // imageSubresource
						VkImageAspectFlags(aspect), // aspectMask
						level,                      // mipLevel
						layer,                      // baseArrayLayer
						1,                          // layerCount
					),
					MakeVkOffset3D(sb.ta), // imageOffset
					NewVkExtent3D(sb.ta,
						uint32(levelSize.width),
						uint32(levelSize.height),
						uint32(levelSize.depth)), // imageExtent
				)
				bufImgCopies = append(bufImgCopies, bufImgCopy)
				bufferOffset += VkDeviceSize(levelSize.levelSize)

				// transfer image layout to VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
				preBarrier := NewVkImageMemoryBarrier(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
					0, // pNext
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
					levelData.Layout(), // oldLayout
					VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // newLayout
					queueFamilyIgnore, // srcQueueFamilyIndex
					queueFamilyIgnore, // dstQueueFamilyIndex
					img,               // image
					NewVkImageSubresourceRange(sb.ta, // subresourceRange
						// ipImageBarrierAspectFlags(aspect, imgObj.Info().Fmt()), // aspectMask
						imgObj.ImageAspect(),
						level, // baseMipLevel
						1,     // levelCount
						layer, // baseArrayLayer
						1,     // layerCount
					),
				)
				preCopyImgBarriers = append(preCopyImgBarriers, preBarrier)

				// transfer the layout back to original
				postBarrier := NewVkImageMemoryBarrier(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
					0, // pNext
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
					VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,                                                         // oldLayout
					levelData.Layout(), // newLayout
					queueFamilyIgnore,  // srcQueueFamilyIndex
					queueFamilyIgnore,  // dstQueueFamilyIndex
					img,                // image
					NewVkImageSubresourceRange(sb.ta, // subresourceRange
						// ipImageBarrierAspectFlags(aspect, imgObj.Info().Fmt()), // aspectMask
						imgObj.ImageAspect(),
						level, // baseMipLevel
						1,     // levelCount
						layer, // baseArrayLayer
						1,     // layerCount
					),
				)
				postCopyImgBarriers = append(postCopyImgBarriers, postBarrier)
			}
		}
	}

	sb.write(sb.cb.VkCmdPipelineBarrier(
		commandBufferID,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		uint32(0),
		memory.Nullptr,
		uint32(1),
		sb.MustAllocReadData(
			NewVkBufferMemoryBarrier(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
				0, // pNext
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
				queueFamilyIgnore,          // srcQueueFamilyIndex
				queueFamilyIgnore,          // dstQueueFamilyIndex
				buffer,                     // buffer
				0,                          // offset
				VkDeviceSize(bufferOffset), // size
			)).Ptr(),
		uint32(len(preCopyImgBarriers)),
		sb.MustAllocReadData(preCopyImgBarriers).Ptr(),
	))

	sb.write(sb.cb.VkCmdCopyImageToBuffer(
		commandBufferID,
		img,
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
		buffer,
		uint32(len(bufImgCopies)),
		sb.MustAllocReadData(bufImgCopies).Ptr(),
	))

	sb.write(sb.cb.VkCmdPipelineBarrier(
		commandBufferID,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		uint32(0),
		memory.Nullptr,
		uint32(0),
		memory.Nullptr,
		uint32(len(postCopyImgBarriers)),
		sb.MustAllocReadData(postCopyImgBarriers).Ptr(),
	))

	sb.write(sb.cb.VkEndCommandBuffer(
		commandBufferID,
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkQueueSubmit(
		queue,
		1,
		sb.MustAllocReadData(NewVkSubmitInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
			0, // pNext
			0, // waitSemaphoreCount
			0, // pWaitSemaphores
			0, // pWaitDstStageMask
			1, // commandBufferCount
			NewVkCommandBufferᶜᵖ(sb.MustAllocReadData(commandBufferID).Ptr()), // pCommandBuffers
			0, // signalSemaphoreCount
			0, // pSignalSemaphores
		)).Ptr(),
		VkFence(0),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkQueueWaitIdle(queue, VkResult_VK_SUCCESS))
	dstObj := GetState(sb.newState).Buffers().Get(buffer)

	// copy dst buffer back
	size = uint64(bufferOffset)
	buflen := uint64(size)
	dstData := sb.MustReserve(buflen)
	ptrDstData := sb.newState.AllocDataOrPanic(sb.ctx, NewVoidᵖ(dstData.Ptr()))
	sb.write(sb.cb.VkMapMemory(
		dstObj.Device(), dstObj.Memory().VulkanHandle(), VkDeviceSize(0), VkDeviceSize(size),
		VkMemoryMapFlags(0), ptrDstData.Ptr(), VkResult_VK_SUCCESS,
	).AddRead(ptrDstData.Data()).AddWrite(ptrDstData.Data()))

	sb.write(sb.cb.VkInvalidateMappedMemoryRanges(
		dstObj.Device(),
		1,
		sb.MustAllocReadData(
			NewVkMappedMemoryRange(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
				0,                              // pNext
				dstObj.Memory().VulkanHandle(), // memory
				VkDeviceSize(0),                // offset
				VkDeviceSize(size),
			)).Ptr(),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		notificationID := b.GetNotificationID()
		log.I(ctx, "register notification for id %d", notificationID)

		b.Notification(notificationID, value.ObservedPointer(dstData.Address()), buflen)
		return b.RegisterNotificationReader(notificationID, func(n gapir.Notification) {
			d := n.GetData()
			id := n.GetId()
			if id != notificationID {
				log.I(ctx, "received incorrect notification id %, expect %d", id, notificationID)
			}
			data := d.GetData()
			name := fmt.Sprintf("VkImage_src_%v_dst_%v_ts_%v", img, buffer, time.Now().UnixNano())
			file, err := os.Create(name)
			defer file.Close()
			if err != nil {
				log.E(ctx, "%v", err)
			}

			byteOrder := s.MemoryLayout.GetEndian()
			r := endian.Reader(bytes.NewReader(data), byteOrder)
			bytes := make([]byte, buflen)
			r.Data(bytes)
			r.Error()
			_, err = file.Write(bytes)
			if err != nil {
				log.E(ctx, "err write file %v", err)
			}
		})
		return nil
	}))
	ptrDstData.Free()
	sb.write(sb.cb.VkUnmapMemory(
		dstObj.Device(),
		dstObj.Memory().VulkanHandle(),
	))

	return nil
}

func (f *frameLoop) getImageSizeforBuffer(img VkImage, sb *stateBuilder) uint64 {
	size := uint64(0)
	imgObj := GetState(f.startState).Images().Get(img)

	for aspect, aspectData := range imgObj.Aspects().All() {
		for _, layerData := range aspectData.Layers().All() {
			for level := range layerData.Levels().All() {

				levelSize := sb.levelSize(imgObj.Info().Extent(), imgObj.Info().Fmt(), level, aspect)
				size += levelSize.levelSize
			}
		}
	}

	return size
}
