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
	"context"
	"fmt"
	"time"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service/path"
)

// Default query pool size
const queryPoolSize = 256

// queryPoolInfo contains the information about the query pool
type queryPoolInfo struct {
	queryPool     VkQueryPool
	queryPoolSize uint32
	device        VkDevice
}

type queryTimestamps struct {
	commandPools map[VkDevice]VkCommandPool
	queryPools   map[VkQueue]queryPoolInfo
	replayResult []replay.Result
	timestamps   []replay.Timestamp
	allocated    []*api.AllocResult
	resultIndex  uint32
}

func newQueryTimestamps(ctx context.Context, c *capture.Capture, numInitialCmds int) *queryTimestamps {
	transform := &queryTimestamps{
		commandPools: make(map[VkDevice]VkCommandPool),
		queryPools:   make(map[VkQueue]queryPoolInfo),
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

func (t *queryTimestamps) reportTo(r replay.Result) { t.replayResult = append(t.replayResult, r) }

func (t *queryTimestamps) createQueryPoolIfNeeded(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	queue VkQueue,
	device VkDevice,
	numQuery uint32) VkQueryPool {
	s := out.State()

	qSize := uint32(0)
	q, ok := t.queryPools[queue]
	if ok && GetState(s).QueryPools().Contains(q.queryPool) {
		if numQuery <= q.queryPoolSize {
			return q.queryPool
		}
		// Destroy old query pool if it can not hold the results.
		newCmd := cb.VkDestroyQueryPool(
			q.device,
			q.queryPool,
			memory.Nullptr)
		out.MutateAndWrite(ctx, api.CmdNoID, newCmd)
		// Increase the size of pool to 1.5 times of previous size every time.
		qSize = max(numQuery, q.queryPoolSize*3/2)
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
	t.queryPools[queue] = queryPoolInfo{queryPool, qSize, device}
	out.MutateAndWrite(ctx, api.CmdNoID, newCmd)
	return queryPool
}

func (t *queryTimestamps) createComandpoolIfNeeded(ctx context.Context,
	cb CommandBuilder,
	out transform.Writer,
	device VkDevice,
	queueFamilyIndex uint32) VkCommandPool {
	s := out.State()

	if cp, ok := t.commandPools[device]; ok {
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

	t.commandPools[device] = commandPoolID
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
	queryPool VkQueryPool,
	commandPool VkCommandPool,
	cmd *VkQueueSubmit) {

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
	query := uint32(0)
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
		newCmdCount := cmdCount*2 + 1

		commandbuffer := t.generateQueryCommand(ctx,
			cb,
			out,
			device,
			queryPool,
			commandPool,
			query)
		newCmdBuffers := make([]VkCommandBuffer, newCmdCount)
		newCmdBuffers[0] = commandbuffer
		for j := uint32(0); j < cmdCount; j++ {
			buf := cmdBuffers[j]
			newCmdBuffers[j*2+1] = buf

			query++
			commandbuffer = t.generateQueryCommand(ctx,
				cb,
				out,
				device,
				queryPool,
				commandPool,
				query)
			newCmdBuffers[j*2+2] = commandbuffer

			begin := &path.Command{
				Indices: []uint64{uint64(id), uint64(i), uint64(j), 0},
			}
			c, ok := GetState(s).CommandBuffers().Lookup(buf)
			if !ok {
				fmt.Errorf("Invalid command buffer %v", buf)
			}
			n := c.CommandReferences().Len()
			end := &path.Command{
				Indices: []uint64{uint64(id), uint64(i), uint64(j), uint64(n - 1)},
			}
			t.timestamps = append(t.timestamps, replay.Timestamp{Begin: begin, End: end})
		}

		cmdBufferPtr := allocAndRead(newCmdBuffers).Ptr()
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
	out.MutateAndWrite(ctx, id, newCmd)
}

func (t *queryTimestamps) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
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
		timestampPeriod := physicalDevice.PhysicalDeviceProperties().Limits().TimestampPeriod()

		submitCount := cmd.SubmitCount()
		submitInfos := cmd.pSubmits.Slice(0, uint64(submitCount), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
		cmdBufferCount := uint32(0)
		for i := uint32(0); i < submitCount; i++ {
			si := submitInfos[i]
			cmdBufferCount += si.CommandBufferCount()
		}
		queryCount := cmdBufferCount * 2

		commandPool := t.createComandpoolIfNeeded(ctx, cb, out, vkDevice, queueFamilyIndex)
		queryPool := t.createQueryPoolIfNeeded(ctx, cb, out, vkQueue, vkDevice, queryCount)
		t.rewriteQueueSubmit(ctx, cb, out, id, vkDevice, queryPool, commandPool, cmd)

		// TODO: Retrieve the results when the query pool is full or when the replay
		// is done instead of wait queue idle for every submission.
		// Further improvement is doing a pre-pass over the commands to
		// determine the query pool size.
		waitCmd := cb.VkQueueWaitIdle(vkQueue, VkResult_VK_SUCCESS)
		out.MutateAndWrite(ctx, id, waitCmd)

		// Get results back.
		buflen := uint64(queryCount * 8)
		tmp := s.AllocOrPanic(ctx, buflen)
		flags := VkQueryResultFlags(VkQueryResultFlagBits_VK_QUERY_RESULT_64_BIT | VkQueryResultFlagBits_VK_QUERY_RESULT_WAIT_BIT)
		newCmd := cb.VkGetQueryPoolResults(
			vkDevice,
			queryPool,
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
			b.Post(value.ObservedPointer(tmp.Address()), buflen, func(r binary.Reader, err error) {
				if err != nil {
					log.I(ctx, "b post get err %v", err)
					return
				}

				tStart := r.Uint64()
				for i := uint32(1); i < queryCount; i++ {
					tEnd := r.Uint64()
					t.timestamps[t.resultIndex].Time = time.Duration(uint64(float32(tEnd-tStart)*timestampPeriod)) * time.Nanosecond
					t.resultIndex++
					tStart = tEnd
				}
			})
			return nil
		}))

		tmp.Free()

	default:
		out.MutateAndWrite(ctx, id, cmd)
	}
}

func (t *queryTimestamps) Flush(ctx context.Context, out transform.Writer) {
	s := out.State()
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}
	out.MutateAndWrite(ctx, api.CmdNoID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		code := uint32(0xbeefcace)
		b.Push(value.U32(code))
		b.Post(b.Buffer(1), 4, func(r binary.Reader, err error) {
			for _, res := range t.replayResult {
				res.Do(func() (interface{}, error) {
					if err != nil {
						return nil, log.Err(ctx, err, "Flush did not get expected EOS code: '%v'")
					}
					if r.Uint32() != code {
						return nil, log.Err(ctx, nil, "Flush did not get expected EOS code")
					}
					return t.timestamps, nil
				})
			}
		})
		return nil
	}))
}
