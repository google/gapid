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
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type timestampRecord struct {
	timestamp service.TimestampsItem
	// Is this timestamp from the last command in the commandbuffer
	IsEoC bool
}

type queryResults []timestampRecord

// queryPoolInfo contains the information about the query pool
type queryPoolInfo struct {
	queryPool     VkQueryPool
	queryPoolSize uint32
	device        VkDevice
	queue         VkQueue
	queryCount    uint32
	results       queryResults
}

// commandPoolKey is used to find a command pool suitable for a specific queue family
type commandPoolKey struct {
	device      VkDevice
	queueFamily uint32
}

type queryTimestamps struct {
	endOfReplay
	commandPools    map[commandPoolKey]VkCommandPool
	queryPools      map[VkQueue]*queryPoolInfo
	timestampPeriod float32
	replayResult    []replay.Result
	handler         service.TimeStampsHandler
	results         map[uint64]queryResults
	allocations     *allocationTracker
}

func newQueryTimestamps(ctx context.Context, handler service.TimeStampsHandler) *queryTimestamps {
	transform := &queryTimestamps{
		commandPools: make(map[commandPoolKey]VkCommandPool),
		queryPools:   make(map[VkQueue]*queryPoolInfo),
		handler:      handler,
		results:      make(map[uint64]queryResults),
		allocations:  nil,
	}
	return transform
}

func (timestampTransform *queryTimestamps) RequiresAccurateState() bool {
	return false
}

func (timestampTransform *queryTimestamps) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	timestampTransform.allocations = NewAllocationTracker(inputState)
	return nil
}

func (timestampTransform *queryTimestamps) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	cmds := make([]api.Cmd, 0)
	cb := CommandBuilder{Thread: 0}

	for _, queryPoolInfo := range timestampTransform.queryPools {
		queryCommands := timestampTransform.getQueryResults(ctx, cb, inputState, queryPoolInfo)
		cmds = append(cmds, queryCommands...)
	}

	cleanupCmds := timestampTransform.cleanup(ctx, inputState)
	cmds = append(cmds, cleanupCmds...)

	notifyCmd := timestampTransform.CreateNotifyInstruction(ctx, func() interface{} {
		return timestampTransform.replayResult
	})
	cmds = append(cmds, notifyCmd)

	return cmds, nil
}

func (timestampTransform *queryTimestamps) ClearTransformResources(ctx context.Context) {
	timestampTransform.allocations.FreeAllocations()
}

func (timestampTransform *queryTimestamps) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	outputCmds := make([]api.Cmd, 0, len(inputCommands))

	for i, cmd := range inputCommands {
		vkQueueSubmitCmd, ok := cmd.(*VkQueueSubmit)
		if !ok {
			outputCmds = append(outputCmds, inputCommands[i])
			continue
		}

		newCommands := timestampTransform.modifyQueueSubmit(ctx, id.GetID(), vkQueueSubmitCmd, inputState)
		outputCmds = append(outputCmds, newCommands...)
	}

	return outputCmds, nil
}

func (timestampTransform *queryTimestamps) modifyQueueSubmit(ctx context.Context, id api.CmdID, cmd *VkQueueSubmit, inputState *api.GlobalState) []api.Cmd {
	ctx = log.Enter(ctx, "queryTimestamps")

	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	timestampTransform.timestampPeriod = getTimestampLimit(cmd, inputState)
	queryCount := getQueryCount(ctx, cmd, inputState)

	outputCmds := make([]api.Cmd, 0)
	commandPool, ok := timestampTransform.getCommandPool(ctx, cmd, inputState)
	if !ok {
		newCmd, newCommandPool := timestampTransform.createCommandPool(ctx, cmd, inputState)
		outputCmds = append(outputCmds, newCmd)
		commandPool = newCommandPool
	}

	queryPoolInfo := timestampTransform.getQueryPoolInfo(ctx, cmd, inputState)
	if queryPoolInfo == nil {
		const defaultQueryPoolSize = 256
		newCmds, newQueryPoolInfo := timestampTransform.createQueryPool(ctx, cmd, inputState, defaultQueryPoolSize)
		outputCmds = append(outputCmds, newCmds...)
		queryPoolInfo = newQueryPoolInfo
	} else if queryCount > queryPoolInfo.queryPoolSize {
		queryPoolSize := uint32(math.Max(float64(queryCount), float64(queryPoolInfo.queryPoolSize*3/2)))
		newCmds, newQueryPoolInfo := timestampTransform.recreateQueryPool(ctx, cmd, inputState, queryPoolInfo, queryPoolSize)
		outputCmds = append(outputCmds, newCmds...)
		queryPoolInfo = newQueryPoolInfo
	}

	numSlotAvailable := queryPoolInfo.queryPoolSize - queryPoolInfo.queryCount
	if numSlotAvailable < queryCount {
		cb := CommandBuilder{Thread: cmd.Thread()}
		queryCmds := timestampTransform.getQueryResults(ctx, cb, inputState, queryPoolInfo)
		outputCmds = append(outputCmds, queryCmds...)
	}

	newCmds := timestampTransform.rewriteQueueSubmit(ctx, inputState, id, cmd, queryPoolInfo, commandPool)
	outputCmds = append(outputCmds, newCmds...)
	return outputCmds
}

