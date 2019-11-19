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

	watcher        *stateWatcher
	loopStartState *api.GlobalState
	loopEndState   *api.GlobalState

	instanceToDestroy map[VkInstance]bool
	instanceToCreate  map[VkInstance]bool

	deviceToDestroy map[VkDevice]bool
	deviceToCreate  map[VkDevice]bool

	memoryToFree     map[VkDeviceMemory]bool
	memoryToAllocate map[VkDeviceMemory]bool
	memoryToUnmap    map[VkDeviceMemory]bool
	memoryToMap      map[VkDeviceMemory]bool
	mappedAddress    map[uint64]value.Pointer

	bufferToDestroy map[VkBuffer]bool
	bufferChanged   map[VkBuffer]bool
	bufferToCreate  map[VkBuffer]bool
	bufferToRestore map[VkBuffer]VkBuffer

	bufferViewToDestroy map[VkBufferView]bool
	bufferViewToCreate  map[VkBufferView]bool

	surfaceToDestroy map[VkSurfaceKHR]bool
	surfaceToCreate  map[VkSurfaceKHR]bool

	swapchainToDestroy map[VkSwapchainKHR]bool
	swapchainToCreate  map[VkSwapchainKHR]bool

	imageToDestroy map[VkImage]bool
	imageChanged   map[VkImage]bool
	imageToCreate  map[VkImage]bool
	imageToRestore map[VkImage]VkImage

	imageViewToDestroy map[VkImageView]bool
	imageViewToCreate  map[VkImageView]bool

	samplerYcbcrConversionToDestroy map[VkSamplerYcbcrConversion]bool
	samplerYcbcrConversionToCreate  map[VkSamplerYcbcrConversion]bool

	samplerToDestroy map[VkSampler]bool
	samplerToCreate  map[VkSampler]bool

	shaderModuleToDestroy map[VkShaderModule]bool
	shaderModuleToCreate  map[VkShaderModule]bool

	descriptorSetLayoutToDestroy map[VkDescriptorSetLayout]bool
	descriptorSetLayoutToCreate  map[VkDescriptorSetLayout]bool

	pipelineLayoutToDestroy map[VkPipelineLayout]bool
	pipelineLayoutToCreate  map[VkPipelineLayout]bool

	pipelineCacheToDestroy map[VkPipelineCache]bool
	pipelineCacheToCreate  map[VkPipelineCache]bool

	pipelineToDestroy        map[VkPipeline]bool
	computePipelineToCreate  map[VkPipeline]bool
	graphicsPipelineToCreate map[VkPipeline]bool

	descriptorPoolToDestroy map[VkDescriptorPool]bool
	descriptorPoolToCreate  map[VkDescriptorPool]bool

	descriptorSetToFree     map[VkDescriptorSet]bool
	descriptorSetToAllocate map[VkDescriptorSet]bool
	descriptorSetChanged    map[VkDescriptorSet]bool
	descriptorSetAutoFreed  map[VkDescriptorSet]bool

	semaphoreToDestroy map[VkSemaphore]bool
	semaphoreChanged   map[VkSemaphore]bool
	semaphoreToCreate  map[VkSemaphore]bool

	fenceToDestroy map[VkFence]bool
	fenceChanged   map[VkFence]bool
	fenceToCreate  map[VkFence]bool

	eventToDestroy map[VkEvent]bool
	eventChanged   map[VkEvent]bool
	eventToCreate  map[VkEvent]bool

	framebufferToDestroy map[VkFramebuffer]bool
	framebufferToCreate  map[VkFramebuffer]bool

	renderPassToDestroy map[VkRenderPass]bool
	renderPassToCreate  map[VkRenderPass]bool

	queryPoolToDestroy map[VkQueryPool]bool
	queryPoolToCreate  map[VkQueryPool]bool

	commandPoolToDestroy map[VkCommandPool]bool
	commandPoolToCreate  map[VkCommandPool]bool

	commandBufferToFree     map[VkCommandBuffer]bool
	commandBufferToAllocate map[VkCommandBuffer]bool
	commandBufferToRecord   map[VkCommandBuffer]bool

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

		instanceToDestroy: make(map[VkInstance]bool),
		instanceToCreate:  make(map[VkInstance]bool),

		deviceToDestroy: make(map[VkDevice]bool),
		deviceToCreate:  make(map[VkDevice]bool),

		memoryToFree:     make(map[VkDeviceMemory]bool),
		memoryToAllocate: make(map[VkDeviceMemory]bool),
		memoryToUnmap:    make(map[VkDeviceMemory]bool),
		memoryToMap:      make(map[VkDeviceMemory]bool),
		mappedAddress:    make(map[uint64]value.Pointer),

		bufferToDestroy: make(map[VkBuffer]bool),
		bufferChanged:   make(map[VkBuffer]bool),
		bufferToCreate:  make(map[VkBuffer]bool),
		bufferToRestore: make(map[VkBuffer]VkBuffer),

		bufferViewToDestroy: make(map[VkBufferView]bool),
		bufferViewToCreate:  make(map[VkBufferView]bool),

		surfaceToDestroy: make(map[VkSurfaceKHR]bool),
		surfaceToCreate:  make(map[VkSurfaceKHR]bool),

		swapchainToDestroy: make(map[VkSwapchainKHR]bool),
		swapchainToCreate:  make(map[VkSwapchainKHR]bool),

		imageToDestroy: make(map[VkImage]bool),
		imageChanged:   make(map[VkImage]bool),
		imageToCreate:  make(map[VkImage]bool),
		imageToRestore: make(map[VkImage]VkImage),

		imageViewToDestroy: make(map[VkImageView]bool),
		imageViewToCreate:  make(map[VkImageView]bool),

		samplerYcbcrConversionToDestroy: make(map[VkSamplerYcbcrConversion]bool),
		samplerYcbcrConversionToCreate:  make(map[VkSamplerYcbcrConversion]bool),

		samplerToDestroy: make(map[VkSampler]bool),
		samplerToCreate:  make(map[VkSampler]bool),

		shaderModuleToDestroy: make(map[VkShaderModule]bool),
		shaderModuleToCreate:  make(map[VkShaderModule]bool),

		descriptorSetLayoutToDestroy: make(map[VkDescriptorSetLayout]bool),
		descriptorSetLayoutToCreate:  make(map[VkDescriptorSetLayout]bool),

		pipelineLayoutToDestroy: make(map[VkPipelineLayout]bool),
		pipelineLayoutToCreate:  make(map[VkPipelineLayout]bool),

		pipelineCacheToDestroy: make(map[VkPipelineCache]bool),
		pipelineCacheToCreate:  make(map[VkPipelineCache]bool),

		pipelineToDestroy:        make(map[VkPipeline]bool),
		computePipelineToCreate:  make(map[VkPipeline]bool),
		graphicsPipelineToCreate: make(map[VkPipeline]bool),

		descriptorPoolToDestroy: make(map[VkDescriptorPool]bool),
		descriptorPoolToCreate:  make(map[VkDescriptorPool]bool),

		descriptorSetToFree:     make(map[VkDescriptorSet]bool),
		descriptorSetToAllocate: make(map[VkDescriptorSet]bool),
		descriptorSetChanged:    make(map[VkDescriptorSet]bool),
		descriptorSetAutoFreed:  make(map[VkDescriptorSet]bool),

		semaphoreToDestroy: make(map[VkSemaphore]bool),
		semaphoreChanged:   make(map[VkSemaphore]bool),
		semaphoreToCreate:  make(map[VkSemaphore]bool),

		fenceToDestroy: make(map[VkFence]bool),
		fenceChanged:   make(map[VkFence]bool),
		fenceToCreate:  make(map[VkFence]bool),

		eventToDestroy: make(map[VkEvent]bool),
		eventChanged:   make(map[VkEvent]bool),
		eventToCreate:  make(map[VkEvent]bool),

		framebufferToDestroy: make(map[VkFramebuffer]bool),
		framebufferToCreate:  make(map[VkFramebuffer]bool),

		renderPassToDestroy: make(map[VkRenderPass]bool),
		renderPassToCreate:  make(map[VkRenderPass]bool),

		queryPoolToDestroy: make(map[VkQueryPool]bool),
		queryPoolToCreate:  make(map[VkQueryPool]bool),

		commandPoolToDestroy: make(map[VkCommandPool]bool),
		commandPoolToCreate:  make(map[VkCommandPool]bool),

		commandBufferToFree:     make(map[VkCommandBuffer]bool),
		commandBufferToAllocate: make(map[VkCommandBuffer]bool),
		commandBufferToRecord:   make(map[VkCommandBuffer]bool),

		loopTerminated:      false,
		lastObservedCommand: api.CmdNoID,
	}
}

func (f *frameLoop) Transform(ctx context.Context, cmdId api.CmdID, cmd api.Cmd, out transform.Writer) {

	ctx = log.Enter(ctx, "FrameLoop Transform")
	log.D(ctx, "FrameLoop: looping from %v to %v. Current CmdID/CmD = %v/%v", f.loopStartIdx, f.loopEndIdx, cmdId, cmd)
	log.D(ctx, "f.loopTerminated = %v, f.lastObservedCommand = %v", f.loopTerminated, f.lastObservedCommand)

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
	if lastObservedCommand == api.CmdNoID || lastObservedCommand < f.loopStartIdx {

		// This is the start of the loop.
		if api.CmdID.Real(cmdId) >= f.loopStartIdx && cmdId != api.CmdNoID {

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
			log.D(ctx, "FrameLoop: end loop at frame %v cmdId %v, cmd is %v.", f.frameNum, cmdId, cmd)

			// Do start loop stuff.
			{
				// Now that we know the complete contents of the loop (only since we've just seen it finish!)...
				// We can finally run over the loop contents looking for resources that have changed.
				// This is required so we can emit extra instructions before the loop capturing the values of
				// anything that we need to restore at the end of the loop. Do that now.
				f.buildStartEndStates(ctx, out.State())
				f.detectChangedResources(ctx)

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
				cb := CommandBuilder{Thread: cmd.Thread(), Arena: out.State().Arena}

				// Iterate through the loop contents, emitting instructions one by one.
				for cmdIndex, cmd := range f.capturedLoopCmds {

					switch cmd.(type) {
					case *VkUnmapMemory:
						vkCmd := cmd.(*VkUnmapMemory)
						mem := vkCmd.Memory()
						if _, ok := f.memoryToMap[mem]; !ok {
							break
						}

						memObj := GetState(out.State()).DeviceMemories().Get(mem)
						addr := memObj.MappedLocation().Address()
						if addr == 0 {
							break
						}

						// Only remember the first mapped target.
						if _, ok := f.mappedAddress[addr]; !ok {
							out.MutateAndWrite(ctx, api.CmdNoID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
								target, err := b.GetMappedTarget(value.ObservedPointer(addr))
								if err == nil {
									f.mappedAddress[addr] = target
								}
								return err
							}))
						}
					}

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

				// Add conditional jump instruction to break us out of the loop if we are done.
				stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
					b.Load(protocol.Type_Int32, f.loopCountPtr)
					b.Sub(1)
					b.Clone(0)
					b.Store(f.loopCountPtr)
					b.JumpZ(uint32(0x2))
					return nil
				}))

				// Now we need to emit the instructions to reset the state, before the conditional branch back to the start of the loop.
				if err := f.resetResources(ctx, stateBuilder); err != nil {
					log.E(ctx, "FrameLoop: Failed to reset changed resources %v.", err)
					return
				}

				// Add unconditional jump instruction to bring us back to the start of the loop if we made it past the conditional break.
				stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
					b.Push(value.S32(1))
					b.JumpNZ(uint32(0x1))
					b.JumpLabel(uint32(0x2))
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

