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
	backupState    *api.GlobalState
	watcher        *stateWatcher

	bufferCreated   map[VkBuffer]bool
	bufferChanged   map[VkBuffer]bool
	bufferDestroyed map[VkBuffer]BufferObjectʳ
	bufferToBackup  map[VkBuffer]VkBuffer

	imageCreated   map[VkImage]bool
	imageChanged   map[VkImage]bool
	imageDestroyed map[VkImage]bool
	imageToBackup  map[VkImage]VkImage

	fenceChanged     map[VkFence]bool
	eventChanged     map[VkEvent]bool
	semaphoreChanged map[VkSemaphore]bool

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
		bufferDestroyed: make(map[VkBuffer]BufferObjectʳ),
		bufferToBackup:  make(map[VkBuffer]VkBuffer),

		imageCreated:   make(map[VkImage]bool),
		imageChanged:   make(map[VkImage]bool),
		imageDestroyed: make(map[VkImage]bool),
		imageToBackup:  make(map[VkImage]VkImage),

		fenceChanged:     make(map[VkFence]bool),
		eventChanged:     make(map[VkEvent]bool),
		semaphoreChanged: make(map[VkSemaphore]bool),
	}

	f.loopStartCmd, f.loopEndCmd = f.getLoopStartAndEndCmd(ctx, Cmds)

	return f
}

func (f *frameLoop) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	ctx = log.Enter(ctx, "FrameLoop Transform")

	if cmd == f.loopStartCmd {
		log.D(ctx, "Loop: start loop at frame %v, id %v, cmd %v.", f.frameNum, id, cmd)
		f.detectChangedResource(ctx, out.State())

		st := GetState(out.State())
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		defer sb.ta.Dispose()

		if err := f.backupChangedResources(ctx, sb); err != nil {
			log.E(ctx, "Failed to backup changed resources: %v", err)
			return
		}
		sb.scratchRes.Free(sb)
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
		log.D(ctx, "Loop: last frame is %v id %v, cmd is %v.", f.frameNum, id, cmd)
		st := GetState(out.State())
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		defer sb.ta.Dispose()
		out.MutateAndWrite(ctx, id, cmd)

		// Notify this is the end part of the loop to next transformer
		out.NotifyPostLoop(ctx)
		if err := f.resetResource(ctx, sb); err != nil {
			log.E(ctx, "Failed to reset changed resources %v.", err)
			return
		}
		sb.scratchRes.Free(sb)

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
	switch cmd.(type) {
	case *VkQueueSubmit:
		vkCmd := cmd.(*VkQueueSubmit)
		st := GetState(out.State())
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		cmd = f.rewriteQueueSubmit(ctx, sb, vkCmd)

		for _, read := range sb.readMemories {
			cmd.Extras().GetOrAppendObservations().AddRead(read.Data())
		}
		for _, ir := range sb.extraReadIDsAndRanges {
			cmd.Extras().GetOrAppendObservations().AddRead(ir.rng, ir.id)
		}
		for _, write := range sb.writeMemories {
			cmd.Extras().GetOrAppendObservations().AddWrite(write.Data())
		}
		out.MutateAndWrite(ctx, id, cmd)
		for _, read := range sb.readMemories {
			read.Free()
		}
		for _, write := range sb.writeMemories {
			write.Free()
		}
		return
	case *VkQueuePresentKHR:
		f.frameNum++
	}
	out.MutateAndWrite(ctx, id, cmd)

}

func (f *frameLoop) Flush(ctx context.Context, out transform.Writer)    {}
func (f *frameLoop) PreLoop(ctx context.Context, out transform.Writer)  {}
func (f *frameLoop) PostLoop(ctx context.Context, out transform.Writer) {}

// TODO: Find out from which command are the start and the end the loop.
func (f *frameLoop) getLoopStartAndEndCmd(ctx context.Context, Cmds []api.Cmd) (startCmd, endCmd api.Cmd) {
	startCmd = Cmds[f.numInitialCmds]
	endCmd = Cmds[len(Cmds)-1]

	return startCmd, endCmd
}