func getTimestampLimit(cmd *VkQueueSubmit, inputState *api.GlobalState) float32 {
	vkQueue := cmd.Queue()
	queue := GetState(inputState).Queues().Get(vkQueue)
	vkDevice := queue.Device()
	device := GetState(inputState).Devices().Get(vkDevice)
	vkPhysicalDevice := device.PhysicalDevice()
	physicalDevice := GetState(inputState).PhysicalDevices().Get(vkPhysicalDevice)
	return physicalDevice.PhysicalDeviceProperties().Limits().TimestampPeriod()
}

func getQueryCount(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) uint32 {
	submitCount := cmd.SubmitCount()
	submitInfos := cmd.pSubmits.Slice(0, uint64(submitCount), inputState.MemoryLayout).MustRead(ctx, cmd, inputState, nil)
	cmdBufferCount := uint32(0)
	for i := uint32(0); i < submitCount; i++ {
		si := submitInfos[i]
		cmdBufferCount += si.CommandBufferCount()
	}
	queryCount := cmdBufferCount * 2
	return queryCount
}

func (timestampTransform *queryTimestamps) getQueryPoolInfo(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) *queryPoolInfo {
	vkQueue := cmd.Queue()

	info, ok := timestampTransform.queryPools[vkQueue]
	if !ok {
		return nil
	}

	if !GetState(inputState).QueryPools().Contains(info.queryPool) {
		return nil
	}

	return info
}

func (timestampTransform *queryTimestamps) recreateQueryPool(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState, info *queryPoolInfo, poolSize uint32) ([]api.Cmd, *queryPoolInfo) {
	cb := CommandBuilder{Thread: cmd.Thread()}
	recreateCommands := make([]api.Cmd, 0)

	// Get the results back before destroy the old querypool
	queryCommands := timestampTransform.getQueryResults(ctx, cb, inputState, info)
	recreateCommands = append(recreateCommands, queryCommands...)

	newCmd := cb.VkDestroyQueryPool(info.device, info.queryPool, memory.Nullptr)
	recreateCommands = append(recreateCommands, newCmd)

	createCommands, newInfo := timestampTransform.createQueryPool(ctx, cmd, inputState, poolSize)
	return append(recreateCommands, createCommands...), newInfo
}

func (timestampTransform *queryTimestamps) createQueryPool(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState, poolSize uint32) ([]api.Cmd, *queryPoolInfo) {
	log.I(ctx, "Create query pool of size %d", poolSize)

	queryPool := VkQueryPool(newUnusedID(false, func(id uint64) bool {
		return GetState(inputState).QueryPools().Contains(VkQueryPool(id))
	}))

	queryPoolHandleData := timestampTransform.allocations.AllocDataOrPanic(ctx, queryPool)
	queryPoolCreateInfo := timestampTransform.allocations.AllocDataOrPanic(ctx, NewVkQueryPoolCreateInfo(

		VkStructureType_VK_STRUCTURE_TYPE_QUERY_POOL_CREATE_INFO, // sType
		0,                                   // pNext
		0,                                   // flags
		VkQueryType_VK_QUERY_TYPE_TIMESTAMP, // queryType
		poolSize,                            // queryCount
		0,                                   // pipelineStatistics
	))

	queue := cmd.Queue()
	device := GetState(inputState).Queues().Get(queue).Device()
	cb := CommandBuilder{Thread: cmd.Thread()}
	newCmd := cb.VkCreateQueryPool(
		device,
		queryPoolCreateInfo.Ptr(),
		memory.Nullptr,
		queryPoolHandleData.Ptr(),
		VkResult_VK_SUCCESS)
	newCmd.AddRead(queryPoolCreateInfo.Data()).AddWrite(queryPoolHandleData.Data())

	poolInfo := &queryPoolInfo{queryPool, poolSize, device, queue, 0, []timestampRecord{}}
	timestampTransform.queryPools[queue] = poolInfo
	return []api.Cmd{newCmd}, poolInfo
}

