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

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Default query pool size
const queryPoolSize = 256

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
	replay.EndOfReplay
	cmds            []api.Cmd
	commandPools    map[commandPoolKey]VkCommandPool
	queryPools      map[VkQueue]*queryPoolInfo
	timestampPeriod float32
	replayResult    []replay.Result
	allocated       []*api.AllocResult
	willLoop        bool
	readyToLoop     bool
	handler         service.TimeStampsHandler
	results         map[uint64]queryResults
}

func newQueryTimestamps(ctx context.Context, c *capture.GraphicsCapture, Cmds []api.Cmd, willLoop bool, handler service.TimeStampsHandler) *queryTimestamps {
	transform := &queryTimestamps{
		cmds:         Cmds,
		commandPools: make(map[commandPoolKey]VkCommandPool),
		queryPools:   make(map[VkQueue]*queryPoolInfo),
		willLoop:     willLoop,
		handler:      handler,
		results:      make(map[uint64]queryResults),
	}
	return transform
}

func max(x, y uint32) uint32 {
	if x > y {
		return x
	}
	return y
}

func (t *queryTimestamps) mustAllocData(ctx context.Context, s *api.GlobalState, v ...interface{}) api.AllocResult {
	res := s.AllocDataOrPanic(ctx, v...)
	t.allocated = append(t.allocated, &res)
	return res
}

func (t *queryTimestamps) createQueryPoolIfNeeded(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	queue VkQueue,
	device VkDevice,
	numQuery uint32) *queryPoolInfo {
	s := out.State()

	qSize := uint32(0)
	info, ok := t.queryPools[queue]
	if ok && GetState(s).QueryPools().Contains(info.queryPool) {
		if numQuery <= info.queryPoolSize {
			return info
		}
		// Get the results back before destroy the old querypool
		t.GetQueryResults(ctx, cb, out, info)
		newCmd := cb.VkDestroyQueryPool(
			info.device,
			info.queryPool,
			memory.Nullptr)
		out.MutateAndWrite(ctx, api.CmdNoID, newCmd)
		// Increase the size of pool to 1.5 times of previous size or set it to numQuery whichever is larger.
		qSize = max(numQuery, info.queryPoolSize*3/2)
	} else {
		qSize = queryPoolSize
	}
	log.I(ctx, "Create query pool of size %d", qSize)

	queryPool := VkQueryPool(newUnusedID(false, func(id uint64) bool {
		return GetState(s).QueryPools().Contains(VkQueryPool(id))
	}))

	queryPoolHandleData := t.mustAllocData(ctx, s, queryPool)
	queryPoolCreateInfo := t.mustAllocData(ctx, s, NewVkQueryPoolCreateInfo(s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_QUERY_POOL_CREATE_INFO, // sType
		0,                                   // pNext
		0,                                   // flags
		VkQueryType_VK_QUERY_TYPE_TIMESTAMP, // queryType
		qSize,                               // queryCount
		0,                                   // pipelineStatistics
	))

	newCmd := cb.VkCreateQueryPool(
		device,
		queryPoolCreateInfo.Ptr(),
		memory.Nullptr,
		queryPoolHandleData.Ptr(),
		VkResult_VK_SUCCESS)
	newCmd.AddRead(queryPoolCreateInfo.Data()).AddWrite(queryPoolHandleData.Data())

	info = &queryPoolInfo{queryPool, qSize, device, queue, 0, []timestampRecord{}}
	t.queryPools[queue] = info
	out.MutateAndWrite(ctx, api.CmdNoID, newCmd)
	return info
}