func (f *frameLoop) detectChangedResource(ctx context.Context, startState *api.GlobalState) {
	f.backupState = f.capture.NewUninitializedState(ctx)
	f.backupState.Memory = startState.Memory.Clone()
	for k, v := range startState.APIs {
		s := v.Clone(f.backupState.Arena)
		s.SetupInitialState(ctx)
		f.backupState.APIs[k] = s
	}
	s := f.backupState

	err := api.ForeachCmd(ctx, f.cmds[f.numInitialCmds:], func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		switch cmd.(type) {
		// Buffers.
		case *VkCreateBuffer:
			vkCmd := cmd.(*VkCreateBuffer)
			vkCmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
			buffer := vkCmd.PBuffer().MustRead(ctx, vkCmd, s, nil)
			f.bufferCreated[buffer] = true
			cmd.Mutate(ctx, id, f.backupState, nil, f.watcher)
		case *VkDestroyBuffer:
			vkCmd := cmd.(*VkDestroyBuffer)
			vkCmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
			buffer := vkCmd.Buffer()
			if _, ok := f.bufferCreated[buffer]; !ok {
				log.D(ctx, "Buffer %v destroyed during loop.", buffer)
				f.bufferDestroyed[buffer] = GetState(startState).Buffers().Get(buffer).Clone(f.backupState.Arena, api.CloneContext{})
				f.bufferChanged[buffer] = true
			}
			cmd.Mutate(ctx, id, f.backupState, nil, f.watcher)

		// Images
		case *VkCreateImage:
			vkCmd := cmd.(*VkCreateImage)
			vkCmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
			img := vkCmd.PImage().MustRead(ctx, vkCmd, s, nil)
			f.imageCreated[img] = true
			cmd.Mutate(ctx, id, f.backupState, nil, f.watcher)
		case *VkDestroyImage:
			vkCmd := cmd.(*VkDestroyImage)
			vkCmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
			img := vkCmd.Image()
			if _, ok := f.imageCreated[img]; !ok {
				log.D(ctx, "Image %v destroyed", img)
				f.imageDestroyed[img] = true
			}
			cmd.Mutate(ctx, id, f.backupState, nil, f.watcher)
		// TODO: Recreate destroyed resources.
		default:
			if err := cmd.Mutate(ctx, id, f.backupState, nil, f.watcher); err != nil {
				return fmt.Errorf("%v: %v: %v", id, cmd, err)
			}
		}

		return nil
	})

	if err != nil {
		log.E(ctx, "Mutate error: [%v].", err)
	}

	st := GetState(f.backupState)
	// Find out changed buffers.
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

	// Find out changed images.
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
							break
						}
					}
				}
			}
		}
	}

	fences := st.Fences().All()
	for k, v := range GetState(startState).Fences().All() {
		if fence, present := fences[k]; present {
			if v.Signaled() != fence.Signaled() {
				log.D(ctx, "Fence %v status changed during loop.", k)
				f.fenceChanged[k] = true
			}
		}
	}

	events := st.Events().All()
	for k, v := range GetState(startState).Events().All() {
		if event, present := events[k]; present {
			if v.Signaled() != event.Signaled() {
				log.D(ctx, "Event %v status changed during loop.", k)
				f.eventChanged[k] = true
			}
		}
	}

	semaphores := st.Semaphores().All()
	for k, v := range GetState(startState).Semaphores().All() {
		if semaphore, present := semaphores[k]; present {
			if v.Signaled() != semaphore.Signaled() {
				log.D(ctx, "Semaphore %v status  changed during loop", k)
				f.semaphoreChanged[k] = true
			}
		}
	}
	// TODO: Find out other changed resources.
}

func (f *frameLoop) backupChangedResources(ctx context.Context, sb *stateBuilder) error {
	if err := f.backupChangedBuffers(ctx, sb); err != nil {
		return err
	}
	if err := f.backupChangedImages(ctx, sb); err != nil {
		return err
	}
	// TODO: Backup other resources.
	return nil
}