func (timestampTransform *queryTimestamps) getCommandPool(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandPool, bool) {
	vkQueue := cmd.Queue()
	queue := GetState(inputState).Queues().Get(vkQueue)
	queueFamilyIndex := queue.Family()
	device := queue.Device()

	key := commandPoolKey{device, queueFamilyIndex}
	if cp, ok := timestampTransform.commandPools[key]; ok {
		if GetState(inputState).CommandPools().Contains(VkCommandPool(cp)) {
			return cp, true
		}
	}
	return VkCommandPool(0), false
}

func (timestampTransform *queryTimestamps) createCommandPool(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) (api.Cmd, VkCommandPool) {
	vkQueue := cmd.Queue()
	queue := GetState(inputState).Queues().Get(vkQueue)
	queueFamilyIndex := queue.Family()
	device := queue.Device()

	commandPoolID := VkCommandPool(newUnusedID(false,
		func(x uint64) bool {
			ok := GetState(inputState).CommandPools().Contains(VkCommandPool(x))
			return ok
		}))
	commandPoolCreateInfo := NewVkCommandPoolCreateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
		VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
		queueFamilyIndex, // queueFamilyIndex
	)

	commandPoolCreateInfoData := timestampTransform.allocations.AllocDataOrPanic(ctx, commandPoolCreateInfo)
	commandPoolData := timestampTransform.allocations.AllocDataOrPanic(ctx, commandPoolID)

	cb := CommandBuilder{Thread: cmd.Thread()}
	newCmd := cb.VkCreateCommandPool(
		device,
		commandPoolCreateInfoData.Ptr(),
		memory.Nullptr,
		commandPoolData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		commandPoolCreateInfoData.Data(),
	).AddWrite(
		commandPoolData.Data(),
	)

	key := commandPoolKey{device, queueFamilyIndex}
	timestampTransform.commandPools[key] = commandPoolID
	return newCmd, commandPoolID
}

func (timestampTransform *queryTimestamps) getQueryResults(ctx context.Context, cb CommandBuilder, inputState *api.GlobalState, info *queryPoolInfo) []api.Cmd {
	if info == nil {
		log.E(ctx, "queryPoolInfo is invalid")
		return nil
	}

	if info.queryCount == 0 {
		return nil
	}

	outputCmds := make([]api.Cmd, 0)
	outputCmds = append(outputCmds, cb.VkQueueWaitIdle(info.queue, VkResult_VK_SUCCESS))

	bufferLength := uint64(info.queryCount * 8)
	temp := timestampTransform.allocations.AllocOrPanic(ctx, bufferLength)
	flags := VkQueryResultFlags(VkQueryResultFlagBits_VK_QUERY_RESULT_64_BIT | VkQueryResultFlagBits_VK_QUERY_RESULT_WAIT_BIT)
	getQueryPoolResultsCmd := cb.VkGetQueryPoolResults(
		info.device,
		info.queryPool,
		0,
		info.queryCount,
		memory.Size(bufferLength),
		temp.Ptr(),
		8,
		flags,
		VkResult_VK_SUCCESS)

	outputCmds = append(outputCmds, getQueryPoolResultsCmd)

	// info will be cleared after we create this command. Therefore a copy
	// is needed as this command will be mutated later
	infoResults := info.results

	newCmd := cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		b.ReserveMemory(temp.Range())
		notificationID := b.GetNotificationID()
		timestampTransform.results[notificationID] = infoResults
		b.Notification(notificationID, value.ObservedPointer(temp.Address()), bufferLength)
		err := b.RegisterNotificationReader(notificationID, func(n gapir.Notification) {
			timestampTransform.processNotification(ctx, inputState, n)
		})

		return err
	})

	outputCmds = append(outputCmds, newCmd)

	info.queryCount = 0
	info.results = []timestampRecord{}
	return outputCmds
}