func (t *queryTimestamps) createCommandpoolIfNeeded(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	device VkDevice,
	queueFamilyIndex uint32) VkCommandPool {
	key := commandPoolKey{device, queueFamilyIndex}
	s := out.State()

	if cp, ok := t.commandPools[key]; ok {
		if GetState(s).CommandPools().Contains(VkCommandPool(cp)) {
			return cp
		}
	}

	commandPoolID := VkCommandPool(newUnusedID(false,
		func(x uint64) bool {
			ok := GetState(s).CommandPools().Contains(VkCommandPool(x))
			return ok
		}))
	commandPoolCreateInfo := NewVkCommandPoolCreateInfo(s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
		VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
		queueFamilyIndex, // queueFamilyIndex
	)
	commandPoolCreateInfoData := t.mustAllocData(ctx, s, commandPoolCreateInfo)
	commandPoolData := t.mustAllocData(ctx, s, commandPoolID)

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
	out.MutateAndWrite(ctx, api.CmdNoID, newCmd)

	t.commandPools[key] = commandPoolID
	return commandPoolID
}

func (t *queryTimestamps) generateQueryCommand(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	device VkDevice,
	queryPool VkQueryPool,
	commandPool VkCommandPool,
	query uint32) VkCommandBuffer {
	s := out.State()
	commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		commandPool,                                                    // commandPool
		VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
		1, // commandBufferCount
	)
	commandBufferAllocateInfoData := t.mustAllocData(ctx, s, commandBufferAllocateInfo)
	commandBufferID := VkCommandBuffer(newUnusedID(true,
		func(x uint64) bool {
			ok := GetState(s).CommandBuffers().Contains(VkCommandBuffer(x))
			return ok
		}))
	commandBufferData := t.mustAllocData(ctx, s, commandBufferID)

	beginCommandBufferInfo := NewVkCommandBufferBeginInfo(s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		0, // pNext
		VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
		0, // pInheritanceInfo
	)
	beginCommandBufferInfoData := t.mustAllocData(ctx, s, beginCommandBufferInfo)

	writeEach(ctx, out,
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
		cb.VkCmdResetQueryPool(commandBufferID, queryPool, query, 1),
		cb.VkCmdWriteTimestamp(commandBufferID,
			VkPipelineStageFlagBits_VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
			queryPool,
			query),
		cb.VkEndCommandBuffer(
			commandBufferID,
			VkResult_VK_SUCCESS,
		),
	)

	return commandBufferID
}