func (f *frameLoop) backupChangedBuffers(ctx context.Context, sb *stateBuilder) error {
	s := sb.oldState

	for buffer := range f.bufferChanged {

		log.D(ctx, "Buffer [%v] changed during loop.", buffer)
		bufferObj := GetState(s).Buffers().Get(buffer)

		if bufferObj == NilBufferObjectʳ {
			return log.Err(ctx, nil, "Buffer is nil")
		}
		queue := sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			queueFamilyIndicesToU32Slice(bufferObj.Info().QueueFamilyIndices()),
			bufferObj.Device(),
			bufferObj.LastBoundQueue())
		if queue == NilQueueObjectʳ {
			return log.Err(ctx, nil, "Queue is nil")
		}
		stagingBuffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
			return GetState(s).Buffers().Contains(VkBuffer(x))
		}))

		err := sb.createSameBuffer(bufferObj, stagingBuffer)
		if err != nil {
			return log.Errf(ctx, err, "Create staging buffer for buffer %v failed: %v", buffer)
		}

		tsk := newQueueCommandBatch(
			fmt.Sprintf("Copy buffer: %v", stagingBuffer),
		)

		sb.copyBuffer(buffer, stagingBuffer, queue, tsk)

		if err := tsk.Commit(sb, sb.scratchRes.GetQueueCommandHandler(sb, queue.VulkanHandle())); err != nil {
			return log.Errf(ctx, err, "Copy from buffer %v to %v failed", buffer, stagingBuffer)
		}

		f.bufferToBackup[buffer] = stagingBuffer
	}

	return nil
}

func (f *frameLoop) backupChangedImages(ctx context.Context, sb *stateBuilder) error {
	imgPrimer := newImagePrimer(sb)
	s := GetState(sb.oldState)
	defer imgPrimer.Free()
	for img := range f.imageChanged {
		if _, present := f.imageCreated[img]; present {
			continue
		}
		log.D(ctx, "Image [%v] changed during loop.", img)

		// Create staging Image which is used to backup the changed images
		imgObj := s.Images().Get(img)
		stagingImage, _, err := imgPrimer.CreateSameStagingImage(imgObj)
		if err != nil {
			return log.Err(ctx, err, "Create staging image failed.")
		}
		f.imageToBackup[img] = stagingImage.VulkanHandle()

		if err := f.copyImage(ctx, imgObj, stagingImage, sb); err != nil {
			return log.Err(ctx, err, "Copy image failed")
		}
	}
	return nil
}

func (f *frameLoop) resetResource(ctx context.Context, sb *stateBuilder) error {
	if err := f.resetBuffers(ctx, sb); err != nil {
		return err
	}
	if err := f.resetImages(ctx, sb); err != nil {
		return err
	}
	if err := f.resetFences(ctx, sb); err != nil {
		return err
	}
	if err := f.resetEvents(ctx, sb); err != nil {
		return err
	}
	if err := f.resetSemaphores(ctx, sb); err != nil {
		return err
	}

	//TODO: Reset other resources.
	return nil
}