func (timestampTransform *queryTimestamps) rewriteQueueSubmit(ctx context.Context,
	inputState *api.GlobalState,
	id api.CmdID,
	cmd *VkQueueSubmit,
	info *queryPoolInfo,
	commandPool VkCommandPool) []api.Cmd {

	vkQueue := cmd.Queue()
	queue := GetState(inputState).Queues().Get(vkQueue)
	device := queue.Device()
	layout := inputState.MemoryLayout

	reads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := timestampTransform.allocations.AllocDataOrPanic(ctx, v)
		reads = append(reads, res)
		return res
	}

	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	submitCount := cmd.SubmitCount()
	submitInfos := cmd.pSubmits.Slice(0, uint64(submitCount), layout).MustRead(ctx, cmd, inputState, nil)
	newSubmitInfos := make([]VkSubmitInfo, submitCount)

	cb := CommandBuilder{Thread: cmd.Thread()}
	outputCmds := make([]api.Cmd, 0)

	for i := uint32(0); i < submitCount; i++ {
		si := submitInfos[i]
		waitSemPtr := memory.Nullptr
		waitDstStagePtr := memory.Nullptr
		if count := uint64(si.WaitSemaphoreCount()); count > 0 {
			waitSemPtr = allocAndRead(si.PWaitSemaphores().
				Slice(0, count, layout).
				MustRead(ctx, cmd, inputState, nil)).Ptr()
			waitDstStagePtr = allocAndRead(si.PWaitDstStageMask().
				Slice(0, count, layout).
				MustRead(ctx, cmd, inputState, nil)).Ptr()
		}

		signalSemPtr := memory.Nullptr

		if count := uint64(si.SignalSemaphoreCount()); count > 0 {
			signalSemPtr = allocAndRead(si.PSignalSemaphores().
				Slice(0, count, layout).
				MustRead(ctx, cmd, inputState, nil)).Ptr()
		}

		cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()), layout).MustRead(ctx, cmd, inputState, nil)
		cmdCount := si.CommandBufferCount()
		cmdBufferPtr := memory.Nullptr
		newCmdCount := uint32(0)
		if cmdCount != 0 {
			newCmdCount = cmdCount*2 + 1
			queryCommands, commandbuffer := timestampTransform.generateQueryCommand(ctx,
				cb,
				inputState,
				device,
				info,
				commandPool)
			outputCmds = append(outputCmds, queryCommands...)
			info.queryCount++
			newCmdBuffers := make([]VkCommandBuffer, newCmdCount)
			newCmdBuffers[0] = commandbuffer
			for j := uint32(0); j < cmdCount; j++ {
				buf := cmdBuffers[j]
				newCmdBuffers[j*2+1] = buf
				queryCommands, commandbuffer = timestampTransform.generateQueryCommand(ctx,
					cb,
					inputState,
					device,
					info,
					commandPool)
				outputCmds = append(outputCmds, queryCommands...)
				info.queryCount++
				newCmdBuffers[j*2+2] = commandbuffer

				begin := &path.Command{
					Indices: []uint64{uint64(id), uint64(i), uint64(j), 0},
				}
				c, ok := GetState(inputState).CommandBuffers().Lookup(buf)
				if !ok {
					fmt.Errorf("Invalid command buffer %v", buf)
				}
				n := c.CommandReferences().Len()

				k := 0
				if n > 0 {
					k = n - 1
				}
				end := &path.Command{
					Indices: []uint64{uint64(id), uint64(i), uint64(j), uint64(k)},
				}
				timestampItem := service.TimestampsItem{Begin: begin, End: end, TimeInNanoseconds: 0}
				info.results = append(info.results,
					timestampRecord{timestamp: timestampItem, IsEoC: j == cmdCount-1})
			}

			cmdBufferPtr = allocAndRead(newCmdBuffers).Ptr()
		}
		newSubmitInfos[i] = NewVkSubmitInfo(
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
			0,                            // pNext
			si.WaitSemaphoreCount(),      // waitSemaphoreCount
			NewVkSemaphoreᶜᵖ(waitSemPtr), // pWaitSemaphores
			NewVkPipelineStageFlagsᶜᵖ(waitDstStagePtr), // pWaitDstStageMask
			newCmdCount,                        // commandBufferCount
			NewVkCommandBufferᶜᵖ(cmdBufferPtr), // pCommandBuffers
			si.SignalSemaphoreCount(),          // signalSemaphoreCount
			NewVkSemaphoreᶜᵖ(signalSemPtr),     // pSignalSemaphores
		)
	}

	submitInfoPtr := allocAndRead(newSubmitInfos).Ptr()

	newCmd := cb.VkQueueSubmit(
		cmd.Queue(),
		cmd.SubmitCount(),
		submitInfoPtr,
		cmd.Fence(),
		VkResult_VK_SUCCESS,
	)

	outputCmds = append(outputCmds, newCmd)

	for _, read := range reads {
		newCmd.AddRead(read.Data())
	}
	return outputCmds
}