func (t *queryTimestamps) rewriteQueueSubmit(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	id api.CmdID,
	device VkDevice,
	queryPoolInfo *queryPoolInfo,
	commandPool VkCommandPool,
	cmd *VkQueueSubmit) error {

	s := out.State()
	l := s.MemoryLayout
	reads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := t.mustAllocData(ctx, s, v)
		reads = append(reads, res)
		return res
	}

	cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
	submitCount := cmd.SubmitCount()
	submitInfos := cmd.pSubmits.Slice(0, uint64(submitCount), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
	newSubmitInfos := make([]VkSubmitInfo, submitCount)

	for i := uint32(0); i < submitCount; i++ {
		si := submitInfos[i]
		waitSemPtr := memory.Nullptr
		waitDstStagePtr := memory.Nullptr
		if count := uint64(si.WaitSemaphoreCount()); count > 0 {
			waitSemPtr = allocAndRead(si.PWaitSemaphores().
				Slice(0, count, l).
				MustRead(ctx, cmd, s, nil)).Ptr()
			waitDstStagePtr = allocAndRead(si.PWaitDstStageMask().
				Slice(0, count, l).
				MustRead(ctx, cmd, s, nil)).Ptr()
		}

		signalSemPtr := memory.Nullptr

		if count := uint64(si.SignalSemaphoreCount()); count > 0 {
			signalSemPtr = allocAndRead(si.PSignalSemaphores().
				Slice(0, count, l).
				MustRead(ctx, cmd, s, nil)).Ptr()
		}

		cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
		cmdCount := si.CommandBufferCount()
		cmdBufferPtr := memory.Nullptr
		newCmdCount := uint32(0)
		if cmdCount != 0 {
			newCmdCount = cmdCount*2 + 1
			commandbuffer := t.generateQueryCommand(ctx,
				cb,
				out,
				device,
				queryPoolInfo.queryPool,
				commandPool,
				queryPoolInfo.queryCount)
			queryPoolInfo.queryCount++
			newCmdBuffers := make([]VkCommandBuffer, newCmdCount)
			newCmdBuffers[0] = commandbuffer
			for j := uint32(0); j < cmdCount; j++ {
				buf := cmdBuffers[j]
				newCmdBuffers[j*2+1] = buf

				commandbuffer = t.generateQueryCommand(ctx,
					cb,
					out,
					device,
					queryPoolInfo.queryPool,
					commandPool,
					queryPoolInfo.queryCount)
				queryPoolInfo.queryCount++
				newCmdBuffers[j*2+2] = commandbuffer

				begin := &path.Command{
					Indices: []uint64{uint64(id), uint64(i), uint64(j), 0},
				}
				c, ok := GetState(s).CommandBuffers().Lookup(buf)
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
				queryPoolInfo.results = append(queryPoolInfo.results,
					timestampRecord{timestamp: timestampItem, IsEoC: j == cmdCount-1})
			}

			cmdBufferPtr = allocAndRead(newCmdBuffers).Ptr()
		}
		newSubmitInfos[i] = NewVkSubmitInfo(s.Arena,
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

	for _, read := range reads {
		newCmd.AddRead(read.Data())
	}
	return out.MutateAndWrite(ctx, id, newCmd)
}

func (t *queryTimestamps) GetQueryResults(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	queryPoolInfo *queryPoolInfo) {
	if queryPoolInfo == nil {
		log.E(ctx, "queryPoolInfo is invalid")
		return
	}
	queryCount := queryPoolInfo.queryCount
	queue := queryPoolInfo.queue
	if queryCount == 0 {
		return
	}
	s := out.State()
	waitCmd := cb.VkQueueWaitIdle(queue, VkResult_VK_SUCCESS)
	out.MutateAndWrite(ctx, api.CmdNoID, waitCmd)

	buflen := uint64(queryCount * 8)
	tmp := s.AllocOrPanic(ctx, buflen)
	flags := VkQueryResultFlags(VkQueryResultFlagBits_VK_QUERY_RESULT_64_BIT | VkQueryResultFlagBits_VK_QUERY_RESULT_WAIT_BIT)
	newCmd := cb.VkGetQueryPoolResults(
		queryPoolInfo.device,
		queryPoolInfo.queryPool,
		0,
		queryCount,
		memory.Size(buflen),
		tmp.Ptr(),
		8,
		flags,
		VkResult_VK_SUCCESS)

	out.MutateAndWrite(ctx, api.CmdNoID, newCmd)

	out.MutateAndWrite(ctx, api.CmdNoID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		b.ReserveMemory(tmp.Range())
		notificationID := b.GetNotificationID()
		t.results[notificationID] = queryPoolInfo.results
		b.Notification(notificationID, value.ObservedPointer(tmp.Address()), buflen)
		return b.RegisterNotificationReader(notificationID, func(n gapir.Notification) {
			t.processNotification(ctx, s, n)
		})
	}))

	queryPoolInfo.queryCount = 0
	queryPoolInfo.results = []timestampRecord{}
	tmp.Free()
}

func (t *queryTimestamps) processNotification(ctx context.Context, s *api.GlobalState, n gapir.Notification) {
	notificationData := n.GetData()
	notificationID := n.GetId()
	timestampsData := notificationData.GetData()

	res, ok := t.results[notificationID]
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
		record.timestamp.TimeInNanoseconds = uint64(float32(tEnd-tStart) * t.timestampPeriod)
		if record.IsEoC {
			tStart = r.Uint64()
			i++
		} else {
			tStart = tEnd
		}
		timestamps.Timestamps = append(timestamps.Timestamps, &record.timestamp)
		resIdx++
	}

	t.handler(&service.GetTimestampsResponse{
		Res: &service.GetTimestampsResponse_Timestamps{
			Timestamps: &timestamps,
		},
	})
}