func (f *frameLoop) recreateDestroyedBuffer(ctx context.Context, sb *stateBuilder, buffer BufferObjectʳ) {
	memReq := NewVkMemoryRequirements(sb.ta,
		buffer.MemoryRequirements().Size(), buffer.MemoryRequirements().Alignment(), buffer.MemoryRequirements().MemoryTypeBits())

	createWithMemReq := sb.cb.VkCreateBuffer(
		buffer.Device(),
		sb.MustAllocReadData(
			NewVkBufferCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
				0,                           // pNext
				buffer.Info().CreateFlags(), // flags
				buffer.Info().Size(),        // size
				VkBufferUsageFlags(uint32(buffer.Info().Usage())|uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT|VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT)), // usage
				buffer.Info().SharingMode(),                                                    // sharingMode
				uint32(buffer.Info().QueueFamilyIndices().Len()),                               // queueFamilyIndexCount
				NewU32ᶜᵖ(sb.MustUnpackReadMap(buffer.Info().QueueFamilyIndices().All()).Ptr()), // pQueueFamilyIndices
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(buffer.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	)
	createWithMemReq.Extras().Add(memReq)
	sb.write(createWithMemReq)
	sb.write(sb.cb.VkGetBufferMemoryRequirements(
		buffer.Device(),
		buffer.VulkanHandle(),
		sb.MustAllocWriteData(memReq).Ptr(),
	))

	mem := buffer.Memory()
	sb.createDeviceMemory(mem, false)
	sb.write(sb.cb.VkBindBufferMemory(
		buffer.Device(),
		buffer.VulkanHandle(),
		mem.VulkanHandle(),
		buffer.MemoryOffset(),
		VkResult_VK_SUCCESS,
	))

}

func (f *frameLoop) rewriteQueueSubmit(ctx context.Context, sb *stateBuilder, cmd *VkQueueSubmit) *VkQueueSubmit {
	s := sb.newState

	cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
	submitCount := cmd.SubmitCount()
	submitInfos := cmd.pSubmits.Slice(0, uint64(submitCount), s.MemoryLayout).MustRead(ctx, cmd, s, nil)

	for i := uint32(0); i < submitCount; i++ {
		si := submitInfos[i]
		cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
		cmdCount := si.CommandBufferCount()

		newCmdBuffers := make([]VkCommandBuffer, cmdCount)
		for j := uint32(0); j < cmdCount; j++ {
			commandbuffer := f.recreateCommandBuffer(ctx, sb, cmdBuffers[j])
			newCmdBuffers[j] = commandbuffer
		}
		submitInfos[i].SetPCommandBuffers(NewVkCommandBufferᶜᵖ(sb.MustAllocReadData(newCmdBuffers).Ptr()))
	}

	return sb.cb.VkQueueSubmit(
		cmd.Queue(),
		submitCount,
		sb.MustAllocReadData(submitInfos).Ptr(),
		cmd.Fence(),
		VkResult_VK_SUCCESS,
	)
}

func (f *frameLoop) recreateCommandBuffer(ctx context.Context, sb *stateBuilder, vkCommandBuffer VkCommandBuffer) VkCommandBuffer {
	s := sb.newState

	commandBuffer := GetState(s).CommandBuffers().Get(vkCommandBuffer)
	commandBufferID, x, cleanup := allocateNewCmdBufFromExistingOneAndBegin(ctx, sb.cb, commandBuffer.VulkanHandle(), s)
	for i := uint32(0); i < uint32(commandBuffer.CommandReferences().Len()); i++ {
		cmd := commandBuffer.CommandReferences().Get(i)
		c, a, _ := AddCommand(ctx, sb.cb, commandBufferID, s, s, GetCommandArgs(ctx, cmd, GetState(s)))
		x = append(x, a)
		cleanup = append(cleanup, c)
	}
	x = append(x,
		sb.cb.VkEndCommandBuffer(commandBufferID, VkResult_VK_SUCCESS))

	for _, cmd := range x {
		sb.write(cmd)
	}
	for _, f := range cleanup {
		f()
	}
	return commandBufferID
}

func (f *frameLoop) resetBuffers(ctx context.Context, sb *stateBuilder) error {

	for buf := range f.bufferCreated {
		log.D(ctx, "Destroy buffer that was created during loop")
		bufObj := GetState(sb.newState).Buffers().Get(buf)
		memID := bufObj.Memory().VulkanHandle()
		if bufObj.Memory().MappedLocation().Address() != 0 {
			sb.write(sb.cb.VkUnmapMemory(bufObj.Device(), memID))
		}
		sb.write(sb.cb.VkDestroyBuffer(bufObj.Device(), buf, memory.Nullptr))
		sb.write(sb.cb.VkFreeMemory(bufObj.Device(), memID, memory.Nullptr))
	}
	for _, buffer := range f.bufferDestroyed {
		log.D(ctx, "Recreate buffer %v that was destroyed during loop", buffer)
		f.recreateDestroyedBuffer(ctx, sb, buffer)
	}

	for dst, src := range f.bufferToBackup {
		bufferObj := GetState(sb.newState).Buffers().Get(src)
		queue := sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			queueFamilyIndicesToU32Slice(bufferObj.Info().QueueFamilyIndices()),
			bufferObj.Device(),
			bufferObj.LastBoundQueue())
		tsk := newQueueCommandBatch(
			fmt.Sprintf("Reset buffer %v", dst),
		)
		sb.copyBuffer(src, dst, queue, tsk)
		if err := tsk.Commit(sb, sb.scratchRes.GetQueueCommandHandler(sb, queue.VulkanHandle())); err != nil {
			return log.Errf(ctx, err, "Reset buffer [%v] with buffer [%v] failed", dst, src)
		}
		log.D(ctx, "Reset buffer [%v] with buffer [%v] succeed", dst, src)
	}

	return nil
}

