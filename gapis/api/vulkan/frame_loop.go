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
func (b *stateWatcher) OnWriteObs(ctx context.Context, observations []api.CmdObservation) {
}
func (b *stateWatcher) OnReadObs(ctx context.Context, observations []api.CmdObservation) {
}
func (b *stateWatcher) OpenForwardDependency(ctx context.Context, dependencyID interface{}) {
}
func (b *stateWatcher) CloseForwardDependency(ctx context.Context, dependencyID interface{}) {
}
func (b *stateWatcher) DropForwardDependency(ctx context.Context, dependencyID interface{}) {
}

// Transfrom
type frameLoop struct {
	capture   *capture.GraphicsCapture
	loopCount int32

	loopStartIdx api.CmdID
	loopEndIdx   api.CmdID

	capturedLoopCmds   []api.Cmd
	capturedLoopCmdIds []api.CmdID

	backupState *api.GlobalState
	watcher     *stateWatcher

	bufferCreated   map[VkBuffer]bool
	bufferChanged   map[VkBuffer]bool
	bufferDestroyed map[VkBuffer]bool
	bufferToBackup  map[VkBuffer]VkBuffer

	imageCreated   map[VkImage]bool
	imageChanged   map[VkImage]bool
	imageDestroyed map[VkImage]bool
	imageToBackup  map[VkImage]VkImage

	loopCountPtr value.Pointer

	frameNum uint32

	loopTerminated      bool
	lastObservedCommand api.CmdID
}

func newFrameLoop(ctx context.Context, graphicsCapture *capture.GraphicsCapture, loopStart api.CmdID, loopEnd api.CmdID, loopCount int32) *frameLoop {

	if api.CmdID.Real(loopStart) >= api.CmdID.Real(loopEnd) {
		log.F(ctx, true, "FrameLoop: Cannot create FrameLoop for zero or negative length loop")
		return nil
	}

	if loopStart == api.CmdNoID || loopEnd == api.CmdNoID {
		log.F(ctx, true, "FrameLoop: Cannot create FrameLoop that starts or ends on api.CmdNoID")
		return nil
	}

	return &frameLoop{

		capture:   graphicsCapture,
		loopCount: loopCount,

		loopStartIdx: api.CmdID.Real(loopStart),
		loopEndIdx:   api.CmdID.Real(loopEnd),

		capturedLoopCmds:   make([]api.Cmd, 0),
		capturedLoopCmdIds: make([]api.CmdID, 0),

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

		loopTerminated:      false,
		lastObservedCommand: api.CmdNoID,
	}
}