func (t *queryTimestamps) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	ctx = log.Enter(ctx, "queryTimestamps")
	if t.willLoop && !t.readyToLoop {
		return out.MutateAndWrite(ctx, id, cmd)
	}
	s := out.State()
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}

	defer func() {
		for _, d := range t.allocated {
			d.Free()
		}
	}()

	switch cmd := cmd.(type) {
	case *VkQueueSubmit:
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		vkQueue := cmd.Queue()
		queue := GetState(s).Queues().Get(vkQueue)
		queueFamilyIndex := queue.Family()
		vkDevice := queue.Device()
		device := GetState(s).Devices().Get(vkDevice)
		vkPhysicalDevice := device.PhysicalDevice()
		physicalDevice := GetState(s).PhysicalDevices().Get(vkPhysicalDevice)
		t.timestampPeriod = physicalDevice.PhysicalDeviceProperties().Limits().TimestampPeriod()

		submitCount := cmd.SubmitCount()
		submitInfos := cmd.pSubmits.Slice(0, uint64(submitCount), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
		cmdBufferCount := uint32(0)
		for i := uint32(0); i < submitCount; i++ {
			si := submitInfos[i]
			cmdBufferCount += si.CommandBufferCount()
		}
		queryCount := cmdBufferCount * 2

		commandPool := t.createCommandpoolIfNeeded(ctx, cb, out, vkDevice, queueFamilyIndex)
		queryPoolInfo := t.createQueryPoolIfNeeded(ctx, cb, out, vkQueue, vkDevice, queryCount)
		numSlotAvailable := queryPoolInfo.queryPoolSize - queryPoolInfo.queryCount
		if numSlotAvailable < queryCount {
			t.GetQueryResults(ctx, cb, out, queryPoolInfo)
		}
		return t.rewriteQueueSubmit(ctx, cb, out, id, vkDevice, queryPoolInfo, commandPool, cmd)

	default:
		return out.MutateAndWrite(ctx, id, cmd)
	}
	return nil
}

func (t *queryTimestamps) Flush(ctx context.Context, out transform.Writer) error {
	s := out.State()
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}
	for _, queryPoolInfo := range t.queryPools {
		t.GetQueryResults(ctx, cb, out, queryPoolInfo)
	}
	t.cleanup(ctx, out)
	t.AddNotifyInstruction(ctx, out, func() interface{} { return t.replayResult })
	return nil
}

func (t *queryTimestamps) PreLoop(ctx context.Context, out transform.Writer) {
	t.readyToLoop = true
}

func (t *queryTimestamps) PostLoop(ctx context.Context, out transform.Writer) {
	t.readyToLoop = false
	s := out.State()
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}
	for _, queryPoolInfo := range t.queryPools {
		t.GetQueryResults(ctx, cb, out, queryPoolInfo)
	}
	t.cleanup(ctx, out)
}

func (t *queryTimestamps) cleanup(ctx context.Context, out transform.Writer) {
	s := out.State()
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}

	// Destroy query pool created in this transform.
	for _, info := range t.queryPools {
		cmd := cb.VkDestroyQueryPool(
			info.device,
			info.queryPool,
			memory.Nullptr,
		)
		out.MutateAndWrite(ctx, api.CmdNoID, cmd)
	}
	t.queryPools = make(map[VkQueue]*queryPoolInfo)

	// Free commandpool allocated in this transform.
	for commandPoolkey, commandPool := range t.commandPools {
		cmd := cb.VkDestroyCommandPool(
			commandPoolkey.device,
			commandPool,
			memory.Nullptr,
		)
		out.MutateAndWrite(ctx, api.CmdNoID, cmd)
	}
	t.commandPools = make(map[commandPoolKey]VkCommandPool)
}

func (t *queryTimestamps) BuffersCommands() bool { return false }

func writeEach(ctx context.Context, out transform.Writer, cmds ...api.Cmd) error {
	for _, cmd := range cmds {
		if err := out.MutateAndWrite(ctx, api.CmdNoID, cmd); err != nil {
			return err
		}
	}
	return nil
}