func (f *frameLoop) resetImages(ctx context.Context, sb *stateBuilder) error {
	if len(f.imageToBackup) == 0 {
		return nil
	}
	imgPrimer := newImagePrimer(sb)
	s := GetState(sb.newState)
	defer imgPrimer.Free()
	for dst, src := range f.imageToBackup {
		dstObj := s.Images().Get(dst)

		prime := func() error {
			primeable, err := imgPrimer.newPrimeableImageDataFromDevice(src, dst)
			if err != nil {
				return log.Errf(ctx, err, "Create primeable image data for image %v", dst)
			}
			defer primeable.free(sb)
			err = primeable.prime(sb, useSpecifiedLayout(dstObj.Info().InitialLayout()), sameLayoutsOfImage(dstObj))
			if err != nil {
				return log.Errf(ctx, err, "Priming image %v with data", dst)
			}
			log.D(ctx, "Prime image from [%v] to [%v] succeed", src, dst)
			return nil
		}

		if err := prime(); err != nil {
			return err
		}
	}

	return nil
}

func (f *frameLoop) resetFences(ctx context.Context, sb *stateBuilder) error {
	s := GetState(sb.newState)

	for k := range f.fenceChanged {
		fence := s.Fences().Get(k)
		if fence.Signaled() {
			pFence := sb.MustAllocReadData(fence.VulkanHandle()).Ptr()
			// Wait fence to be signaled before resetting it.
			sb.write(sb.cb.VkWaitForFences(fence.Device(), 1, pFence, VkBool32(1), 0xFFFFFFFFFFFFFFFF, VkResult_VK_SUCCESS))
			log.D(ctx, "Reset fence %v.", k)
			sb.write(sb.cb.VkResetFences(fence.Device(), 1, pFence, VkResult_VK_SUCCESS))
		} else {
			sb.write(sb.cb.ReplayGetFenceStatus(fence.Device(), fence.VulkanHandle(), VkResult_VK_SUCCESS, VkResult_VK_SUCCESS))
			log.D(ctx, "Singal fence %v.", k)
			queue := sb.getQueueFor(
				VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
				[]uint32{},
				fence.Device(),
				NilQueueObjectʳ)
			if queue == NilQueueObjectʳ {
				return log.Err(ctx, nil, "queue is nil queue")
			}
			sb.write(sb.cb.VkQueueSubmit(
				queue.VulkanHandle(),
				0,
				memory.Nullptr,
				fence.VulkanHandle(),
				VkResult_VK_SUCCESS,
			))

			sb.write(sb.cb.VkQueueWaitIdle(queue.VulkanHandle(), VkResult_VK_SUCCESS))
		}
	}
	return nil
}