func (timestampTransform *queryTimestamps) generateQueryCommand(ctx context.Context,
	cb CommandBuilder,
	inputState *api.GlobalState,
	device VkDevice,
	info *queryPoolInfo,
	commandPool VkCommandPool) ([]api.Cmd, VkCommandBuffer) {

	commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		commandPool,                                                    // commandPool
		VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
		1, // commandBufferCount
	)

	commandBufferAllocateInfoData := timestampTransform.allocations.AllocDataOrPanic(ctx, commandBufferAllocateInfo)
	commandBufferID := VkCommandBuffer(newUnusedID(true,
		func(x uint64) bool {
			ok := GetState(inputState).CommandBuffers().Contains(VkCommandBuffer(x))
			return ok
		}))
	commandBufferData := timestampTransform.allocations.AllocDataOrPanic(ctx, commandBufferID)

	beginCommandBufferInfo := NewVkCommandBufferBeginInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		0, // pNext
		VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
		0, // pInheritanceInfo
	)
	beginCommandBufferInfoData := timestampTransform.allocations.AllocDataOrPanic(ctx, beginCommandBufferInfo)

	outputCmds := make([]api.Cmd, 0)

	outputCmds = append(outputCmds,
		cb.VkAllocateCommandBuffers(
			device,
			commandBufferAllocateInfoData.Ptr(),
			commandBufferData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			commandBufferAllocateInfoData.Data(),
		).AddWrite(
			commandBufferData.Data(),
		),
		cb.VkBeginCommandBuffer(
			commandBufferID,
			beginCommandBufferInfoData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			beginCommandBufferInfoData.Data(),
		),
		cb.VkCmdResetQueryPool(commandBufferID, info.queryPool, info.queryCount, 1),
		cb.VkCmdWriteTimestamp(commandBufferID,
			VkPipelineStageFlagBits_VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
			info.queryPool,
			info.queryCount),
		cb.VkEndCommandBuffer(
			commandBufferID,
			VkResult_VK_SUCCESS,
		))

	return outputCmds, commandBufferID
}

func (timestampTransform *queryTimestamps) processNotification(ctx context.Context, s *api.GlobalState, n gapir.Notification) {
	notificationData := n.GetData()
	notificationID := n.GetId()
	timestampsData := notificationData.GetData()

	res, ok := timestampTransform.results[notificationID]
	if !ok {
		log.I(ctx, "Invalid notificationID %d", notificationID)
		return
	}

	byteOrder := s.MemoryLayout.GetEndian()
	r := endian.Reader(bytes.NewReader(timestampsData), byteOrder)
	tStart := r.Uint64()
	var timestamps service.Timestamps
	resultCount := uint32(len(timestampsData) / 8)
	resIdx := 0

	for i := uint32(1); i < resultCount; i++ {
		tEnd := r.Uint64()
		record := res[resIdx]
		record.timestamp.TimeInNanoseconds = uint64(float32(tEnd-tStart) * timestampTransform.timestampPeriod)
		if record.IsEoC {
			tStart = r.Uint64()
			i++
		} else {
			tStart = tEnd
		}
		timestamps.Timestamps = append(timestamps.Timestamps, &record.timestamp)
		resIdx++
	}

	timestampTransform.handler(&service.GetTimestampsResponse{
		Res: &service.GetTimestampsResponse_Timestamps{
			Timestamps: &timestamps,
		},
	})
}

func (timestampTransform *queryTimestamps) cleanup(ctx context.Context, inputState *api.GlobalState) []api.Cmd {
	cb := CommandBuilder{Thread: 0}
	outputCmds := make([]api.Cmd, 0, len(timestampTransform.queryPools)+len(timestampTransform.commandPools))

	for _, info := range timestampTransform.queryPools {
		cmd := cb.VkDestroyQueryPool(
			info.device,
			info.queryPool,
			memory.Nullptr,
		)
		outputCmds = append(outputCmds, cmd)
	}

	for commandPoolkey, commandPool := range timestampTransform.commandPools {
		cmd := cb.VkDestroyCommandPool(
			commandPoolkey.device,
			commandPool,
			memory.Nullptr,
		)
		outputCmds = append(outputCmds, cmd)
	}

	timestampTransform.queryPools = make(map[VkQueue]*queryPoolInfo)
	timestampTransform.commandPools = make(map[commandPoolKey]VkCommandPool)
	return outputCmds
}