func (f *frameLoop) cloneState(ctx context.Context, startState *api.GlobalState) *api.GlobalState {

	clone := f.capture.NewUninitializedState(ctx)
	clone.Memory = startState.Memory.Clone()

	for apiState, graphicsApi := range startState.APIs {

		clonedState := graphicsApi.Clone(clone.Arena)
		clonedState.SetupInitialState(ctx)

		clone.APIs[apiState] = clonedState
	}

	return clone
}

func (f *frameLoop) buildStartEndStates(ctx context.Context, startState *api.GlobalState) {

	f.loopStartState = f.cloneState(ctx, startState)
	currentState := f.cloneState(ctx, startState)

	// Loop through each command mutating the shadow state and looking at what has been created/destroyed
	err := api.ForeachCmd(ctx, f.capturedLoopCmds, func(ctx context.Context, cmdId api.CmdID, cmd api.Cmd) error {

		cmd.Extras().Observations().ApplyReads(currentState.Memory.ApplicationPool())
		cmd.Extras().Observations().ApplyWrites(currentState.Memory.ApplicationPool())

		switch cmd.(type) {

		// Instances
		case *VkCreateInstance:
			vkCmd := cmd.(*VkCreateInstance)
			instance := vkCmd.PInstance().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Instance %v created.", instance)
			f.instanceToDestroy[instance] = true

		case *VkDestroyInstance:
			vkCmd := cmd.(*VkDestroyInstance)
			instance := vkCmd.Instance()
			log.D(ctx, "Instance %v destroyed.", instance)
			if _, ok := f.instanceToDestroy[instance]; ok {
				delete(f.instanceToDestroy, instance)
			} else {
				f.instanceToCreate[instance] = true
			}

		// Device
		case *VkCreateDevice:
			vkCmd := cmd.(*VkCreateDevice)
			device := vkCmd.PDevice().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Device %v created.", device)
			f.deviceToDestroy[device] = true

		case *VkDestroyDevice:
			vkCmd := cmd.(*VkDestroyDevice)
			device := vkCmd.Device()
			log.D(ctx, "Device %v destroyed.", device)
			if _, ok := f.deviceToDestroy[device]; ok {
				delete(f.deviceToDestroy, device)
			} else {
				f.deviceToCreate[device] = true
			}

		// Memories
		case *VkAllocateMemory:
			vkCmd := cmd.(*VkAllocateMemory)
			mem := vkCmd.PMemory().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Memory %v allocated", mem)
			f.memoryToFree[mem] = true

		case *VkFreeMemory:
			vkCmd := cmd.(*VkFreeMemory)
			mem := vkCmd.Memory()
			log.D(ctx, "Memory %v freed", mem)
			if _, ok := f.memoryToFree[mem]; ok {
				delete(f.memoryToFree, mem)
			} else {
				f.memoryToAllocate[mem] = true
			}

		// Memory mappings
		case *VkMapMemory:
			vkCmd := cmd.(*VkMapMemory)
			mem := vkCmd.Memory()
			log.D(ctx, "Memory %v mapped", mem)
			f.memoryToUnmap[mem] = true

		case *VkUnmapMemory:
			vkCmd := cmd.(*VkUnmapMemory)
			mem := vkCmd.Memory()
			log.D(ctx, "Memory %v unmapped", mem)
			if _, ok := f.memoryToUnmap[mem]; ok {
				delete(f.memoryToUnmap, mem)
			} else {
				f.memoryToMap[mem] = true
			}

		// Buffers.
		case *VkCreateBuffer:
			vkCmd := cmd.(*VkCreateBuffer)
			buffer := vkCmd.PBuffer().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Buffer %v created.", buffer)
			f.bufferToDestroy[buffer] = true

		case *VkDestroyBuffer:
			vkCmd := cmd.(*VkDestroyBuffer)
			buffer := vkCmd.Buffer()
			log.D(ctx, "Buffer %v destroyed.", buffer)
			if _, ok := f.bufferToDestroy[buffer]; ok {
				delete(f.bufferToDestroy, buffer)
			} else {
				f.bufferToCreate[buffer] = true
			}

		// Surfaces
		case *VkCreateXlibSurfaceKHR:
			vkCmd := cmd.(*VkCreateXlibSurfaceKHR)
			surface := vkCmd.PSurface().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Surface %v created", surface)
			f.surfaceToDestroy[surface] = true

		case *VkCreateWaylandSurfaceKHR:
			vkCmd := cmd.(*VkCreateWaylandSurfaceKHR)
			surface := vkCmd.PSurface().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Surface %v created", surface)
			f.surfaceToDestroy[surface] = true

		case *VkCreateWin32SurfaceKHR:
			vkCmd := cmd.(*VkCreateWin32SurfaceKHR)
			surface := vkCmd.PSurface().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Surface %v created", surface)
			f.surfaceToDestroy[surface] = true

		case *VkCreateAndroidSurfaceKHR:
			vkCmd := cmd.(*VkCreateAndroidSurfaceKHR)
			surface := vkCmd.PSurface().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Surface %v created", surface)
			f.surfaceToDestroy[surface] = true

		case *VkCreateDisplayPlaneSurfaceKHR:
			vkCmd := cmd.(*VkCreateDisplayPlaneSurfaceKHR)
			surface := vkCmd.PSurface().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Surface %v created", surface)
			f.surfaceToDestroy[surface] = true

		case *VkDestroySurfaceKHR:
			vkCmd := cmd.(*VkDestroySurfaceKHR)
			surface := vkCmd.Surface()
			log.D(ctx, "Surface %v destroyed", surface)
			if _, ok := f.surfaceToDestroy[surface]; ok {
				delete(f.surfaceToDestroy, surface)
			} else {
				f.surfaceToCreate[surface] = true
			}

		// Swapchains
		case *VkCreateSwapchainKHR:
			vkCmd := cmd.(*VkCreateSwapchainKHR)
			swapchain := vkCmd.PSwapchain().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Swapchain %v created", swapchain)
			f.swapchainToDestroy[swapchain] = true

		case *VkDestroySwapchainKHR:
			vkCmd := cmd.(*VkDestroySwapchainKHR)
			swapchain := vkCmd.Swapchain()
			log.D(ctx, "Swapchain %v destroyed", swapchain)
			if _, ok := f.swapchainToDestroy[swapchain]; ok {
				delete(f.swapchainToDestroy, swapchain)
			} else {
				f.swapchainToCreate[swapchain] = true
			}

		// BufferViews
		case *VkCreateBufferView:
			vkCmd := cmd.(*VkCreateBufferView)
			buffer := vkCmd.PView().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "BuferView %v created", buffer)
			f.bufferViewToDestroy[buffer] = true

		case *VkDestroyBufferView:
			vkCmd := cmd.(*VkDestroyBufferView)
			bufferView := vkCmd.BufferView()
			log.D(ctx, "BufferView %v destroyed", bufferView)
			if _, ok := f.bufferViewToDestroy[bufferView]; ok {
				delete(f.bufferViewToDestroy, bufferView)
			} else {
				f.bufferViewToCreate[bufferView] = true
			}

		// Images
		case *VkCreateImage:
			vkCmd := cmd.(*VkCreateImage)
			img := vkCmd.PImage().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Image %v created", img)
			f.imageToDestroy[img] = true

		case *VkDestroyImage:
			vkCmd := cmd.(*VkDestroyImage)
			img := vkCmd.Image()
			log.D(ctx, "Image %v destroyed", img)
			if _, ok := f.imageToDestroy[img]; ok {
				delete(f.imageToDestroy, img)
			} else {
				f.imageToCreate[img] = true
			}

		// ImageViews
		case *VkCreateImageView:
			vkCmd := cmd.(*VkCreateImageView)
			img := vkCmd.PView().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "ImageView %v created", img)
			f.imageViewToDestroy[img] = true

		case *VkDestroyImageView:
			vkCmd := cmd.(*VkDestroyImageView)
			img := vkCmd.ImageView()
			log.D(ctx, "ImageView %v destroyed", img)
			if _, ok := f.imageViewToDestroy[img]; ok {
				delete(f.imageViewToDestroy, img)
			} else {
				f.imageViewToCreate[img] = true
			}

		// SamplerYcbcrConversion(s)
		case *VkCreateSamplerYcbcrConversion:
			vkCmd := cmd.(*VkCreateSamplerYcbcrConversion)
			samplerYcbcrConversion := vkCmd.PYcbcrConversion().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "SamplerYcbcrConversion %v created", samplerYcbcrConversion)
			f.samplerYcbcrConversionToDestroy[samplerYcbcrConversion] = true

		case *VkDestroySamplerYcbcrConversion:
			vkCmd := cmd.(*VkDestroySamplerYcbcrConversion)
			samplerYcbcrConversion := vkCmd.YcbcrConversion()
			log.D(ctx, "SamplerYcbcrConversion %v destroyed", samplerYcbcrConversion)
			if _, ok := f.samplerYcbcrConversionToDestroy[samplerYcbcrConversion]; ok {
				delete(f.samplerYcbcrConversionToDestroy, samplerYcbcrConversion)
			} else {
				f.samplerYcbcrConversionToCreate[samplerYcbcrConversion] = true
			}

		// Sampler(s)
		case *VkCreateSampler:
			vkCmd := cmd.(*VkCreateSampler)
			sampler := vkCmd.PSampler().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Sampler %v created", sampler)
			f.samplerToDestroy[sampler] = true

		case *VkDestroySampler:
			vkCmd := cmd.(*VkDestroySampler)
			sampler := vkCmd.Sampler()
			log.D(ctx, "Sampler %v destroyed", sampler)
			if _, ok := f.samplerToDestroy[sampler]; ok {
				delete(f.samplerToDestroy, sampler)
			} else {
				f.samplerToCreate[sampler] = true
			}

		// ShaderModule(s)
		case *VkCreateShaderModule:
			vkCmd := cmd.(*VkCreateShaderModule)
			shaderModule := vkCmd.PShaderModule().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "ShaderModule %v created", shaderModule)
			f.shaderModuleToDestroy[shaderModule] = true

		case *VkDestroyShaderModule:
			vkCmd := cmd.(*VkDestroyShaderModule)
			shaderModule := vkCmd.ShaderModule()
			log.D(ctx, "ShaderModule %v destroyed", shaderModule)
			if _, ok := f.shaderModuleToDestroy[shaderModule]; ok {
				delete(f.shaderModuleToDestroy, shaderModule)
			} else {
				f.shaderModuleToCreate[shaderModule] = true
			}

		// DescriptionSetLayout(s)
		case *VkCreateDescriptorSetLayout:
			vkCmd := cmd.(*VkCreateDescriptorSetLayout)
			descriptorSetLayout := vkCmd.PSetLayout().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "DescriptorSetLayout %v created", descriptorSetLayout)
			f.descriptorSetLayoutToDestroy[descriptorSetLayout] = true

		case *VkDestroyDescriptorSetLayout:
			vkCmd := cmd.(*VkDestroyDescriptorSetLayout)
			descriptorSetLayout := vkCmd.DescriptorSetLayout()
			log.D(ctx, "DescriptorSetLayout %v destroyed", descriptorSetLayout)
			if _, ok := f.descriptorSetLayoutToDestroy[descriptorSetLayout]; ok {
				delete(f.descriptorSetLayoutToDestroy, descriptorSetLayout)
			} else {
				f.descriptorSetLayoutToCreate[descriptorSetLayout] = true
			}

		// PipelineLayout(s)
		case *VkCreatePipelineLayout:
			vkCmd := cmd.(*VkCreatePipelineLayout)
			pipelineLayout := vkCmd.PPipelineLayout().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "PipelineLayout %v created", pipelineLayout)
			f.pipelineLayoutToDestroy[pipelineLayout] = true

		case *VkDestroyPipelineLayout:
			vkCmd := cmd.(*VkDestroyPipelineLayout)
			pipelineLayout := vkCmd.PipelineLayout()
			log.D(ctx, "PipelineLayout %v destroyed", pipelineLayout)
			if _, ok := f.pipelineLayoutToDestroy[pipelineLayout]; ok {
				delete(f.pipelineLayoutToDestroy, pipelineLayout)
			} else {
				f.pipelineLayoutToCreate[pipelineLayout] = true
			}

		// PipelineCache(s)
		case *VkCreatePipelineCache:
			vkCmd := cmd.(*VkCreatePipelineCache)
			pipelineCache := vkCmd.PPipelineCache().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "PipelineCache %v created", pipelineCache)
			f.pipelineCacheToDestroy[pipelineCache] = true

		case *VkDestroyPipelineCache:
			vkCmd := cmd.(*VkDestroyPipelineCache)
			pipelineCache := vkCmd.PipelineCache()
			log.D(ctx, "PipelineCache %v destroyed", pipelineCache)
			if _, ok := f.pipelineCacheToDestroy[pipelineCache]; ok {
				delete(f.pipelineCacheToDestroy, pipelineCache)
			} else {
				f.pipelineCacheToCreate[pipelineCache] = true
			}

		// ComputePipelines(s)
		case *VkCreateComputePipelines:
			vkCmd := cmd.(*VkCreateComputePipelines)
			count := vkCmd.CreateInfoCount()
			pipelines := vkCmd.PPipelines().Slice(0, (uint64)(count), startState.MemoryLayout).MustRead(ctx, vkCmd, currentState, nil)
			for index := range pipelines {
				log.D(ctx, "ComputePipeline %v created", pipelines[index])
				f.pipelineToDestroy[pipelines[index]] = true
			}

		// GraphicsPipelines(s)
		case *VkCreateGraphicsPipelines:
			vkCmd := cmd.(*VkCreateGraphicsPipelines)
			count := vkCmd.CreateInfoCount()
			pipelines := vkCmd.PPipelines().Slice(0, (uint64)(count), startState.MemoryLayout).MustRead(ctx, vkCmd, currentState, nil)
			for index := range pipelines {
				log.D(ctx, "GraphicsPipeline %v created", pipelines[index])
				f.pipelineToDestroy[pipelines[index]] = true
			}

		case *VkDestroyPipeline:
			vkCmd := cmd.(*VkDestroyPipeline)
			pipeline := vkCmd.Pipeline()
			log.D(ctx, "Pipeline %v destroyed", pipeline)
			if _, ok := f.pipelineToDestroy[pipeline]; ok {
				delete(f.pipelineToDestroy, pipeline)
			} else {
				isCompute := GetState(currentState).ComputePipelines().Contains(pipeline)
				isGraphics := GetState(currentState).GraphicsPipelines().Contains(pipeline)
				if isCompute {
					f.computePipelineToCreate[pipeline] = true
				} else if isGraphics {
					f.graphicsPipelineToCreate[pipeline] = true
				} else {
					log.E(ctx, "Pipeline %v is of unknown type.", pipeline)
				}
			}

		// DescriptorPool(s)
		case *VkCreateDescriptorPool:
			vkCmd := cmd.(*VkCreateDescriptorPool)
			descriptorPool := vkCmd.PDescriptorPool().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "DescriptorPool %v created", descriptorPool)
			f.descriptorPoolToDestroy[descriptorPool] = true

		case *VkDestroyDescriptorPool:
			vkCmd := cmd.(*VkDestroyDescriptorPool)
			descriptorPool := vkCmd.DescriptorPool()
			log.D(ctx, "DescriptorPool %v destroyed", descriptorPool)
			if _, ok := f.descriptorPoolToDestroy[descriptorPool]; ok {
				delete(f.descriptorPoolToDestroy, descriptorPool)
			} else {
				f.descriptorPoolToCreate[descriptorPool] = true
			}
			descriptorPoolData := GetState(currentState).DescriptorPools().All()[descriptorPool]
			for _, descriptorSetDataValue := range descriptorPoolData.DescriptorSets().All() {
				containedDescriptorSet := descriptorSetDataValue.VulkanHandle()
				f.descriptorSetAutoFreed[containedDescriptorSet] = true
			}

		// DescriptorSet(s)
		case *VkAllocateDescriptorSets:
			vkCmd := cmd.(*VkAllocateDescriptorSets)
			allocInfo := vkCmd.PAllocateInfo().MustRead(ctx, vkCmd, currentState, nil)
			descSetCount := allocInfo.DescriptorSetCount()
			descriptorSets := vkCmd.PDescriptorSets().Slice(0, (uint64)(descSetCount), startState.MemoryLayout).MustRead(ctx, vkCmd, currentState, nil)
			for index := range descriptorSets {
				log.D(ctx, "DescriptorSet %v allocated", descriptorSets[index])
				f.descriptorSetToFree[descriptorSets[index]] = true
			}

		case *VkFreeDescriptorSets:
			vkCmd := cmd.(*VkFreeDescriptorSets)
			descSetCount := vkCmd.DescriptorSetCount()
			descriptorSets := vkCmd.PDescriptorSets().Slice(0, (uint64)(descSetCount), startState.MemoryLayout).MustRead(ctx, vkCmd, currentState, nil)
			for index := range descriptorSets {
				log.D(ctx, "DescriptorSet %v freed", descriptorSets[index])
				if _, ok := f.descriptorSetToFree[descriptorSets[index]]; ok {
					delete(f.descriptorSetToFree, descriptorSets[index])
				} else {
					f.descriptorSetToAllocate[descriptorSets[index]] = true
				}
			}

		// Semaphores
		case *VkCreateSemaphore:
			vkCmd := cmd.(*VkCreateSemaphore)
			sem := vkCmd.PSemaphore().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Semaphore %v is created during loop.", sem)
			f.semaphoreToDestroy[sem] = true

		case *VkDestroySemaphore:
			vkCmd := cmd.(*VkDestroySemaphore)
			sem := vkCmd.Semaphore()
			log.D(ctx, "Semaphore %v is destroyed during loop.", sem)
			if _, ok := f.semaphoreToDestroy[sem]; ok {
				delete(f.semaphoreToDestroy, sem)
			} else {
				f.semaphoreToCreate[sem] = true
			}

		// Fences
		case *VkCreateFence:
			vkCmd := cmd.(*VkCreateFence)
			fence := vkCmd.PFence().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Fence %v is created during loop.", fence)
			f.fenceToDestroy[fence] = true

		case *VkDestroyFence:
			vkCmd := cmd.(*VkDestroyFence)
			fence := vkCmd.Fence()
			log.D(ctx, "Fence %v is destroyed during loop.", fence)
			if _, ok := f.fenceToDestroy[fence]; ok {
				delete(f.fenceToDestroy, fence)
			} else {
				f.fenceToCreate[fence] = true
			}

		// Events
		case *VkCreateEvent:
			vkCmd := cmd.(*VkCreateEvent)
			event := vkCmd.PEvent().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Event %v is created during loop.", event)
			f.eventToDestroy[event] = true

		case *VkDestroyEvent:
			vkCmd := cmd.(*VkDestroyEvent)
			event := vkCmd.Event()
			log.D(ctx, "Event %v is destroyed during loop.", event)
			if _, ok := f.eventToDestroy[event]; ok {
				delete(f.eventToDestroy, event)
			} else {
				f.eventToCreate[event] = true
			}

		// FrameBuffers
		case *VkCreateFramebuffer:
			vkCmd := cmd.(*VkCreateFramebuffer)
			framebuffer := vkCmd.PFramebuffer().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "Framebuffer %v created", framebuffer)
			f.framebufferToDestroy[framebuffer] = true

		case *VkDestroyFramebuffer:
			vkCmd := cmd.(*VkDestroyFramebuffer)
			framebuffer := vkCmd.Framebuffer()
			log.D(ctx, "Framebuffer %v created", framebuffer)
			if _, ok := f.framebufferToDestroy[framebuffer]; ok {
				delete(f.framebufferToDestroy, framebuffer)
			} else {
				f.framebufferToCreate[framebuffer] = true
			}

		// RenderPass(s)
		case *VkCreateRenderPass:
			vkCmd := cmd.(*VkCreateRenderPass)
			renderPass := vkCmd.PRenderPass().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "RenderPass %v created", renderPass)
			f.renderPassToDestroy[renderPass] = true

		case *VkDestroyRenderPass:
			vkCmd := cmd.(*VkDestroyRenderPass)
			renderPass := vkCmd.RenderPass()
			log.D(ctx, "RenderPass %v destroyed", renderPass)
			if _, ok := f.renderPassToDestroy[renderPass]; ok {
				delete(f.renderPassToDestroy, renderPass)
			} else {
				f.renderPassToCreate[renderPass] = true
			}

		// QueryPool(s)
		case *VkCreateQueryPool:
			vkCmd := cmd.(*VkCreateQueryPool)
			queryPool := vkCmd.PQueryPool().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "QueryPool %v created", queryPool)
			f.queryPoolToDestroy[queryPool] = true

		case *VkDestroyQueryPool:
			vkCmd := cmd.(*VkDestroyQueryPool)
			queryPool := vkCmd.QueryPool()
			log.D(ctx, "QueryPool %v destroyed", queryPool)
			if _, ok := f.queryPoolToDestroy[queryPool]; ok {
				delete(f.queryPoolToDestroy, queryPool)
			} else {
				f.queryPoolToCreate[queryPool] = true
			}

		case *VkCmdBeginQuery:
			vkCmd := cmd.(*VkCmdBeginQuery)
			queryPool := vkCmd.QueryPool()
			log.D(ctx, "QueryPool %v began query", queryPool)
			f.queryPoolToDestroy[queryPool] = true
			f.queryPoolToCreate[queryPool] = true

		case *VkCmdEndQuery:
			vkCmd := cmd.(*VkCmdEndQuery)
			queryPool := vkCmd.QueryPool()
			log.D(ctx, "QueryPool %v ended query", queryPool)
			f.queryPoolToDestroy[queryPool] = true
			f.queryPoolToCreate[queryPool] = true

		case *VkCmdWriteTimestamp:
			vkCmd := cmd.(*VkCmdWriteTimestamp)
			queryPool := vkCmd.QueryPool()
			log.D(ctx, "QueryPool %v wrote timestamp", queryPool)
			f.queryPoolToDestroy[queryPool] = true
			f.queryPoolToCreate[queryPool] = true

		case *VkCmdResetQueryPool:
			vkCmd := cmd.(*VkCmdResetQueryPool)
			queryPool := vkCmd.QueryPool()
			log.D(ctx, "QueryPool %v reset", queryPool)
			f.queryPoolToDestroy[queryPool] = true
			f.queryPoolToCreate[queryPool] = true

		// CommandPool(s)
		case *VkCreateCommandPool:
			vkCmd := cmd.(*VkCreateCommandPool)
			commandPool := vkCmd.PCommandPool().MustRead(ctx, vkCmd, currentState, nil)
			log.D(ctx, "CommandPool %v created", commandPool)
			f.commandPoolToDestroy[commandPool] = true

		case *VkDestroyCommandPool:
			vkCmd := cmd.(*VkDestroyCommandPool)
			commandPool := vkCmd.CommandPool()
			log.D(ctx, "CommandPool %v destroyed", commandPool)
			if _, ok := f.commandPoolToDestroy[commandPool]; ok {
				delete(f.commandPoolToDestroy, commandPool)
			} else {
				f.commandPoolToCreate[commandPool] = true
			}

		// Command Buffers
		case *VkAllocateCommandBuffers:
			vkCmd := cmd.(*VkAllocateCommandBuffers)
			cmdBufCount := vkCmd.PAllocateInfo().MustRead(ctx, vkCmd, currentState, nil).CommandBufferCount()
			cmdBuffers := vkCmd.PCommandBuffers().Slice(0, uint64(cmdBufCount), currentState.MemoryLayout).MustRead(ctx, vkCmd, currentState, nil)
			for _, cmdBuf := range cmdBuffers {
				f.commandBufferToFree[cmdBuf] = true
				log.D(ctx, "Command buffer %v allocated.", cmdBuf)
			}

		case *VkFreeCommandBuffers:
			vkCmd := cmd.(*VkFreeCommandBuffers)
			cmdBufCount := vkCmd.CommandBufferCount()
			cmdBufs := vkCmd.PCommandBuffers().Slice(0, uint64(cmdBufCount), currentState.MemoryLayout).MustRead(ctx, cmd, currentState, nil)
			for _, cmdBuf := range cmdBufs {
				log.D(ctx, "Command buffer %v freed.", cmdBufs)
				if _, ok := f.commandBufferToFree[cmdBuf]; ok {
					// The command buffer freed in this call was created during loop, no action needed.
					delete(f.commandBufferToFree, cmdBuf)
				} else {
					// The command buffer freed in this call was not created during loop, need to back up it
					f.commandBufferToAllocate[cmdBuf] = true
				}
			}

		case *VkQueueSubmit:
			vkCmd := cmd.(*VkQueueSubmit)
			submitCount := vkCmd.SubmitCount()
			submitInfos := vkCmd.pSubmits.Slice(0, uint64(submitCount), currentState.MemoryLayout).MustRead(ctx, cmd, currentState, nil)
			for _, si := range submitInfos {
				cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()), currentState.MemoryLayout).MustRead(ctx, cmd, currentState, nil)
				for _, cmdBuf := range cmdBuffers {
					// Re-record all command buffers that are not allocated during the loop for now.
					if _, ok := f.commandBufferToFree[cmdBuf]; !ok {
						f.commandBufferToRecord[cmdBuf] = true
					}
				}
			}
		}

		return cmd.Mutate(ctx, cmdId, currentState, nil, f.watcher)
	})

	if err != nil {
		log.E(ctx, "Mutate error: [%v].", err)
	}

	f.loopEndState = currentState
}