func (f *frameLoop) resetEvents(ctx context.Context, sb *stateBuilder) error {
	s := GetState(sb.newState)

	for k := range f.eventChanged {
		event := s.Events().Get(k)
		if event.Signaled() {
			// Wait event to be signaled before resetting it.
			sb.write(sb.cb.ReplayGetEventStatus(event.Device(), event.VulkanHandle(), VkResult_VK_EVENT_SET, true, VkResult_VK_SUCCESS))
			sb.write(sb.cb.VkResetEvent(event.Device(), event.VulkanHandle(), VkResult_VK_SUCCESS))
			log.D(ctx, "Reset event %v ", k)
		} else {
			sb.write(sb.cb.ReplayGetEventStatus(event.Device(), event.VulkanHandle(), VkResult_VK_EVENT_RESET, true, VkResult_VK_SUCCESS))
			sb.write(sb.cb.VkSetEvent(event.Device(), event.VulkanHandle(), VkResult_VK_SUCCESS))
			log.D(ctx, "Set event %v ", k)
		}
	}
	return nil
}

func (f *frameLoop) resetSemaphores(ctx context.Context, sb *stateBuilder) error {
	s := GetState(sb.newState)

	for k := range f.semaphoreChanged {
		semaphore := s.Semaphores().Get(k)
		queue := sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			[]uint32{},
			semaphore.Device(),
			s.Queues().Get(semaphore.LastQueue()))

		if semaphore.Signaled() {
			log.D(ctx, "Wait for semaphore %v to be signaled", semaphore)
			sb.write(sb.cb.VkQueueSubmit(
				queue.VulkanHandle(),
				1,
				sb.MustAllocReadData(NewVkSubmitInfo(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
					0, // pNext
					1, // waitSemaphoreCount
					NewVkSemaphoreᶜᵖ(sb.MustAllocReadData(semaphore.VulkanHandle()).Ptr()), // pWaitSemaphores
					0, // pWaitDstStageMask
					0, // commandBufferCount
					0, // pCommandBuffers
					0, // signalSemaphoreCount
					0, // pSignalSemaphores
				)).Ptr(),
				VkFence(0),
				VkResult_VK_SUCCESS,
			))
		} else {
			log.D(ctx, "Signal semaphore %v", semaphore)
			sb.write(sb.cb.VkQueueSubmit(
				queue.VulkanHandle(),
				1,
				sb.MustAllocReadData(NewVkSubmitInfo(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
					0, // pNext
					0, // waitSemaphoreCount
					0, // pWaitSemaphores
					0, // pWaitDstStageMask
					0, // commandBufferCount
					0, // pCommandBuffers
					1, // signalSemaphoreCount
					NewVkSemaphoreᶜᵖ(sb.MustAllocReadData(semaphore.VulkanHandle()).Ptr()), // pSignalSemaphores
				)).Ptr(),
				VkFence(0),
				VkResult_VK_SUCCESS,
			))
		}
	}
	return nil
}

func (f *frameLoop) copyImage(ctx context.Context, srcImg, dstImg ImageObjectʳ, sb *stateBuilder) error {

	dck, err := ipBuildDeviceCopyKit(sb, srcImg.VulkanHandle(), dstImg.VulkanHandle())
	if err != nil {
		return log.Err(ctx, err, "create ipBuildDeviceCopyKit failed")
	}

	queue := getQueueForPriming(sb, srcImg, VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)
	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, queue.VulkanHandle())
	preCopyBarriers := ipImageLayoutTransitionBarriers(sb, dstImg, useSpecifiedLayout(srcImg.Info().InitialLayout()), useSpecifiedLayout(ipHostCopyImageLayout))
	if err = ipRecordImageMemoryBarriers(sb, queueHandler, preCopyBarriers...); err != nil {
		return log.Err(ctx, err, "Failed at pre device copy image layout transition")
	}

	cmdBatch := dck.BuildDeviceCopyCommands(sb)
	if err = cmdBatch.Commit(sb, queueHandler); err != nil {
		return log.Err(ctx, err, "Failed at commit buffer image copy commands")
	}
	postCopyBarriers := ipImageLayoutTransitionBarriers(sb, dstImg, useSpecifiedLayout(ipHostCopyImageLayout), sameLayoutsOfImage(dstImg))
	if err = ipRecordImageMemoryBarriers(sb, queueHandler, postCopyBarriers...); err != nil {
		return log.Err(ctx, err, "Failed at post device copy image layout transition")
	}

	return nil
}