func (f *frameLoop) Transform(ctx context.Context, cmdId api.CmdID, cmd api.Cmd, out transform.Writer) {

	ctx = log.Enter(ctx, "FrameLoop Transform")
	log.D(ctx, "FrameLoop: looping from %v to %v.", f.loopStartIdx, f.loopEndIdx)

	// Lets capture and update the last observed frame from f. From this point on use the local lastObservedCommand variable.
	lastObservedCommand := f.lastObservedCommand
	f.lastObservedCommand = cmdId

	if lastObservedCommand != api.CmdNoID && lastObservedCommand > api.CmdID.Real(cmdId) {
		log.F(ctx, true, "FrameLoop: expected next observed command ID to be >= last observed command ID")
	}

	// Walk the frame count forwards if we just hit the end of one.
	if _, ok := cmd.(*VkQueuePresentKHR); ok {
		f.frameNum++
	}

	// Are we before the loop or just at the start of it?
	if lastObservedCommand == api.CmdNoID {

		// This is the start of the loop.
		if api.CmdID.Real(cmdId) >= f.loopStartIdx {

			log.D(ctx, "FrameLoop: start loop at frame %v, cmdId %v, cmd %v.", f.frameNum, cmdId, cmd)

			f.capturedLoopCmds = append(f.capturedLoopCmds, cmd)
			f.capturedLoopCmdIds = append(f.capturedLoopCmdIds, cmdId)

			return

		} else {
			// The current command is before the loop begins and needs no special treatment. Just pass-through.
			log.D(ctx, "FrameLoop: before loop at frame %v, cmdId %v, cmd %v.", f.frameNum, cmdId, cmd)
			out.MutateAndWrite(ctx, cmdId, cmd)
			return
		}

	} else if f.loopTerminated == false { // We're not before or at the start of the loop: thus, are we inside the loop or just at the end of it?

		// This is the end of the loop. We have a lot of deferred things to do.
		if api.CmdID.Real(cmdId) >= f.loopEndIdx {

			if lastObservedCommand == api.CmdNoID {
				log.F(ctx, true, "FrameLoop: Somehow, the FrameLoop ended before it began. Did an earlier transform delete the whole loop? Were your loop indexes realistic?")
			}

			if len(f.capturedLoopCmdIds) != len(f.capturedLoopCmds) {
				log.F(ctx, true, "FrameLoop: Control flow error: Somehow, the number of captured commands and commandIds are not equal.")
			}

			f.loopTerminated = true
			log.D(ctx, "FrameLoop: last frame is %v cmdId %v, cmd is %v.", f.frameNum, cmdId, cmd)

			// Do start loop stuff.
			{
				// Now that we know the complete contents of the loop (only since we've just seen it finish!)...
				// We can finally run over the loop contents looking for resources that have changed.
				// This is required so we can emit extra instructions before the loop capturing the values of
				// anything that we need to restore at the end of the loop. Do that now.
				f.detectChangedResource(ctx, out.State())

				apiState := GetState(out.State())

				stateBuilder := apiState.newStateBuilder(ctx, newTransformerOutput(out))
				defer stateBuilder.ta.Dispose()

				// Back up the resources that change in the loop (as indentified above)
				if err := f.backupChangedResources(ctx, stateBuilder); err != nil {
					log.E(ctx, "FrameLoop: Failed to backup changed resources: %v", err)
					return
				}

				// Write out some custom bytecode for the start of the loop...
				stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
					f.loopCountPtr = b.AllocateMemory(4)
					b.Push(value.S32(f.loopCount))
					b.Store(f.loopCountPtr)
					b.JumpLabel(uint32(0x1))
					return nil
				}))

				// Notify the other transforms that we're about to emit the start of the loop.
				out.NotifyPreLoop(ctx)

				// Write out the command that started all of this loop work.
				out.MutateAndWrite(ctx, f.capturedLoopCmdIds[0], f.capturedLoopCmds[0])
			}

			// Do mid-loop stuff.
			{
				// Iterate through the loop contents, emitting instructions one by one.
				for cmdIndex, cmd := range f.capturedLoopCmds {

					// We've already processed the first loop command, so skip that one.
					// All others get emitted.
					if cmdIndex > 0 {
						out.MutateAndWrite(ctx, f.capturedLoopCmdIds[cmdIndex], cmd)
					}
				}
			}

			// Do end loop stuff.
			{
				apiState := GetState(out.State())

				stateBuilder := apiState.newStateBuilder(ctx, newTransformerOutput(out))
				defer stateBuilder.ta.Dispose()

				// Write out the command that ended the loop. That one is outside of the captured commands since it's the one that called this code.
				out.MutateAndWrite(ctx, cmdId, cmd)

				// Notify the other transforms that we have just emitted the end of the loop.
				out.NotifyPostLoop(ctx)

				// Now we need to emit the instructions to reset the state, before the conditional branch back to the start of the loop.
				if err := f.resetResource(ctx, stateBuilder); err != nil {
					log.E(ctx, "FrameLoop: Failed to reset changed resources %v.", err)
					return
				}

				// Add conditional jump instruction to bring us back to the start of the loop while we've not done.
				stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
					b.Load(protocol.Type_Int32, f.loopCountPtr)
					b.Sub(1)
					b.Clone(0)
					b.Store(f.loopCountPtr)
					b.JumpNZ(uint32(0x1))
					return nil
				}))
			}

			// Finally, we've done all the processing for a loop. Nothing left to do.
			return

		} else { // We're currently inside the loop.

			// Lets just remember the command we've seen so we can do all the work we need at the end of the loop.
			// This is done because the information we need to transform the loop is only available at that time;
			// due to the possibility of preceeding transforms modifing the loop contents in-flight.
			f.capturedLoopCmds = append(f.capturedLoopCmds, cmd)
			f.capturedLoopCmdIds = append(f.capturedLoopCmdIds, cmdId)

			log.D(ctx, "FrameLoop: inside loop at frame %v, cmdId %v, cmd %v.", f.frameNum, cmdId, cmd)

			return
		}

	} else { // We're after the loop. Again, we can simply pass-through commands.
		out.MutateAndWrite(ctx, cmdId, cmd)
		return
	}

	// Should have early out-ed before this point.
	log.F(ctx, true, "FrameLoop: Internal control flow error: Should not be possible to reach this statement.")
}

func (f *frameLoop) Flush(ctx context.Context, out transform.Writer) {

	if f.loopTerminated == false {
		if f.lastObservedCommand == api.CmdNoID {
			log.W(ctx, "FrameLoop transform was applied to whole trace (Flush() has been called) without the loop starting.")
		} else {
			log.E(ctx, "FrameLoop: current frame is %v cmdId %v, cmd is %v.", f.frameNum, f.capturedLoopCmdIds[len(f.capturedLoopCmdIds)-1], f.capturedLoopCmds[len(f.capturedLoopCmds)-1])
			log.F(ctx, true, "FrameLoop transform was applied to whole trace (Flush() has been called) mid loop. Cannot end transformation in this state.")
		}
	}
}

func (f *frameLoop) PreLoop(ctx context.Context, out transform.Writer) {
}
func (f *frameLoop) PostLoop(ctx context.Context, out transform.Writer) {
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
	err := api.ForeachCmd(ctx, f.capturedLoopCmds, func(ctx context.Context, cmdId api.CmdID, cmd api.Cmd) error {

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