func (f *frameLoop) detectChangedResources(ctx context.Context) {

	f.detectChangedBuffers(ctx)
	f.detectChangedImages(ctx)
	f.detectChangedDescriptorSets(ctx)
	f.detectChangedSemaphores(ctx)
	f.detectChangedFences(ctx)
	f.detectChangedEvents(ctx)

	// TODO: Find out other changed resources.
}

func (f *frameLoop) detectChangedBuffers(ctx context.Context) {

	apiState := GetState(f.loopStartState)

	// Find out changed buffers.
	for bufferKey, buffer := range apiState.Buffers().All() {

		toDestroy := f.bufferToDestroy[buffer.VulkanHandle()]
		toCreate := f.bufferToCreate[buffer.VulkanHandle()]

		if toCreate == true {

			// If we're going to recreate this object for the start of the loop we need to set its state back to the right conditions
			f.bufferChanged[bufferKey] = true
			continue

		} else if toDestroy == true {

			// If we created this object during the loop and we're going to destroy this object at the end of the loop then we don't need to capture the state
			continue

		} else {

			// Otherwise, we'll need to capture this objects state IFF it was modified during the loop.

			data := buffer.Memory().Data()
			span := interval.U64Span{data.Base(), data.Base() + data.Size()}
			poolID := data.Pool()

			// Did we see this buffer get written to during the loop? If we did, then we need to capture the values at the start of the loop.
			// TODO: This code does not handle the possibility of new DeviceMemory being bound to the object during the loop. TODO(purvisa).
			if writes, ok := f.watcher.memoryWrites[poolID]; ok {

				// We do this by comparing the buffer's memory extent with all the observed written areas.
				if _, count := interval.Intersect(writes, span); count != 0 {

					f.bufferChanged[bufferKey] = true
				}
			}
		}
	}
}

func (f *frameLoop) detectChangedImages(ctx context.Context) {

	apiState := GetState(f.loopStartState)

	// Find out changed images.
	for imageKey, image := range apiState.Images().All() {

		// We exempt the frame buffer (swap chain) images from capture.
		if image.IsSwapchainImage() {
			continue
		}

		// Skip the multi-sampled images.
		if image.Info().Samples() != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
			log.W(ctx, "Multi-sampled image %v is not supported for backup/reset.", image)
			continue
		}

		toDestroy := f.imageToDestroy[image.VulkanHandle()]
		toCreate := f.imageToCreate[image.VulkanHandle()]
		if toCreate == true {

			// If we're going to recreate this object for the start of the loop we need to set its state back to the right conditions
			f.imageChanged[image.VulkanHandle()] = true
			continue

		} else if toDestroy == true {

			// If we created this object during the loop and we're going to destroy this object at the end of the loop then we don't need to capture the state
			continue

		} else {

			// Otherwise, we'll need to capture this objects state IFF it was modified during the loop
			// Gotta remember to process all aspects, layers and levels of an image

			for _, imageAspect := range image.Aspects().All() {

				for _, layer := range imageAspect.Layers().All() {

					for _, level := range layer.Levels().All() {

						data := level.Data()
						span := interval.U64Span{data.Base(), data.Base() + data.Size()}
						poolID := data.Pool()

						// Did we see this part of this image get written to during the loop? If we did, then we need to capture the values at the start of the loop.
						// TODO: This code does not handle the possibility of new DeviceMemory being bound to the object during the loop. TODO(purvisa).
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
	}
}

func (f *frameLoop) detectChangedDescriptorSets(ctx context.Context) {

	startState := GetState(f.loopStartState)
	endState := GetState(f.loopEndState)

	for descriptorSetKey, descriptorSetDataAtStart := range startState.descriptorSets.All() {

		descriptorSetDataAtEnd, descriptorExistsOverLoop := endState.descriptorSets.All()[descriptorSetKey]
		_, descriptorExplicitlyDestroyedDuringLoop := f.descriptorSetToAllocate[descriptorSetKey]
		_, descriptorAutoDestroyedDuringLoop := f.descriptorSetAutoFreed[descriptorSetKey]

		descriptorDestroyedDuringLoop := descriptorExplicitlyDestroyedDuringLoop || descriptorAutoDestroyedDuringLoop

		if descriptorExistsOverLoop == true && descriptorDestroyedDuringLoop == false {

			descriptorChanged := false

			descriptorChanged = descriptorChanged || descriptorSetDataAtStart.Device() != descriptorSetDataAtEnd.Device()
			descriptorChanged = descriptorChanged || descriptorSetDataAtStart.Bindings() != descriptorSetDataAtEnd.Bindings()
			descriptorChanged = descriptorChanged || descriptorSetDataAtStart.Layout() != descriptorSetDataAtEnd.Layout()
			descriptorChanged = descriptorChanged || descriptorSetDataAtStart.DebugInfo() != descriptorSetDataAtEnd.DebugInfo()

			if descriptorChanged == true {
				log.D(ctx, "DescriptorSet %v modified", descriptorSetKey)
				f.descriptorSetChanged[descriptorSetKey] = true
			}

		}
	}
}

func (f *frameLoop) detectChangedSemaphores(ctx context.Context) {
	semaphores := GetState(f.loopEndState).Semaphores().All()
	for semaphore, semaphoreStartState := range GetState(f.loopStartState).Semaphores().All() {
		if semaphoreEndState, present := semaphores[semaphore]; present {
			if semaphoreStartState.Signaled() != semaphoreEndState.Signaled() {
				f.semaphoreChanged[semaphore] = true
			}
		}
	}
}

func (f *frameLoop) detectChangedFences(ctx context.Context) {
	fences := GetState(f.loopEndState).Fences().All()
	for fence, fenceStartState := range GetState(f.loopStartState).Fences().All() {
		if fenceEndState, present := fences[fence]; present {
			if fenceStartState.Signaled() != fenceEndState.Signaled() {
				f.fenceChanged[fence] = true
			}
		}
	}
}

func (f *frameLoop) detectChangedEvents(ctx context.Context) {
	events := GetState(f.loopEndState).Events().All()
	for event, eventStartState := range GetState(f.loopStartState).Events().All() {
		if eventEndState, present := events[event]; present {
			if eventStartState.Signaled() != eventEndState.Signaled() {
				f.eventChanged[event] = true
			}
		}
	}
}

func (f *frameLoop) waitDeviceIdle(stateBuilder *stateBuilder) {
	currentState := GetState(stateBuilder.newState)
	for device := range currentState.Devices().All() {
		stateBuilder.write(stateBuilder.cb.VkDeviceWaitIdle(device, VkResult_VK_SUCCESS))
	}
}

func (f *frameLoop) backupChangedResources(ctx context.Context, stateBuilder *stateBuilder) error {

	if err := f.backupChangedBuffers(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.backupChangedImages(ctx, stateBuilder); err != nil {
		return err
	}

	// TODO: Backup other resources.

	// Flush out the backup commands
	stateBuilder.scratchRes.Free(stateBuilder)

	f.waitDeviceIdle(stateBuilder)
	return nil
}

func (f *frameLoop) createStagingBuffer(ctx context.Context, stateBuilder *stateBuilder, src BufferObjectʳ) (VkBuffer, error) {

	stagingBuffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
		return GetState(stateBuilder.newState).Buffers().Contains(VkBuffer(x))
	}))

	mem := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
		return GetState(stateBuilder.newState).DeviceMemories().Contains(VkDeviceMemory(x))
	}))

	bufferObj := src.Clone(GetState(stateBuilder.newState).Arena(), api.CloneContext{})
	usage := VkBufferUsageFlags(uint32(bufferObj.Info().Usage()) | uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT|VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT))
	bufferObj.Info().SetUsage(usage)

	memObj := bufferObj.Memory().Clone(GetState(stateBuilder.newState).Arena(), api.CloneContext{})
	memObj.SetVulkanHandle(mem)
	memObj.SetMappedLocation(Voidᵖ(0))
	memObj.SetMappedOffset(VkDeviceSize(uint64(0)))
	memObj.SetMappedSize(VkDeviceSize(uint64(0)))

	stateBuilder.createDeviceMemory(memObj, false)

	err := stateBuilder.createSameBuffer(bufferObj, stagingBuffer, memObj)

	return stagingBuffer, err
}

func (f *frameLoop) backupChangedBuffers(ctx context.Context, stateBuilder *stateBuilder) error {

	for buffer := range f.bufferChanged {

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

		stagingBuffer, err := f.createStagingBuffer(ctx, stateBuilder, bufferObj)
		if err != nil {
			return err
		}

		task := newQueueCommandBatch(
			fmt.Sprintf("Copy buffer: %v", stagingBuffer),
		)

		stateBuilder.copyBuffer(buffer, stagingBuffer, queue, task)

		if err := task.Commit(stateBuilder, stateBuilder.scratchRes.GetQueueCommandHandler(stateBuilder, queue.VulkanHandle())); err != nil {
			return log.Errf(ctx, err, "Copy from buffer %v to %v failed", buffer, stagingBuffer)
		}

		f.bufferToRestore[buffer] = stagingBuffer
	}

	return nil
}

func (f *frameLoop) backupChangedImages(ctx context.Context, stateBuilder *stateBuilder) error {

	apiState := GetState(stateBuilder.oldState)

	imgPrimer := newImagePrimer(stateBuilder)
	defer imgPrimer.Free()

	for img := range f.imageChanged {

		log.D(ctx, "Image [%v] changed during loop.", img)

		// Create staging Image which is used to backup the changed images
		imgObj := apiState.Images().Get(img).Clone(apiState.Arena(), api.CloneContext{})
		usage := VkImageUsageFlags(uint32(imgObj.Info().Usage()) | uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT|VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT))
		imgObj.Info().SetUsage(usage)

		stagingImage, _, err := imgPrimer.CreateSameStagingImage(imgObj)

		if err != nil {
			return log.Err(ctx, err, "Create staging image failed.")
		}

		f.imageToRestore[img] = stagingImage.VulkanHandle()

		if err := f.copyImage(ctx, imgObj, stagingImage, stateBuilder); err != nil {
			return log.Err(ctx, err, "Copy image failed")
		}
	}

	return nil
}

func (f *frameLoop) resetResources(ctx context.Context, stateBuilder *stateBuilder) error {

	f.waitDeviceIdle(stateBuilder)

	if err := f.resetInstances(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetDevices(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetDeviceMemory(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetBuffers(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetBufferViews(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetSurfaces(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetSwapchains(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetImages(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetImageViews(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetSamplerYcbcrConversions(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetSamplers(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetShaderModules(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetDescriptorSetLayouts(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetPipelineLayouts(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetPipelineCaches(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetPipelines(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetDescriptorPools(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetDescriptorSets(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetSemaphores(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetFences(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetEvents(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetFramebuffers(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetRenderPasses(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetQueryPools(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetCommandPools(ctx, stateBuilder); err != nil {
		return err
	}

	if err := f.resetCommandBuffers(ctx, stateBuilder); err != nil {
		return err
	}

	//TODO: Reset other resources.

	// Flush out the reset commands
	stateBuilder.scratchRes.Free(stateBuilder)

	f.waitDeviceIdle(stateBuilder)

	return nil
}

func (f *frameLoop) resetInstances(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Instance that we need to destroy at the end of the loop...
	for toDestroy := range f.instanceToDestroy {
		// Write the command to delete the created object
		instance := GetState(f.loopEndState).instances.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyInstance(instance.VulkanHandle(), memory.Nullptr))
	}

	// For every Instance that we need to create at the end of the loop...
	for toCreate := range f.instanceToCreate {
		// Write the commands needed to recreate the destroyed object
		instance := GetState(f.loopStartState).instances.Get(toCreate)
		stateBuilder.createInstance(instance.VulkanHandle(), instance)
	}

	for _, deviceObject := range GetState(f.loopStartState).Devices().All() {

		physicalDevice := deviceObject.PhysicalDevice()
		physicalDeviceObject := GetState(f.loopStartState).PhysicalDevices().All()[physicalDevice]
		instance := physicalDeviceObject.Instance()

		if _, ok := f.instanceToCreate[instance]; ok {
			f.deviceToDestroy[deviceObject.VulkanHandle()] = true
			f.deviceToCreate[deviceObject.VulkanHandle()] = true
		}
	}

	for _, surfaceObject := range GetState(f.loopStartState).Surfaces().All() {

		instance := surfaceObject.Instance()

		if _, ok := f.instanceToCreate[instance]; ok {
			f.surfaceToDestroy[surfaceObject.VulkanHandle()] = true
			f.surfaceToCreate[surfaceObject.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetDevices(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Device that we need to destroy at the end of the loop...
	for toDestroy := range f.deviceToDestroy {
		// Write the command to delete the created object
		device := GetState(f.loopEndState).devices.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyDevice(device.VulkanHandle(), memory.Nullptr))
	}

	// For every Device that we need to create at the end of the loop...
	for toCreate := range f.deviceToCreate {
		// Write the commands needed to recreate the destroyed object
		device := GetState(f.loopStartState).devices.Get(toCreate)
		stateBuilder.createDevice(device)
	}

	for _, memoryObject := range GetState(f.loopStartState).DeviceMemories().All() {

		device := memoryObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.memoryToFree[memoryObject.VulkanHandle()] = true
			f.memoryToAllocate[memoryObject.VulkanHandle()] = true
			// TODO: @renfengliu What do we need to do here with memoryToUnmap/memoryToMap/mappedAddress?
		}
	}

	for _, bufferObject := range GetState(f.loopStartState).Buffers().All() {

		device := bufferObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.bufferToDestroy[bufferObject.VulkanHandle()] = true
			f.bufferToCreate[bufferObject.VulkanHandle()] = true
			// TODO: @renfengliu What do we need to do here with bufferChanged/bufferToRestore?
		}
	}

	for _, imageObject := range GetState(f.loopStartState).Images().All() {

		device := imageObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.imageToDestroy[imageObject.VulkanHandle()] = true
			f.imageToCreate[imageObject.VulkanHandle()] = true
			// TODO: @renfengliu What do we need to do here with imageChanged/imageToRestore?
		}
	}

	for _, samplerYcbcrObject := range GetState(f.loopStartState).SamplerYcbcrConversions().All() {

		device := samplerYcbcrObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.samplerYcbcrConversionToDestroy[samplerYcbcrObject.VulkanHandle()] = true
			f.samplerYcbcrConversionToCreate[samplerYcbcrObject.VulkanHandle()] = true
		}
	}

	for _, samplerObject := range GetState(f.loopStartState).Samplers().All() {

		device := samplerObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.samplerToDestroy[samplerObject.VulkanHandle()] = true
			f.samplerToCreate[samplerObject.VulkanHandle()] = true
		}
	}

	for _, shaderModuleObject := range GetState(f.loopStartState).ShaderModules().All() {

		device := shaderModuleObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.shaderModuleToDestroy[shaderModuleObject.VulkanHandle()] = true
			f.shaderModuleToCreate[shaderModuleObject.VulkanHandle()] = true
		}
	}

	for _, descriptorSetLayoutObject := range GetState(f.loopStartState).DescriptorSetLayouts().All() {

		device := descriptorSetLayoutObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.descriptorSetLayoutToDestroy[descriptorSetLayoutObject.VulkanHandle()] = true
			f.descriptorSetLayoutToCreate[descriptorSetLayoutObject.VulkanHandle()] = true
		}
	}

	for _, pipelineLayoutObject := range GetState(f.loopStartState).PipelineLayouts().All() {

		device := pipelineLayoutObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.pipelineLayoutToDestroy[pipelineLayoutObject.VulkanHandle()] = true
			f.pipelineLayoutToCreate[pipelineLayoutObject.VulkanHandle()] = true
		}
	}

	for _, pipelineCacheObject := range GetState(f.loopStartState).PipelineCaches().All() {

		device := pipelineCacheObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.pipelineCacheToDestroy[pipelineCacheObject.VulkanHandle()] = true
			f.pipelineCacheToCreate[pipelineCacheObject.VulkanHandle()] = true
		}
	}

	for _, pipelineObject := range GetState(f.loopStartState).GraphicsPipelines().All() {

		device := pipelineObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.pipelineToDestroy[pipelineObject.VulkanHandle()] = true
			f.graphicsPipelineToCreate[pipelineObject.VulkanHandle()] = true
		}
	}

	for _, pipelineObject := range GetState(f.loopStartState).ComputePipelines().All() {

		device := pipelineObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.pipelineToDestroy[pipelineObject.VulkanHandle()] = true
			f.computePipelineToCreate[pipelineObject.VulkanHandle()] = true
		}
	}

	for _, descriptorPoolObject := range GetState(f.loopStartState).DescriptorPools().All() {

		device := descriptorPoolObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.descriptorPoolToDestroy[descriptorPoolObject.VulkanHandle()] = true
			f.descriptorPoolToCreate[descriptorPoolObject.VulkanHandle()] = true
		}
	}

	for _, descriptorSetObject := range GetState(f.loopStartState).DescriptorSets().All() {

		device := descriptorSetObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.descriptorSetToFree[descriptorSetObject.VulkanHandle()] = true
			f.descriptorSetChanged[descriptorSetObject.VulkanHandle()] = true
			f.descriptorSetToAllocate[descriptorSetObject.VulkanHandle()] = true
		}
	}

	for _, semaphoreObject := range GetState(f.loopStartState).Semaphores().All() {

		device := semaphoreObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.semaphoreToDestroy[semaphoreObject.VulkanHandle()] = true
			f.semaphoreChanged[semaphoreObject.VulkanHandle()] = true
			f.semaphoreToCreate[semaphoreObject.VulkanHandle()] = true
		}
	}

	for _, fenceObject := range GetState(f.loopStartState).Fences().All() {

		device := fenceObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.fenceToDestroy[fenceObject.VulkanHandle()] = true
			f.fenceChanged[fenceObject.VulkanHandle()] = true
			f.fenceToCreate[fenceObject.VulkanHandle()] = true
		}
	}

	for _, eventObject := range GetState(f.loopStartState).Events().All() {

		device := eventObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.eventToDestroy[eventObject.VulkanHandle()] = true
			f.eventChanged[eventObject.VulkanHandle()] = true
			f.eventToCreate[eventObject.VulkanHandle()] = true
		}
	}

	for _, framebufferObject := range GetState(f.loopStartState).Framebuffers().All() {

		device := framebufferObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.framebufferToDestroy[framebufferObject.VulkanHandle()] = true
			f.framebufferToCreate[framebufferObject.VulkanHandle()] = true
		}
	}

	for _, renderPassObject := range GetState(f.loopStartState).RenderPasses().All() {

		device := renderPassObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.renderPassToDestroy[renderPassObject.VulkanHandle()] = true
			f.renderPassToCreate[renderPassObject.VulkanHandle()] = true
		}
	}

	for _, queryPoolObject := range GetState(f.loopStartState).QueryPools().All() {

		device := queryPoolObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.queryPoolToDestroy[queryPoolObject.VulkanHandle()] = true
			f.queryPoolToCreate[queryPoolObject.VulkanHandle()] = true
		}
	}

	for _, commandPoolObject := range GetState(f.loopStartState).CommandPools().All() {

		device := commandPoolObject.Device()

		if _, ok := f.deviceToCreate[device]; ok {
			f.commandPoolToDestroy[commandPoolObject.VulkanHandle()] = true
			f.commandPoolToCreate[commandPoolObject.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetDeviceMemory(ctx context.Context, stateBuilder *stateBuilder) error {

	for mem := range f.memoryToUnmap {
		memObj := GetState(f.loopEndState).DeviceMemories().Get(mem)
		if memObj == NilDeviceMemoryObjectʳ {
			return fmt.Errorf("device memory %s doesn't exist in the loop ending state", mem)
		}
		stateBuilder.write(stateBuilder.cb.VkUnmapMemory(memObj.Device(), mem))
	}

	for mem := range f.memoryToFree {
		log.D(ctx, "Free memory %v which was allocated during loop.", mem)
		memObj := GetState(f.loopEndState).DeviceMemories().Get(mem)
		if memObj == NilDeviceMemoryObjectʳ {
			return fmt.Errorf("device memory %s doesn't exist in the loop ending state", mem)
		}

		stateBuilder.write(stateBuilder.cb.VkFreeMemory(
			memObj.Device(),
			memObj.VulkanHandle(),
			memory.Nullptr,
		))
	}

	for mem := range f.memoryToAllocate {
		log.D(ctx, "Allcate memory %v which was freed during loop.", mem)
		memObj := GetState(f.loopStartState).DeviceMemories().Get(mem)
		if memObj == NilDeviceMemoryObjectʳ {
			return fmt.Errorf("device memory %s doesn't exist in the loop starting state", mem)
		}

		stateBuilder.createDeviceMemory(memObj, false)
	}

	for mem := range f.memoryToMap {
		memObj := GetState(f.loopStartState).DeviceMemories().Get(mem)
		if memObj.MappedLocation().Address() == 0 {
			return fmt.Errorf("device memory %s' mapped address is 0", mem)
		}

		// Memory allocation in state rebuilder will handle the VkMapMemory as well,
		// so if the memory is not recreated, need to call VkMapMemory here
		if _, ok := f.memoryToAllocate[mem]; !ok {
			stateBuilder.write(stateBuilder.cb.VkMapMemory(
				memObj.Device(),
				memObj.VulkanHandle(),
				memObj.MappedOffset(),
				memObj.MappedSize(),
				VkMemoryMapFlags(0),
				NewVoidᵖᵖ(stateBuilder.MustAllocWriteData(memObj.MappedLocation()).Ptr()),
				VkResult_VK_SUCCESS,
			))
		}

		// Handles the remapping of the mapped address.
		stateBuilder.write(stateBuilder.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			addr := memObj.MappedLocation().Address()
			originalTarget, ok := f.mappedAddress[addr]
			if !ok {
				return fmt.Errorf("did not find the original mapped address: %v", addr)
			}

			newTarget, err := b.GetMappedTarget(value.ObservedPointer(addr))
			if err != nil {
				return err
			}
			b.Load(protocol.Type_AbsolutePointer, newTarget)
			b.Store(originalTarget)

			return nil
		}))
	}

	return nil
}

func (f *frameLoop) resetBuffers(ctx context.Context, stateBuilder *stateBuilder) error {

	for buf := range f.bufferToDestroy {
		log.D(ctx, "Destroy buffer %v which was created during loop.", buf)
		bufObj := GetState(stateBuilder.newState).Buffers().Get(buf)
		stateBuilder.write(stateBuilder.cb.VkDestroyBuffer(bufObj.Device(), buf, memory.Nullptr))
	}

	for buf := range f.bufferToCreate {
		log.D(ctx, "Recreate buffer %v which was destroyed during loop.", buf)
		srcBuffer := GetState(f.loopStartState).Buffers().Get(buf)
		mem := GetState(stateBuilder.newState).DeviceMemories().Get(srcBuffer.Memory().VulkanHandle())
		stateBuilder.createSameBuffer(srcBuffer, buf, mem)
	}

	for dst, src := range f.bufferToRestore {

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

	return nil
}

func (f *frameLoop) resetBufferViews(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every BufferView that we need to destroy at the end of the loop...
	for toDestroy := range f.bufferViewToDestroy {
		// Write the command to delete the created object
		bufferView := GetState(f.loopEndState).bufferViews.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyBufferView(bufferView.Device(), bufferView.VulkanHandle(), memory.Nullptr))
	}

	// For every BufferView that we need to create at the end of the loop...
	for toCreate := range f.bufferViewToCreate {
		// Write the commands needed to recreate the destroyed object
		bufferView := GetState(f.loopStartState).bufferViews.Get(toCreate)
		stateBuilder.createBufferView(bufferView)
	}

	return nil
}

func (f *frameLoop) resetSurfaces(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Surface that we need to destroy at the end of the loop...
	for toDestroy := range f.surfaceToDestroy {
		// Write the command to delete the created object
		surface := GetState(f.loopEndState).surfaces.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroySurfaceKHR(surface.Instance(), surface.VulkanHandle(), memory.Nullptr))
	}

	// For every Surface that we need to create at the end of the loop...
	for toCreate := range f.surfaceToCreate {
		// Write the commands needed to recreate the destroyed object
		surface := GetState(f.loopStartState).surfaces.Get(toCreate)
		stateBuilder.createSurface(surface)
	}

	// The shadow state for Surfaces does not contain reference to the Swapchains they are used in. So we have to loop around finding the story.
	for _, swapchainObject := range GetState(f.loopStartState).Swapchains().All() {

		surface := swapchainObject.Surface()
		if surface == NilSurfaceObjectʳ {
			continue
		}

		if _, ok := f.surfaceToCreate[surface.VulkanHandle()]; ok {
			f.swapchainToDestroy[swapchainObject.VulkanHandle()] = true
			f.swapchainToCreate[swapchainObject.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetSwapchains(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Swapchain that we need to destroy at the end of the loop...
	for toDestroy := range f.swapchainToDestroy {
		// Write the command to delete the created object
		swapchain := GetState(f.loopEndState).swapchains.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroySwapchainKHR(swapchain.Device(), swapchain.VulkanHandle(), memory.Nullptr))
	}

	// For every Swapchain that we need to create at the end of the loop...
	for toCreate := range f.swapchainToCreate {
		// Write the commands needed to recreate the destroyed object
		swapchain := GetState(f.loopStartState).swapchains.Get(toCreate)
		stateBuilder.createSwapchain(swapchain)
	}

	return nil
}

func (f *frameLoop) resetImages(ctx context.Context, stateBuilder *stateBuilder) error {

	for toDestroy := range f.imageToDestroy {
		log.D(ctx, "Destroy image %v which was created during loop.", toDestroy)
		image := GetState(f.loopEndState).Images().Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyImage(image.Device(), toDestroy, memory.Nullptr))

		imageViewUsers := image.Views().All()
		for imageView := range imageViewUsers {
			f.imageViewToDestroy[imageView] = true
		}
	}

	if len(f.imageToRestore) == 0 {
		return nil
	}

	apiState := GetState(stateBuilder.newState)

	imgPrimer := newImagePrimer(stateBuilder)
	defer imgPrimer.Free()

	for toCreate := range f.imageToCreate {
		log.D(ctx, "Recreate image %v which was destroyed during loop.", toCreate)
		image := GetState(f.loopStartState).Images().Get(toCreate)
		stateBuilder.createImage(image, f.loopStartState, toCreate)
		// For image creation, the associated image views changes are handled in the restore step below.
	}

	for dst, src := range f.imageToRestore {

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

		// If we (re)created an Image, then we will have invalidated all ImageViews that were using it at the time the loop started.
		// (things using it that were created inside the loop will be automatically recreated anyway so they don't need special treatment here)
		// These ImageViews will need to be (re)created, so add them to the maps to destroy and create in that order.
		imageViewUsers := GetState(f.loopStartState).images.Get(dst).Views().All()
		for imageView := range imageViewUsers {
			f.imageViewToDestroy[imageView] = true
			f.imageViewToCreate[imageView] = true
		}
	}

	return nil
}

func (f *frameLoop) resetImageViews(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every ImageView that we need to destroy at the end of the loop...
	for toDestroy := range f.imageViewToDestroy {
		// Write the command to delete the created object
		imageView := GetState(f.loopEndState).imageViews.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyImageView(imageView.Device(), imageView.VulkanHandle(), memory.Nullptr))
	}

	// For every ImageView that we need to create at the end of the loop...
	for toCreate := range f.imageViewToCreate {
		// Write the commands needed to recreate the destroyed object
		imageView := GetState(f.loopStartState).imageViews.Get(toCreate)
		stateBuilder.createImageView(imageView)

		framebufferUsers := imageView.FramebufferUsers().All()
		for framebuffer := range framebufferUsers {
			f.framebufferToDestroy[framebuffer] = true
			f.framebufferToCreate[framebuffer] = true
		}
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
	srcPreCopyBarriers := ipImageLayoutTransitionBarriers(stateBuilder, srcImg, sameLayoutsOfImage(srcImg), useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL))
	dstPreCopyBarriers := ipImageLayoutTransitionBarriers(stateBuilder, dstImg, sameLayoutsOfImage(dstImg), useSpecifiedLayout(ipHostCopyImageLayout))
	preCopyBarriers := append(srcPreCopyBarriers, dstPreCopyBarriers...)

	if err = ipRecordImageMemoryBarriers(stateBuilder, queueHandler, preCopyBarriers...); err != nil {
		return log.Err(ctx, err, "Failed at pre device copy image layout transition")
	}

	cmdBatch := deviceCopyKit.BuildDeviceCopyCommands(stateBuilder)

	if err = cmdBatch.Commit(stateBuilder, queueHandler); err != nil {
		return log.Err(ctx, err, "Failed at commit buffer image copy commands")
	}

	dstPostCopyBarriers := ipImageLayoutTransitionBarriers(stateBuilder, dstImg, useSpecifiedLayout(ipHostCopyImageLayout), sameLayoutsOfImage(dstImg))
	srcPostCopyBarriers := ipImageLayoutTransitionBarriers(stateBuilder, srcImg, useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL), sameLayoutsOfImage(srcImg))
	postCopyBarriers := append(dstPostCopyBarriers, srcPostCopyBarriers...)
	if err = ipRecordImageMemoryBarriers(stateBuilder, queueHandler, postCopyBarriers...); err != nil {
		return log.Err(ctx, err, "Failed at post device copy image layout transition")
	}

	return nil
}

func (f *frameLoop) resetSamplerYcbcrConversions(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every SamplerYcbcrConversion that we need to destroy at the end of the loop...
	for toDestroy := range f.samplerYcbcrConversionToDestroy {
		// Write the command to delete the created object
		samplerYcbcrConversion := GetState(f.loopEndState).samplerYcbcrConversions.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroySamplerYcbcrConversion(samplerYcbcrConversion.Device(), samplerYcbcrConversion.VulkanHandle(), memory.Nullptr))
	}

	// For every SamplerYcbcrConversion that we need to create at the end of the loop...
	for toCreate := range f.samplerYcbcrConversionToCreate {
		// Write the commands needed to recreate the destroyed object
		samplerYcbcrConversion := GetState(f.loopStartState).samplerYcbcrConversions.Get(toCreate)
		stateBuilder.createSamplerYcbcrConversion(samplerYcbcrConversion)
	}

	// The shadow state for SamplerYcbcrConversions does not contain reference to the Samplers they are used in. So we have to loop around finding the story.
	for _, samplerObject := range GetState(f.loopStartState).Samplers().All() {
		ycbcrConversion := samplerObject.YcbcrConversion()
		if ycbcrConversion == NilSamplerYcbcrConversionObjectʳ {
			log.D(ctx, "Sampler %v doesn't enable ycbcrConversion", samplerObject)
			continue
		}
		if _, ok := f.samplerYcbcrConversionToCreate[ycbcrConversion.VulkanHandle()]; ok {
			f.samplerToDestroy[samplerObject.VulkanHandle()] = true
			f.samplerToCreate[samplerObject.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetSamplers(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Sampler that we need to destroy at the end of the loop...
	for toDestroy := range f.samplerToDestroy {
		// Write the command to delete the created object
		sampler := GetState(f.loopEndState).samplers.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroySampler(sampler.Device(), sampler.VulkanHandle(), memory.Nullptr))
	}

	// For every Sampler that we need to create at the end of the loop...
	for toCreate := range f.samplerToCreate {
		// Write the commands needed to recreate the destroyed object
		sampler := GetState(f.loopStartState).samplers.Get(toCreate)
		stateBuilder.createSampler(sampler)

		// If we (re)created a sampler, then we will have invalidated all descriptor sets that were using it at the time the loop started.
		// (things using it that were created inside the loop will be automatically recreated anyway so they don't need special treatment here)
		// These descriptor sets will need to be (re)created, so add them to the maps to destroy, create and restore state in that order.
		descriptorSetUsers := sampler.DescriptorUsers().All()
		for descriptorSet := range descriptorSetUsers {
			f.descriptorSetToFree[descriptorSet] = true
			f.descriptorSetToAllocate[descriptorSet] = true
			f.descriptorSetChanged[descriptorSet] = true
		}
	}

	return nil
}

func (f *frameLoop) resetShaderModules(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every ShaderModule that we need to destroy at the end of the loop...
	for toDestroy := range f.shaderModuleToDestroy {
		// Write the command to delete the created object
		shaderModule := GetState(f.loopEndState).shaderModules.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyShaderModule(shaderModule.Device(), shaderModule.VulkanHandle(), memory.Nullptr))
	}

	// For every ShaderModule that we need to create at the end of the loop...
	for toCreate := range f.shaderModuleToCreate {
		// Write the commands needed to recreate the destroyed object
		shaderModule := GetState(f.loopStartState).shaderModules.Get(toCreate)
		stateBuilder.createShaderModule(shaderModule)
	}

	// The shadow state for ShaderModules does not contain reference to the ComputePipelines they are used in. So we have to loop around finding the story.
	for _, computePipelineObject := range GetState(f.loopStartState).ComputePipelines().All() {
		shaderModule := computePipelineObject.Stage().Module()
		if _, ok := f.shaderModuleToCreate[shaderModule.VulkanHandle()]; ok {
			f.shaderModuleToDestroy[shaderModule.VulkanHandle()] = true
			f.shaderModuleToCreate[shaderModule.VulkanHandle()] = true
		}
	}

	for _, graphicsPipelineObject := range GetState(f.loopStartState).GraphicsPipelines().All() {
		for _, stage := range graphicsPipelineObject.Stages().All() {
			shaderModule := stage.Module()
			if _, ok := f.shaderModuleToCreate[shaderModule.VulkanHandle()]; ok {
				f.shaderModuleToDestroy[shaderModule.VulkanHandle()] = true
				f.shaderModuleToCreate[shaderModule.VulkanHandle()] = true
			}
		}
	}

	return nil
}

func (f *frameLoop) resetDescriptorSetLayouts(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every DescriptorSetLayout that we need to destroy at the end of the loop...
	for toDestroy := range f.descriptorSetLayoutToDestroy {
		// Write the command to delete the created object
		descSetLay := GetState(f.loopEndState).descriptorSetLayouts.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyDescriptorSetLayout(descSetLay.Device(), descSetLay.VulkanHandle(), memory.Nullptr))
	}

	// For every DescriptorSetLayout that we need to create at the end of the loop...
	for toCreate := range f.descriptorSetLayoutToCreate {
		// Write the commands needed to recreate the destroyed object
		stateBuilder.createDescriptorSetLayout(GetState(f.loopStartState).descriptorSetLayouts.Get(toCreate))
	}

	// The shadow state for DescriptorSetLayouts does not contain reference to the PipelineLayouts they are used in. So we have to loop around finding the story.
	for _, pipelineLayout := range GetState(f.loopStartState).PipelineLayouts().All() {
		for _, descriptorSetLayout := range pipelineLayout.SetLayouts().All() {
			if _, ok := f.descriptorSetLayoutToCreate[descriptorSetLayout.VulkanHandle()]; ok {
				f.pipelineLayoutToDestroy[pipelineLayout.VulkanHandle()] = true
				f.pipelineLayoutToCreate[pipelineLayout.VulkanHandle()] = true
			}
		}
	}

	// The shadow state for DescriptorSetLayouts does not contain reference to DescriptorSets that are created from them. So we have to loop around finding the story.
	for _, descriptorSet := range GetState(f.loopStartState).DescriptorSets().All() {
		descriptorSetLayout := descriptorSet.Layout()
		if _, ok := f.descriptorSetLayoutToCreate[descriptorSetLayout.VulkanHandle()]; ok {
			f.descriptorSetToFree[descriptorSet.VulkanHandle()] = true
			f.descriptorSetToAllocate[descriptorSet.VulkanHandle()] = true
			f.descriptorSetChanged[descriptorSet.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetPipelineLayouts(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every PipelineLayout that we need to destroy at the end of the loop...
	for toDestroy := range f.pipelineLayoutToDestroy {
		// Write the command to delete the created object
		pipelineLayout := GetState(f.loopEndState).pipelineLayouts.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyPipelineLayout(pipelineLayout.Device(), pipelineLayout.VulkanHandle(), memory.Nullptr))
	}

	// For every PipelineLayout that we need to create at the end of the loop...
	for toCreate := range f.pipelineLayoutToCreate {
		// Write the commands needed to recreate the destroyed object
		stateBuilder.createPipelineLayout(GetState(f.loopStartState).pipelineLayouts.Get(toCreate))
	}

	// The shadow state for PipelineLayouts does not contain reference to the ComputePipelines they are used in. So we have to loop around finding the story.
	for _, computePipelineObject := range GetState(f.loopStartState).ComputePipelines().All() {
		pipelineLayout := computePipelineObject.PipelineLayout()
		if _, ok := f.pipelineLayoutToCreate[pipelineLayout.VulkanHandle()]; ok {
			f.pipelineLayoutToDestroy[pipelineLayout.VulkanHandle()] = true
			f.pipelineLayoutToCreate[pipelineLayout.VulkanHandle()] = true
		}
	}

	// The shadow state for PipelineLayouts does not contain reference to the GraphicsPipelines they are used in. So we have to loop around finding the story
	for _, graphicsPipelineObject := range GetState(f.loopStartState).GraphicsPipelines().All() {
		pipelineLayout := graphicsPipelineObject.Layout()
		if _, ok := f.pipelineLayoutToCreate[pipelineLayout.VulkanHandle()]; ok {
			f.pipelineLayoutToDestroy[pipelineLayout.VulkanHandle()] = true
			f.pipelineLayoutToCreate[pipelineLayout.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetPipelines(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Pipeline that we need to destroy at the end of the loop...
	for toDestroy := range f.pipelineToDestroy {

		// Write the command to delete the created object
		computePipeline, isComputePipeline := GetState(f.loopEndState).ComputePipelines().All()[toDestroy]
		graphicsPipeline, isGraphicsPipeline := GetState(f.loopEndState).GraphicsPipelines().All()[toDestroy]

		if isComputePipeline && isGraphicsPipeline {
			log.F(ctx, true, "Control flow error: Pipeline can't be both Graphics and Compute at the same time.")
		}

		if isComputePipeline {
			stateBuilder.write(stateBuilder.cb.VkDestroyPipeline(computePipeline.Device(), computePipeline.VulkanHandle(), memory.Nullptr))
		} else if isGraphicsPipeline {
			stateBuilder.write(stateBuilder.cb.VkDestroyPipeline(graphicsPipeline.Device(), graphicsPipeline.VulkanHandle(), memory.Nullptr))
		} else {
			log.F(ctx, true, "FrameLooping: resetPipelines(): Unknown pipeline type")
		}
	}

	// For every ComputePipeline that we need to create at the end of the loop...
	for toCreate := range f.computePipelineToCreate {
		// Write the commands needed to recreate the destroyed object
		stateBuilder.createComputePipeline(GetState(f.loopStartState).computePipelines.Get(toCreate))
	}

	// For every GraphicsPipeline that we need to create at the end of the loop...
	for toCreate := range f.graphicsPipelineToCreate {
		// Write the commands needed to recreate the destroyed object
		stateBuilder.createGraphicsPipeline(GetState(f.loopStartState).graphicsPipelines.Get(toCreate))
	}

	return nil
}

func (f *frameLoop) resetPipelineCaches(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every PipelineCache that we need to destroy at the end of the loop...
	for toDestroy := range f.pipelineCacheToDestroy {
		// Write the command to delete the created object
		pipelineCache := GetState(f.loopEndState).pipelineCaches.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyPipelineCache(pipelineCache.Device(), pipelineCache.VulkanHandle(), memory.Nullptr))
	}

	// For every PipelineCache that we need to create at the end of the loop...
	for toCreate := range f.pipelineCacheToCreate {
		// Write the commands needed to recreate the destroyed object
		stateBuilder.createPipelineCache(GetState(f.loopStartState).pipelineCaches.Get(toCreate))
	}

	// The shadow state for PipelineCaches does not contain reference to the ComputePipelines they are used in. So we have to loop around finding the story.
	for _, computePipelineObject := range GetState(f.loopStartState).ComputePipelines().All() {
		pipelineCache := computePipelineObject.PipelineCache()
		if pipelineCache == NilPipelineCacheObjectʳ {
			log.D(ctx, "computePipelineObject %v doesn't have a pipeline cache.", computePipelineObject)
			continue
		}
		if _, ok := f.pipelineCacheToCreate[pipelineCache.VulkanHandle()]; ok {
			f.pipelineCacheToDestroy[pipelineCache.VulkanHandle()] = true
			f.pipelineCacheToCreate[pipelineCache.VulkanHandle()] = true
		}
	}

	// The shadow state for PipelineCaches does not contain reference to the GraphicsPipelines they are used in. So we have to loop around finding the story
	for _, graphicsPipelineObject := range GetState(f.loopStartState).GraphicsPipelines().All() {
		pipelineCache := graphicsPipelineObject.PipelineCache()
		if pipelineCache == NilPipelineCacheObjectʳ {
			log.D(ctx, "graphicsPipelineObject %v doesn't have a pipeline cache.", graphicsPipelineObject)
			continue
		}
		if _, ok := f.pipelineCacheToCreate[pipelineCache.VulkanHandle()]; ok {
			f.pipelineCacheToDestroy[pipelineCache.VulkanHandle()] = true
			f.pipelineCacheToCreate[pipelineCache.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetDescriptorPools(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every DescriptorPool that we need to destroy at the end of the loop...
	for toDestroy := range f.descriptorPoolToDestroy {
		// Write the command to delete the created object
		descPool := GetState(f.loopEndState).descriptorPools.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyDescriptorPool(descPool.Device(), descPool.VulkanHandle(), memory.Nullptr))
	}

	// For every DescriptorPool that we need to create at the end of the loop...
	for toCreate := range f.descriptorPoolToCreate {
		// Write the commands needed to recreate the destroyed object
		descPool := GetState(f.loopStartState).DescriptorPools().Get(toCreate)
		stateBuilder.createDescriptorPoolAndAllocateDescriptorSets(descPool)

		// Iterate through all the descriptor sets that we just recreated, adding them to the list of descriptor sets
		// that need to be redefined.
		for _, descriptorSetDataValue := range descPool.DescriptorSets().All() {
			f.descriptorSetChanged[descriptorSetDataValue.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetDescriptorSets(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every DescriptorSet that we need to free at the end of the loop...
	for toDestroy := range f.descriptorSetToFree {
		// Write the command to free the created object
		descSetObj := GetState(f.loopEndState).descriptorSets.Get(toDestroy)
		handle := []VkDescriptorSet{descSetObj.VulkanHandle()}
		stateBuilder.write(stateBuilder.cb.VkFreeDescriptorSets(descSetObj.Device(),
			descSetObj.DescriptorPool(),
			1,
			stateBuilder.MustAllocReadData(handle).Ptr(),
			VkResult_VK_SUCCESS))
	}

	// For every DescriptorSet that we need to create at the end of the loop...
	for toCreate := range f.descriptorSetToAllocate {
		// Write the commands needed to reallocate the freed object
		descSetObj := GetState(f.loopStartState).descriptorSets.Get(toCreate)
		descPoolObj := GetState(f.loopStartState).descriptorPools.Get(descSetObj.DescriptorPool())

		descSetHandles := []VkDescriptorSet{descSetObj.VulkanHandle()}
		descSetLayoutHandles := []VkDescriptorSetLayout{descSetObj.Layout().VulkanHandle()}
		stateBuilder.allocateDescriptorSets(descPoolObj, descSetHandles, descSetLayoutHandles)
	}

	// For every DescriptorSet that was modified during the loop...
	for changed := range f.descriptorSetChanged {
		// Write the commands needed to restore the modified object
		descSetObj := GetState(f.loopStartState).descriptorSets.Get(changed)
		stateBuilder.writeDescriptorSet(descSetObj)
	}

	return nil
}

func (f *frameLoop) resetSemaphores(ctx context.Context, stateBuilder *stateBuilder) error {

	for sem := range f.semaphoreToDestroy {
		semObj := GetState(f.loopEndState).Semaphores().Get(sem)
		if semObj != NilSemaphoreObjectʳ {
			log.D(ctx, "Destroy semaphore %v which was created during loop.", sem)
			stateBuilder.write(stateBuilder.cb.VkDestroySemaphore(semObj.Device(), semObj.VulkanHandle(), memory.Nullptr))
		} else {
			log.E(ctx, "Semaphore %v cannot be found in the loop ending state", sem)
		}
	}

	for sem := range f.semaphoreToCreate {
		semObj := GetState(f.loopStartState).Semaphores().Get(sem)
		if semObj != NilSemaphoreObjectʳ {
			log.D(ctx, "Create semaphore %v which was destroyed during loop.", sem)
			stateBuilder.createSemaphore(semObj)
		} else {
			log.E(ctx, "Semaphore %v cannot be found in the loop starting state", sem)
		}
	}

	for sem := range f.semaphoreChanged {
		if _, ok := f.semaphoreToDestroy[sem]; ok {
			continue
		}

		if _, ok := f.semaphoreToCreate[sem]; ok {
			continue
		}

		semObj := GetState(f.loopEndState).Semaphores().Get(sem)
		if semObj == NilSemaphoreObjectʳ {
			log.E(ctx, "Semaphore %v cannot be found in the loop ending state", sem)
			continue
		}
		queue := stateBuilder.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			[]uint32{},
			semObj.Device(),
			GetState(f.loopEndState).Queues().Get(semObj.LastQueue()))

		if semObj.Signaled() {
			// According to vulkan spec:
			// "The act of waiting for a semaphore also unsignals that semaphore. Applications must ensure that
			// between two such wait operations, the semaphore is signaled again, with execution dependencies
			// used to ensure these occur in order. Semaphore waits and signals should thus occur in discrete 1:1 pairs."
			// So there's no need to wait for it be signalled here. And add additional waiting here may break the 1:1 waits and signals pairs.
		} else {
			log.D(ctx, "Signal semaphore %v.", sem)
			stateBuilder.write(stateBuilder.cb.VkQueueSubmit(
				queue.VulkanHandle(),
				1,
				stateBuilder.MustAllocReadData(NewVkSubmitInfo(stateBuilder.ta,
					VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
					0, // pNext
					0, // waitSemaphoreCount
					0, // pWaitSemaphores
					0, // pWaitDstStageMask
					0, // commandBufferCount
					0, // pCommandBuffers
					1, // signalSemaphoreCount
					NewVkSemaphoreᶜᵖ(stateBuilder.MustAllocReadData(semObj.VulkanHandle()).Ptr()), // pSignalSemaphores
				)).Ptr(),
				VkFence(0),
				VkResult_VK_SUCCESS,
			))
		}
	}
	return nil
}

func (f *frameLoop) resetFences(ctx context.Context, stateBuilder *stateBuilder) error {

	for fence := range f.fenceToDestroy {
		fenceObj := GetState(f.loopEndState).Fences().Get(fence)
		if fenceObj != NilFenceObjectʳ {
			log.D(ctx, "Destroy fence: %v which was created during loop.", fence)
			stateBuilder.write(stateBuilder.cb.VkDestroyFence(fenceObj.Device(), fenceObj.VulkanHandle(), memory.Nullptr))
		} else {
			log.E(ctx, "Fence %v cannot be found in the loop ending state", fence)
		}
	}

	for fence := range f.fenceToCreate {
		fenceObj := GetState(f.loopStartState).Fences().Get(fence)
		if fenceObj != NilFenceObjectʳ {
			log.D(ctx, "Create fence %v which was destroyed during loop.", fence)
			stateBuilder.createFence(fenceObj)
		} else {
			log.E(ctx, "Fence %v cannot be found in the loop starting state", fence)
		}
	}

	for fence := range f.fenceChanged {
		if _, ok := f.fenceToDestroy[fence]; ok {
			continue
		}

		if _, ok := f.fenceToCreate[fence]; ok {
			continue
		}

		fenceObj := GetState(f.loopEndState).Fences().Get(fence)
		if fenceObj == NilFenceObjectʳ {
			log.E(ctx, "Fence %v cannot be found in the loop ending state", fence)
			continue
		}

		if fenceObj.Signaled() {
			pFence := stateBuilder.MustAllocReadData(fenceObj.VulkanHandle()).Ptr()
			// Wait fence to be signaled before resetting it.
			stateBuilder.write(stateBuilder.cb.VkWaitForFences(fenceObj.Device(), 1, pFence, VkBool32(1), 0xFFFFFFFFFFFFFFFF, VkResult_VK_SUCCESS))
			log.D(ctx, "Reset fence %v.", fence)
			stateBuilder.write(stateBuilder.cb.VkResetFences(fenceObj.Device(), 1, pFence, VkResult_VK_SUCCESS))
		} else {
			log.D(ctx, "Singal fence %v.", fence)
			queue := stateBuilder.getQueueFor(
				VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
				[]uint32{},
				fenceObj.Device(),
				NilQueueObjectʳ)
			if queue == NilQueueObjectʳ {
				return log.Err(ctx, nil, "queue is nil queue")
			}
			stateBuilder.write(stateBuilder.cb.VkQueueSubmit(
				queue.VulkanHandle(),
				0,
				memory.Nullptr,
				fenceObj.VulkanHandle(),
				VkResult_VK_SUCCESS,
			))

			stateBuilder.write(stateBuilder.cb.VkQueueWaitIdle(queue.VulkanHandle(), VkResult_VK_SUCCESS))
		}
	}
	return nil
}

func (f *frameLoop) resetEvents(ctx context.Context, stateBuilder *stateBuilder) error {

	for event := range f.eventToDestroy {
		eventObj := GetState(f.loopEndState).Events().Get(event)
		if eventObj != NilEventObjectʳ {
			log.D(ctx, "Destroy event: %v which was created during loop.", event)
			stateBuilder.write(stateBuilder.cb.VkDestroyEvent(eventObj.Device(), eventObj.VulkanHandle(), memory.Nullptr))
		} else {
			log.E(ctx, "Event %v cannot be found in loop ending state.", event)
		}
	}

	for event := range f.eventToCreate {
		eventObj := GetState(f.loopStartState).Events().Get(event)
		if eventObj != NilEventObjectʳ {
			log.D(ctx, "Create event %v which was destroyed during loop.", event)
			stateBuilder.createEvent(eventObj)
		} else {
			log.E(ctx, "Event %v cannot be found in loop starting state.", event)
		}
	}

	for event := range f.eventChanged {
		if _, ok := f.eventToDestroy[event]; ok {
			continue
		}

		if _, ok := f.eventToCreate[event]; ok {
			continue
		}

		eventObj := GetState(f.loopEndState).Events().Get(event)
		if eventObj == NilEventObjectʳ {
			log.E(ctx, "Event %v cannot be found in loop ending state.", event)
			continue
		}
		if eventObj.Signaled() {
			log.D(ctx, "Reset event %v ", event)
			// Wait event to be signaled before resetting it.
			stateBuilder.write(stateBuilder.cb.ReplayGetEventStatus(eventObj.Device(), eventObj.VulkanHandle(), VkResult_VK_EVENT_SET, true, VkResult_VK_SUCCESS))
			stateBuilder.write(stateBuilder.cb.VkResetEvent(eventObj.Device(), eventObj.VulkanHandle(), VkResult_VK_SUCCESS))
		} else {
			log.D(ctx, "Set event %v ", event)
			stateBuilder.write(stateBuilder.cb.ReplayGetEventStatus(eventObj.Device(), eventObj.VulkanHandle(), VkResult_VK_EVENT_RESET, true, VkResult_VK_SUCCESS))
			stateBuilder.write(stateBuilder.cb.VkSetEvent(eventObj.Device(), eventObj.VulkanHandle(), VkResult_VK_SUCCESS))
		}
	}

	return nil
}

func (f *frameLoop) resetFramebuffers(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every Framebuffers that we need to destroy at the end of the loop...
	for toDestroy := range f.framebufferToDestroy {
		// Write the command to delete the created object
		framebuffer := GetState(f.loopEndState).framebuffers.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyFramebuffer(framebuffer.Device(), framebuffer.VulkanHandle(), memory.Nullptr))
	}

	// For every Framebuffers that we need to create at the end of the loop...
	for toCreate := range f.framebufferToCreate {
		// Write the commands needed to recreate the destroyed object
		framebuffer := GetState(f.loopStartState).framebuffers.Get(toCreate)
		stateBuilder.createFramebuffer(framebuffer)
	}

	return nil
}

func (f *frameLoop) resetRenderPasses(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every RenderPass that we need to destroy at the end of the loop...
	for toDestroy := range f.renderPassToDestroy {
		// Write the command to delete the created object
		renderPass := GetState(f.loopEndState).renderPasses.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyRenderPass(renderPass.Device(), renderPass.VulkanHandle(), memory.Nullptr))
	}

	// For every RenderPass that we need to create at the end of the loop...
	for toCreate := range f.renderPassToCreate {
		// Write the commands needed to recreate the destroyed object
		renderPass := GetState(f.loopStartState).renderPasses.Get(toCreate)
		stateBuilder.createRenderPass(renderPass)
	}

	return nil
}

func (f *frameLoop) resetQueryPools(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every QueryPools that we need to destroy at the end of the loop...
	for toDestroy := range f.queryPoolToDestroy {
		// Write the command to delete the created object
		queryPool := GetState(f.loopEndState).queryPools.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyQueryPool(queryPool.Device(), queryPool.VulkanHandle(), memory.Nullptr))
	}

	// For every QueryPools that we need to create at the end of the loop...
	for toCreate := range f.queryPoolToCreate {
		// Write the commands needed to recreate the destroyed object
		queryPool := GetState(f.loopStartState).queryPools.Get(toCreate)
		stateBuilder.createQueryPool(queryPool)
	}

	return nil
}

func (f *frameLoop) resetCommandPools(ctx context.Context, stateBuilder *stateBuilder) error {

	// For every CommandPool that we need to destroy at the end of the loop...
	for toDestroy := range f.commandPoolToDestroy {
		// Write the command to delete the created object
		commandPool := GetState(f.loopEndState).commandPools.Get(toDestroy)
		stateBuilder.write(stateBuilder.cb.VkDestroyCommandPool(commandPool.Device(), commandPool.VulkanHandle(), memory.Nullptr))
	}

	// For every CommandPool that we need to create at the end of the loop...
	for toCreate := range f.commandPoolToCreate {
		// Write the commands needed to recreate the destroyed object
		commandPool := GetState(f.loopStartState).commandPools.Get(toCreate)
		stateBuilder.createCommandPool(commandPool)

		// Iterate through all the command pools that we just recreated, adding them to the list of command buffers
		// that need to be redefined.
		for _, commandSetDataValue := range commandPool.CommandBuffers().All() {
			delete(f.commandBufferToFree, commandSetDataValue.VulkanHandle())
			f.commandBufferToAllocate[commandSetDataValue.VulkanHandle()] = true
			f.commandBufferToRecord[commandSetDataValue.VulkanHandle()] = true
		}
	}

	return nil
}

func (f *frameLoop) resetCommandBuffers(ctx context.Context, stateBuilder *stateBuilder) error {

	for cmdBuf := range f.commandBufferToFree {
		log.D(ctx, "Command buffer %v allocated during loop, free it.", cmdBuf)
		cmdBufObj := GetState(f.loopEndState).CommandBuffers().Get(cmdBuf)
		if cmdBufObj != NilCommandBufferObjectʳ {
			stateBuilder.write(stateBuilder.cb.VkFreeCommandBuffers(
				cmdBufObj.Device(),
				cmdBufObj.Pool(),
				1,
				stateBuilder.MustAllocReadData(cmdBufObj.VulkanHandle()).Ptr(),
			))
		} else {
			log.F(ctx, true, "Command buffer %v cannot be found in loop ending state", cmdBuf)
		}
	}

	for cmdBuf := range f.commandBufferToAllocate {
		cmdBufObj := GetState(f.loopStartState).CommandBuffers().Get(cmdBuf)
		if cmdBufObj == NilCommandBufferObjectʳ {
			log.F(ctx, true, "Command buffer %v can not be found in loop starting state", cmdBuf)
			continue
		}
		log.D(ctx, "Command buffer %v freed during loop, recreate it.", cmdBuf)
		stateBuilder.createCommandBuffer(cmdBufObj, cmdBufObj.Level())
	}

	for cmdBuf := range f.commandBufferToRecord {
		cmdBufObj := GetState(f.loopStartState).CommandBuffers().Get(cmdBuf)
		if cmdBufObj == NilCommandBufferObjectʳ {
			log.F(ctx, true, "Command buffer %v can not be found in loop starting state", cmdBuf)
		}
		log.D(ctx, "Command buffer %v changed during loop, re-record it.", cmdBuf)
		stateBuilder.write(stateBuilder.cb.VkResetCommandBuffer(
			cmdBufObj.VulkanHandle(),
			0,
			VkResult_VK_SUCCESS,
		))
		stateBuilder.recordCommandBuffer(cmdBufObj, cmdBufObj.Level(), f.loopStartState)
	}

	return nil
}
