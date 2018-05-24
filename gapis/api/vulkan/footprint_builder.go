// Copyright (C) 2017 Google Inc.
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
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

var emptyDefUseVars = []dependencygraph.DefUseVariable{}

const vkWholeSize = uint64(0xFFFFFFFFFFFFFFFF)
const vkAttachmentUnused = uint32(0xFFFFFFFF)
const vkNullHandle = vkHandle(0x0)

// Assume the value of a Vulkan handle is always unique
type vkHandle uint64

func (vkHandle) DefUseVariable() {}

// label
type label struct {
	uint64
}

var nextLabelVal uint64 = 1

func (label) DefUseVariable() {}

func newLabel() label { i := nextLabelVal; nextLabelVal++; return label{i} }

// Forward-paired label
type forwardPairedLabel struct {
	labelReadBehaviors []*dependencygraph.Behavior
}

func (*forwardPairedLabel) DefUseVariable() {}
func newForwardPairedLabel(ctx context.Context,
	bh *dependencygraph.Behavior) *forwardPairedLabel {
	fpl := &forwardPairedLabel{labelReadBehaviors: []*dependencygraph.Behavior{}}
	write(ctx, bh, fpl)
	return fpl
}

// memory
type memorySpan struct {
	span   interval.U64Span
	memory VkDeviceMemory
}

func (memorySpan) DefUseVariable() {}

// commands
type commandBufferCommand struct {
	isCmdExecuteCommands    bool
	secondaryCommandBuffers []VkCommandBuffer
	behave                  func(submittedCommand, *queueExecutionState)
}

func (cbc *commandBufferCommand) newBehavior(ctx context.Context,
	sc submittedCommand, m *vulkanMachine,
	qei *queueExecutionState) *dependencygraph.Behavior {
	bh := dependencygraph.NewBehavior(sc.id, m)
	read(ctx, bh, cbc)
	read(ctx, bh, qei.currentSubmitInfo.queued)
	if sc.parentCmd != nil {
		read(ctx, bh, sc.parentCmd)
	}
	return bh
}

func (*commandBufferCommand) DefUseVariable() {}

func (cbc *commandBufferCommand) recordSecondaryCommandBuffer(vkCb VkCommandBuffer) {
	cbc.secondaryCommandBuffers = append(cbc.secondaryCommandBuffers, vkCb)
}

// vulkanMachine is the back-propagation machine for Vulkan API commands.
type vulkanMachine struct {
	handles                       map[vkHandle]struct{}
	labels                        map[label]struct{}
	memories                      map[VkDeviceMemory]*interval.U64SpanList
	commandBufferCommands         map[*commandBufferCommand]struct{}
	subpassIndices                map[*subpassIndex]struct{}
	boundDataPieces               map[*resBinding]struct{}
	descriptors                   map[*descriptor]struct{}
	boundDescriptorSets           map[*boundDescriptorSet]struct{}
	forwardPairedLabels           map[*forwardPairedLabel]struct{}
	lastBoundFramebufferImageData map[api.CmdID][]dependencygraph.DefUseVariable
}

func newVulkanMachine() *vulkanMachine {
	return &vulkanMachine{
		handles:                       map[vkHandle]struct{}{},
		labels:                        map[label]struct{}{},
		memories:                      map[VkDeviceMemory]*interval.U64SpanList{},
		commandBufferCommands:         map[*commandBufferCommand]struct{}{},
		subpassIndices:                map[*subpassIndex]struct{}{},
		boundDataPieces:               map[*resBinding]struct{}{},
		descriptors:                   map[*descriptor]struct{}{},
		boundDescriptorSets:           map[*boundDescriptorSet]struct{}{},
		forwardPairedLabels:           map[*forwardPairedLabel]struct{}{},
		lastBoundFramebufferImageData: map[api.CmdID][]dependencygraph.DefUseVariable{},
	}
}

func (m *vulkanMachine) Clear() {
	m.handles = map[vkHandle]struct{}{}
	m.labels = map[label]struct{}{}
	m.memories = map[VkDeviceMemory]*interval.U64SpanList{}
	m.commandBufferCommands = map[*commandBufferCommand]struct{}{}
	m.subpassIndices = map[*subpassIndex]struct{}{}
	m.boundDataPieces = map[*resBinding]struct{}{}
	m.descriptors = map[*descriptor]struct{}{}
	m.boundDescriptorSets = map[*boundDescriptorSet]struct{}{}
	m.forwardPairedLabels = map[*forwardPairedLabel]struct{}{}
	m.lastBoundFramebufferImageData = map[api.CmdID][]dependencygraph.DefUseVariable{}
}

func (m *vulkanMachine) IsAlive(behaviorIndex uint64,
	ft *dependencygraph.Footprint) bool {
	bh := ft.Behaviors[behaviorIndex]
	for _, w := range bh.Writes {
		if m.checkDef(w) {
			return true
		}
	}
	return false
}

func (m *vulkanMachine) FramebufferRequest(id api.CmdID,
	ft *dependencygraph.Footprint) {
	fbData, ok := m.lastBoundFramebufferImageData[id]
	if ok {
		for _, d := range fbData {
			m.use(d)
		}
	}
}

func (m *vulkanMachine) RecordBehaviorEffects(behaviorIndex uint64,
	ft *dependencygraph.Footprint) []uint64 {
	bh := ft.Behaviors[behaviorIndex]
	aliveIndices := []uint64{behaviorIndex}
	for _, w := range bh.Writes {
		extraAliveBehaviors := m.def(w)
		for _, eb := range extraAliveBehaviors {
			aliveIndices = append(aliveIndices, ft.BehaviorIndices[eb])
		}
	}
	for _, r := range bh.Reads {
		m.use(r)
	}
	return aliveIndices
}

func (m *vulkanMachine) checkDef(c dependencygraph.DefUseVariable) bool {
	switch c := c.(type) {
	case vkHandle:
		if _, ok := m.handles[c]; ok {
			return true
		}
	case label:
		if _, ok := m.labels[c]; ok {
			return true
		}
	case memorySpan:
		if _, exist := m.memories[c.memory]; exist {
			if _, count := interval.Intersect(m.memories[c.memory], c.span); count > 0 {
				return true
			}
		}
	case *commandBufferCommand:
		if _, ok := m.commandBufferCommands[c]; ok {
			return true
		}
	case *subpassIndex:
		if _, ok := m.subpassIndices[c]; ok {
			return true
		}
	case *resBinding:
		if _, ok := m.boundDataPieces[c]; ok {
			return true
		}
	case *descriptor:
		if _, ok := m.descriptors[c]; ok {
			return true
		}
	case *boundDescriptorSet:
		if _, ok := m.boundDescriptorSets[c]; ok {
			return true
		}
	case *forwardPairedLabel:
		if _, ok := m.forwardPairedLabels[c]; ok {
			return true
		}
	default:
		return false
	}
	return false
}

func (m *vulkanMachine) use(c dependencygraph.DefUseVariable) {
	switch c := c.(type) {
	case vkHandle:
		m.handles[c] = struct{}{}
	case label:
		m.labels[c] = struct{}{}
	case memorySpan:
		if _, exist := m.memories[c.memory]; !exist {
			m.memories[c.memory] = &interval.U64SpanList{}
		}
		interval.Merge(m.memories[c.memory], c.span, true)
		m.handles[vkHandle(c.memory)] = struct{}{}
	case *commandBufferCommand:
		m.commandBufferCommands[c] = struct{}{}
	case *subpassIndex:
		m.subpassIndices[c] = struct{}{}
	case *resBinding:
		m.boundDataPieces[c] = struct{}{}
	case *descriptor:
		m.descriptors[c] = struct{}{}
	case *boundDescriptorSet:
		m.boundDescriptorSets[c] = struct{}{}
	case *forwardPairedLabel:
		m.forwardPairedLabels[c] = struct{}{}
	default:
	}
}

func (m *vulkanMachine) def(c dependencygraph.DefUseVariable) []*dependencygraph.Behavior {
	switch c := c.(type) {
	case vkHandle:
		delete(m.handles, c)
	case label:
		delete(m.labels, c)
	case memorySpan:
		if _, exist := m.memories[c.memory]; exist {
			interval.Remove(m.memories[c.memory], c.span)
		}
	case *commandBufferCommand:
		delete(m.commandBufferCommands, c)
	case *subpassIndex:
		delete(m.subpassIndices, c)
	case *resBinding:
		delete(m.boundDataPieces, c)
	case *descriptor:
		delete(m.descriptors, c)
	case *boundDescriptorSet:
		delete(m.boundDescriptorSets, c)
	case *forwardPairedLabel:
		// For forward paired labels, if a label is defined, the reading
		// behaviors of the label must also be kept alive. This is used for
		// begin-end pairs, like vkCmdBeginRenderPass and vkCmdEndRenderPass.
		// E.g.: If any rendering output of a render pass is used in future, the
		// vkCmdBeginRenderPass should be kept alive, then the paird
		// vkCmdEndRenderPass must also be kept alive, no matter whether the
		// rendering output of the last subpass is used or not.
		alivePairedBehaviors := []*dependencygraph.Behavior{}
		alivePairedBehaviors = append(alivePairedBehaviors, c.labelReadBehaviors...)
		delete(m.forwardPairedLabels, c)
		return alivePairedBehaviors
	default:
	}
	return []*dependencygraph.Behavior{}
}

// submittedCommand represents Subcommands. When a submidttedCommand/Subcommand
// is executed, it reads the original commands, and if it is secondary command,
// its parent primary command.
type submittedCommand struct {
	id        api.SubCmdIdx
	cmd       *commandBufferCommand
	parentCmd *commandBufferCommand
}

func newSubmittedCommand(fullCmdIndex api.SubCmdIdx,
	c *commandBufferCommand, pc *commandBufferCommand) *submittedCommand {
	return &submittedCommand{id: fullCmdIndex, cmd: c, parentCmd: pc}
}

func (sc *submittedCommand) runCommand(ctx context.Context,
	ft *dependencygraph.Footprint, m *vulkanMachine,
	execInfo *queueExecutionState) {
	sc.cmd.behave(*sc, execInfo)
}

type queueSubmitInfo struct {
	queue            VkQueue
	began            bool
	queued           label
	done             label
	waitSemaphores   []VkSemaphore
	signalSemaphores []VkSemaphore
	signalFence      VkFence
	pendingCommands  []*submittedCommand
}

type event struct {
	signal   label
	unsignal label
}

type fence struct {
	signal   label
	unsignal label
}

type query struct {
	reset  label
	begin  label
	end    label
	result label
}

func newQuery() *query {
	return &query{
		reset:  newLabel(),
		begin:  newLabel(),
		end:    newLabel(),
		result: newLabel(),
	}
}

type queryPool struct {
	queries []*query
}

type subpassAttachmentInfo struct {
	fullImageData bool
	data          []dependencygraph.DefUseVariable
	layout        label
	desc          VkAttachmentDescription
}

type subpassInfo struct {
	loadAttachments        []*subpassAttachmentInfo
	storeAttachments       []*subpassAttachmentInfo
	colorAttachments       []*subpassAttachmentInfo
	resolveAttachments     []*subpassAttachmentInfo
	inputAttachments       []*subpassAttachmentInfo
	depthStencilAttachment *subpassAttachmentInfo
	modifiedDescriptorData []dependencygraph.DefUseVariable
}

type subpassIndex struct {
	val uint64
}

func (*subpassIndex) DefUseVariable() {}

type commandBufferExecutionState struct {
	vertexBufferResBindings map[uint32]resBindingList
	indexBufferResBindings  resBindingList
	indexType               VkIndexType
	descriptorSets          map[uint32]*boundDescriptorSet
	pipeline                label
	dynamicState            label
}

func newCommandBufferExecutionState() *commandBufferExecutionState {
	return &commandBufferExecutionState{
		vertexBufferResBindings: map[uint32]resBindingList{},
		descriptorSets:          map[uint32]*boundDescriptorSet{},
		pipeline:                newLabel(),
		dynamicState:            newLabel(),
	}
}

type queueExecutionState struct {
	currentCmdBufState   *commandBufferExecutionState
	primaryCmdBufState   *commandBufferExecutionState
	secondaryCmdBufState *commandBufferExecutionState

	subpasses       []subpassInfo
	subpass         *subpassIndex
	renderPassBegin *forwardPairedLabel

	currentCommand api.SubCmdIdx

	framebuffer FramebufferObjectʳ

	lastSubmitID      api.CmdID
	currentSubmitInfo *queueSubmitInfo
}

func newQueueExecutionState(id api.CmdID) *queueExecutionState {
	return &queueExecutionState{
		subpasses:      []subpassInfo{},
		lastSubmitID:   id,
		currentCommand: api.SubCmdIdx([]uint64{0, 0, 0, 0}),
	}
}

func (qei *queueExecutionState) updateCurrentCommand(ctx context.Context,
	fci api.SubCmdIdx) {
	switch len(fci) {
	case 4:
		current := api.SubCmdIdx(qei.currentCommand[0:3])
		comming := api.SubCmdIdx(fci[0:3])
		if current.LessThan(comming) {
			// primary command buffer changed
			qei.primaryCmdBufState = newCommandBufferExecutionState()
		}
		qei.currentCmdBufState = qei.primaryCmdBufState
	case 6:
		if len(qei.currentCommand) != 6 {
			// Transit from primary command buffer to secondary command buffer
			qei.secondaryCmdBufState = newCommandBufferExecutionState()
		} else {
			current := api.SubCmdIdx(qei.currentCommand[0:5])
			comming := api.SubCmdIdx(fci[0:5])
			if current.LessThan(comming) {
				// secondary command buffer changed
				qei.secondaryCmdBufState = newCommandBufferExecutionState()
			}
		}
		qei.currentCmdBufState = qei.secondaryCmdBufState
	default:
		log.E(ctx, "FootprintBuilder: Invalid length of full command index")
	}
	qei.currentCommand = fci
}

func (o VkAttachmentLoadOp) isLoad() bool {
	return o == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD
}

func (o VkAttachmentStoreOp) isStore() bool {
	return o == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE
}

func (qei *queueExecutionState) startSubpass(ctx context.Context,
	bh *dependencygraph.Behavior) {
	write(ctx, bh, qei.subpass)
	subpassI := qei.subpass.val
	noDsAttLoadOp := func(ctx context.Context, bh *dependencygraph.Behavior,
		attachment *subpassAttachmentInfo) {
		// TODO: Not all subpasses change layouts
		modify(ctx, bh, attachment.layout)
		if attachment.desc.LoadOp().isLoad() {
			read(ctx, bh, attachment.data...)
		} else {
			if attachment.fullImageData {
				write(ctx, bh, attachment.data...)
			} else {
				modify(ctx, bh, attachment.data...)
			}
		}
	}
	dsAttLoadOp := func(ctx context.Context, bh *dependencygraph.Behavior,
		attachment *subpassAttachmentInfo) {
		// TODO: Not all subpasses change layouts
		modify(ctx, bh, attachment.layout)
		if !attachment.desc.LoadOp().isLoad() && !attachment.desc.StencilLoadOp().isLoad() {
			if attachment.fullImageData {
				write(ctx, bh, attachment.data...)
			} else {
				modify(ctx, bh, attachment.data...)
			}
		} else if attachment.desc.LoadOp().isLoad() && attachment.desc.StencilLoadOp().isLoad() {
			read(ctx, bh, attachment.data...)
		} else {
			modify(ctx, bh, attachment.data...)
		}
	}
	for _, l := range qei.subpasses[subpassI].loadAttachments {
		if qei.subpasses[subpassI].depthStencilAttachment == l {
			dsAttLoadOp(ctx, bh, l)
		} else {
			noDsAttLoadOp(ctx, bh, l)
		}
	}
}

func (qei *queueExecutionState) emitSubpassOutput(ctx context.Context,
	ft *dependencygraph.Footprint, sc submittedCommand, m *vulkanMachine) {
	subpassI := qei.subpass.val
	noDsAttStoreOp := func(ctx context.Context, ft *dependencygraph.Footprint,
		sc submittedCommand, m *vulkanMachine, att *subpassAttachmentInfo,
		readAtt *subpassAttachmentInfo) {
		// Two behaviors for each attachment. One to represent the dependency of
		// image layout, another one for the data.
		behaviorForLayout := sc.cmd.newBehavior(ctx, sc, m, qei)
		modify(ctx, behaviorForLayout, att.layout)
		read(ctx, behaviorForLayout, qei.subpass)
		ft.AddBehavior(ctx, behaviorForLayout)

		behaviorForData := sc.cmd.newBehavior(ctx, sc, m, qei)
		if readAtt != nil {
			read(ctx, behaviorForData, readAtt.data...)
		}
		if att.desc.StoreOp().isStore() {
			modify(ctx, behaviorForData, att.data...)
		} else {
			// If the attachment fully covers the unlying image, this will clear
			// the image data, which is a write operation.
			if att.fullImageData {
				write(ctx, behaviorForData, att.data...)
			} else {
				modify(ctx, behaviorForData, att.data...)
			}
		}
		read(ctx, behaviorForData, qei.subpass)
		ft.AddBehavior(ctx, behaviorForData)
	}

	dsAttStoreOp := func(ctx context.Context, ft *dependencygraph.Footprint,
		sc submittedCommand, m *vulkanMachine, dsAtt *subpassAttachmentInfo) {
		bh := sc.cmd.newBehavior(ctx, sc, m, qei)
		if dsAtt.desc.StoreOp().isStore() || dsAtt.desc.StencilStoreOp().isStore() {
			modify(ctx, bh, dsAtt.data...)
		} else {
			if dsAtt.fullImageData {
				write(ctx, bh, dsAtt.data...)
			} else {
				modify(ctx, bh, dsAtt.data...)
			}
		}
		read(ctx, bh, qei.subpass)
		ft.AddBehavior(ctx, bh)
	}

	isStoreAtt := func(att *subpassAttachmentInfo) bool {
		for _, storeAtt := range qei.subpasses[subpassI].storeAttachments {
			if att == storeAtt {
				return true
			}
		}
		return false
	}

	for i, r := range qei.subpasses[subpassI].resolveAttachments {
		if isStoreAtt(r) {
			c := qei.subpasses[subpassI].colorAttachments[i]
			noDsAttStoreOp(ctx, ft, sc, m, r, c)
		}
	}
	for _, c := range qei.subpasses[subpassI].colorAttachments {
		if isStoreAtt(c) {
			noDsAttStoreOp(ctx, ft, sc, m, c, nil)
		}
	}
	for _, i := range qei.subpasses[subpassI].inputAttachments {
		if isStoreAtt(i) {
			noDsAttStoreOp(ctx, ft, sc, m, i, nil)
		}
	}
	if isStoreAtt(qei.subpasses[subpassI].depthStencilAttachment) {
		dsAttStoreOp(ctx, ft, sc, m, qei.subpasses[subpassI].depthStencilAttachment)
	}
	for _, modified := range qei.subpasses[subpassI].modifiedDescriptorData {
		bh := sc.cmd.newBehavior(ctx, sc, m, qei)
		modify(ctx, bh, modified)
		read(ctx, bh, qei.subpass)
		ft.AddBehavior(ctx, bh)
	}
}

func (qei *queueExecutionState) endSubpass(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behavior,
	sc submittedCommand, m *vulkanMachine) {
	qei.emitSubpassOutput(ctx, ft, sc, m)
	read(ctx, bh, qei.subpass)
}

func (qei *queueExecutionState) beginRenderPass(ctx context.Context,
	vb *FootprintBuilder, bh *dependencygraph.Behavior,
	rp RenderPassObjectʳ, fb FramebufferObjectʳ) {
	read(ctx, bh, vkHandle(rp.VulkanHandle()))
	read(ctx, bh, vkHandle(fb.VulkanHandle()))
	qei.framebuffer = fb
	qei.subpasses = []subpassInfo{}

	// Record which subpass that loads or stores the attachments. A subpass loads
	// an attachment if the attachment is first used in that subpass. A subpass
	// stores an attachment if the subpass is the last use of that attachment.
	attLoadSubpass := map[uint32]uint32{}
	attStoreSubpass := map[uint32]uint32{}
	attStoreAttInfo := map[uint32]*subpassAttachmentInfo{}
	recordAttachment := func(ai, si uint32) *subpassAttachmentInfo {
		viewObj := fb.ImageAttachments().Get(ai)
		imgObj := viewObj.Image()
		imgLayout, imgData := vb.getImageLayoutAndData(ctx, bh, imgObj.VulkanHandle())
		attDesc := rp.AttachmentDescriptions().Get(ai)
		fullImageData := false
		switch viewObj.Type() {
		case VkImageViewType_VK_IMAGE_VIEW_TYPE_2D,
			VkImageViewType_VK_IMAGE_VIEW_TYPE_2D_ARRAY:
			if viewObj.SubresourceRange().BaseArrayLayer() == uint32(0) &&
				imgObj.Info().ArrayLayers() == viewObj.SubresourceRange().LayerCount() &&
				imgObj.Info().ImageType() == VkImageType_VK_IMAGE_TYPE_2D &&
				imgObj.Info().Extent().Width() == fb.Width() &&
				imgObj.Info().Extent().Height() == fb.Height() &&
				fb.Layers() == imgObj.Info().ArrayLayers() {
				fullImageData = true
			}
		}
		attachmentInfo := &subpassAttachmentInfo{fullImageData, imgData, imgLayout, attDesc}
		if _, ok := attLoadSubpass[ai]; !ok {
			attLoadSubpass[ai] = si
			qei.subpasses[si].loadAttachments = append(
				qei.subpasses[si].loadAttachments, attachmentInfo)
		}
		attStoreSubpass[ai] = si
		attStoreAttInfo[ai] = attachmentInfo
		return attachmentInfo
	}
	defer func() {
		for ai, si := range attStoreSubpass {
			qei.subpasses[si].storeAttachments = append(
				qei.subpasses[si].storeAttachments, attStoreAttInfo[ai])
		}
	}()

	for _, subpass := range rp.SubpassDescriptions().Keys() {
		desc := rp.SubpassDescriptions().Get(subpass)
		qei.subpasses = append(qei.subpasses, subpassInfo{})
		if subpass != uint32(len(qei.subpasses)-1) {
			log.E(ctx, "FootprintBuilder: Cannot get subpass info, subpass: %v, length of info: %v",
				subpass, uint32(len(qei.subpasses)))
		}
		colorAs := map[uint32]struct{}{}
		resolveAs := map[uint32]struct{}{}
		inputAs := map[uint32]struct{}{}

		for _, ref := range desc.ColorAttachments().All() {
			if ref.Attachment() != vkAttachmentUnused {
				colorAs[ref.Attachment()] = struct{}{}
			}
		}
		for _, ref := range desc.ResolveAttachments().All() {
			if ref.Attachment() != vkAttachmentUnused {
				resolveAs[ref.Attachment()] = struct{}{}
			}
		}
		for _, ref := range desc.InputAttachments().All() {
			if ref.Attachment() != vkAttachmentUnused {
				inputAs[ref.Attachment()] = struct{}{}
			}
		}
		// TODO: handle preserveAttachments

		for _, viewObj := range fb.ImageAttachments().All() {
			if read(ctx, bh, vkHandle(viewObj.VulkanHandle())) {
				read(ctx, bh, vkHandle(viewObj.Image().VulkanHandle()))
			}
		}

		for _, ai := range rp.AttachmentDescriptions().Keys() {
			if _, ok := colorAs[ai]; ok {
				qei.subpasses[subpass].colorAttachments = append(
					qei.subpasses[subpass].colorAttachments,
					recordAttachment(ai, subpass))
			}
			if _, ok := resolveAs[ai]; ok {
				qei.subpasses[subpass].resolveAttachments = append(
					qei.subpasses[subpass].resolveAttachments,
					recordAttachment(ai, subpass))
			}
			if _, ok := inputAs[ai]; ok {
				qei.subpasses[subpass].inputAttachments = append(
					qei.subpasses[subpass].inputAttachments,
					recordAttachment(ai, subpass))
			}
		}
		if !desc.DepthStencilAttachment().IsNil() {
			dsAi := desc.DepthStencilAttachment().Attachment()
			if dsAi != vkAttachmentUnused {
				viewObj := fb.ImageAttachments().Get(dsAi)
				imgObj := viewObj.Image()
				imgLayout, imgData := vb.getImageLayoutAndData(ctx, bh, imgObj.VulkanHandle())
				attDesc := rp.AttachmentDescriptions().Get(dsAi)
				fullImageData := false
				switch viewObj.Type() {
				case VkImageViewType_VK_IMAGE_VIEW_TYPE_2D,
					VkImageViewType_VK_IMAGE_VIEW_TYPE_2D_ARRAY:
					if viewObj.SubresourceRange().BaseArrayLayer() == uint32(0) &&
						imgObj.Info().ArrayLayers() == viewObj.SubresourceRange().LayerCount() &&
						imgObj.Info().ImageType() == VkImageType_VK_IMAGE_TYPE_2D &&
						imgObj.Info().Extent().Width() == fb.Width() &&
						imgObj.Info().Extent().Height() == fb.Height() &&
						fb.Layers() == imgObj.Info().ArrayLayers() {
						fullImageData = true
					}
				}
				qei.subpasses[subpass].depthStencilAttachment = &subpassAttachmentInfo{
					fullImageData, imgData, imgLayout, attDesc}
			}
		}
	}
	qei.subpass = &subpassIndex{0}
	qei.startSubpass(ctx, bh)
}

func (qei *queueExecutionState) nextSubpass(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behavior,
	sc submittedCommand, m *vulkanMachine) {
	qei.endSubpass(ctx, ft, bh, sc, m)
	qei.subpass.val++
	qei.startSubpass(ctx, bh)
}

func (qei *queueExecutionState) endRenderPass(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behavior,
	sc submittedCommand, m *vulkanMachine) {
	qei.endSubpass(ctx, ft, bh, sc, m)
}

type renderpass struct {
	begin label
	end   label
}

type commandBuffer struct {
	begin           label
	end             label
	renderPassBegin label
}

type resBinding struct {
	resourceOffset uint64
	bindSize       uint64
	backingData    dependencygraph.DefUseVariable
}

// resBinding implements interface memBinding
func (bd *resBinding) size() uint64 {
	return bd.bindSize
}

func (bd *resBinding) span() interval.U64Span {
	return interval.U64Span{Start: bd.resourceOffset, End: bd.resourceOffset + bd.size()}
}

func (bd *resBinding) shrink(offset, size uint64) error {
	if size == vkWholeSize {
		size = bd.size() - offset
	}
	if (offset+size < offset) || (offset+size > bd.size()) {
		return shrinkOutOfMemBindingBound{bd, offset, size}
	}
	if sp, isSpan := bd.backingData.(memorySpan); isSpan {
		bd.bindSize = size
		bd.resourceOffset += offset
		sp.span.Start += offset
		sp.span.End = sp.span.Start + size
		bd.backingData = sp
		return nil
	}
	if offset != 0 || size != bd.size() {
		return fmt.Errorf("Cannot shrink a resBinding whose backing data is not a memorySpan into different size and offset than the original one")
	}
	return nil
}

func (bd *resBinding) duplicate() memBinding {
	newB := *bd
	return &newB
}

func newResBinding(ctx context.Context, bh *dependencygraph.Behavior,
	resOffset, size uint64, res dependencygraph.DefUseVariable) *resBinding {
	d := &resBinding{resourceOffset: resOffset, bindSize: size, backingData: res}
	if bh != nil {
		write(ctx, bh, d)
	}
	return d
}

func newSpanResBinding(ctx context.Context, bh *dependencygraph.Behavior,
	memory VkDeviceMemory, resOffset, size, memoryOffset uint64) *resBinding {
	return newResBinding(ctx, bh, resOffset, size, memorySpan{
		span:   interval.U64Span{Start: memoryOffset, End: memoryOffset + size},
		memory: memory,
	})
}

func newNonSpanResBinding(ctx context.Context, bh *dependencygraph.Behavior,
	size uint64) *resBinding {
	return newResBinding(ctx, bh, 0, size, newLabel())
}

func (bd *resBinding) newSubBinding(ctx context.Context,
	bh *dependencygraph.Behavior, offset, size uint64) (*resBinding, error) {
	subBinding, _ := bd.duplicate().(*resBinding)
	if err := subBinding.shrink(offset, size); err != nil {
		return nil, err
	}
	if bh != nil {
		write(ctx, bh, subBinding)
	}
	return subBinding, nil
}

func (*resBinding) DefUseVariable() {}

// Implements the interval.List interface for resBinding slices
type resBindingList memBindingList

func (rl resBindingList) resBindings() []*resBinding {
	ret := []*resBinding{}
	for _, b := range rl {
		if rb, ok := b.(*resBinding); ok {
			ret = append(ret, rb)
		}
	}
	return ret
}

func addResBinding(ctx context.Context, l resBindingList, b *resBinding) resBindingList {
	var err error
	ml := memBindingList(l)
	ml, err = addBinding(ml, b)
	if err != nil {
		log.E(ctx, "FootprintBuilder: %s", err.Error())
		return l
	}
	return resBindingList(ml)
}

func (l resBindingList) getSubBindingList(ctx context.Context,
	bh *dependencygraph.Behavior, offset, size uint64) resBindingList {
	subBindings := resBindingList{}
	if offset+size < offset {
		// overflow
		size = vkWholeSize - offset
	}
	first, count := interval.Intersect(memBindingList(l),
		interval.U64Span{Start: offset, End: offset + size})
	if count == 0 {
		return subBindings
	} else {
		bl := l.resBindings()
		for i := first; i < first+count; i++ {
			start := bl[i].span().Start
			end := bl[i].span().End
			if offset > start {
				start = offset
			}
			if offset+size < end {
				end = offset + size
			}
			if bh != nil {
				read(ctx, bh, bl[i])
			}
			newB, err := bl[i].newSubBinding(ctx, bh, start-bl[i].span().Start, end-start)
			if err != nil {
				log.E(ctx, "FootprintBuilder: %s", err.Error())
			}
			if newB != nil {
				subBindings = append(subBindings, newB)
			}
		}
	}
	return subBindings
}

func (l resBindingList) getBoundData(ctx context.Context,
	bh *dependencygraph.Behavior, offset, size uint64) []dependencygraph.DefUseVariable {
	data := []dependencygraph.DefUseVariable{}
	bindingList := l.getSubBindingList(ctx, bh, offset, size)
	for _, b := range bindingList.resBindings() {
		if b == nil {
			// skip invalid bindings
			continue
		}
		data = append(data, b.backingData)
	}
	return data
}

type descriptor struct {
	ty VkDescriptorType
	// for image descriptor
	img VkImage
	// only used for sampler and sampler combined descriptors
	sampler vkHandle
	// for buffer descriptor
	buf       VkBuffer
	bufOffset VkDeviceSize
	bufRng    VkDeviceSize
}

func (*descriptor) DefUseVariable() {}

type descriptorSet struct {
	descriptors            api.SubCmdIdxTrie
	descriptorCounts       map[uint64]uint64 // binding -> descriptor count of that binding
	dynamicDescriptorCount uint64
}

func newDescriptorSet() *descriptorSet {
	return &descriptorSet{
		descriptors:            api.SubCmdIdxTrie{},
		descriptorCounts:       map[uint64]uint64{},
		dynamicDescriptorCount: uint64(0),
	}
}

func (ds *descriptorSet) reserveDescriptor(bi, di uint64) {
	ds.descriptors.SetValue([]uint64{bi, di}, &descriptor{})
	if _, ok := ds.descriptorCounts[bi]; !ok {
		ds.descriptorCounts[bi] = uint64(0)
	}
	ds.descriptorCounts[bi]++
}

func (ds *descriptorSet) getDescriptor(ctx context.Context,
	bh *dependencygraph.Behavior, bi, di uint64) *descriptor {
	if v := ds.descriptors.Value([]uint64{bi, di}); v != nil {
		if d, ok := v.(*descriptor); ok {
			read(ctx, bh, d)
			return d
		}
		log.E(ctx, "FootprintBuilder: Not *descriptor type in descriptorSet: %v, with "+
			"binding: %v, array index: %v", *ds, bi, di)
		return nil
	}
	log.E(ctx, "FootprintBuilder: Read target descriptor does not exists in "+
		"descriptorSet: %v, with binding: %v, array index: %v", *ds, bi, di)
	return nil
}

func (ds *descriptorSet) setDescriptor(ctx context.Context,
	bh *dependencygraph.Behavior, bi, di uint64, ty VkDescriptorType,
	vkImg VkImage, sampler vkHandle, vkBuf VkBuffer, boundOffset, rng VkDeviceSize) {
	// dataBindingList resBindingList, sampler vkHandle, rng VkDeviceSize) {
	if v := ds.descriptors.Value([]uint64{bi, di}); v != nil {
		if d, ok := v.(*descriptor); ok {
			write(ctx, bh, d)
			d.img = vkImg
			d.buf = vkBuf
			d.sampler = sampler
			d.ty = ty
			d.bufRng = rng
			d.bufOffset = boundOffset
			if ty == VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC ||
				ty == VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC {
				ds.dynamicDescriptorCount++
			}
		} else {
			log.E(ctx, "FootprintBuilder: Not *descriptor type in descriptorSet: %v, with "+
				"binding: %v, array index: %v", *ds, bi, di)
		}
	} else {
		log.E(ctx, "FootprintBuilder: Write target descriptor does not exist in "+
			"descriptorSet: %v, with binding: %v, array index: %v", *ds, bi, di)
	}
}

func (ds *descriptorSet) useDescriptors(ctx context.Context, vb *FootprintBuilder,
	bh *dependencygraph.Behavior, dynamicOffsets []uint32) []dependencygraph.DefUseVariable {
	modified := []dependencygraph.DefUseVariable{}
	doi := 0
	for binding, count := range ds.descriptorCounts {
		for di := uint64(0); di < count; di++ {
			d := ds.getDescriptor(ctx, bh, binding, di)
			if d != nil {
				read(ctx, bh, d.sampler)
				switch d.ty {
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
					data := vb.getImageData(ctx, bh, d.img)
					modify(ctx, bh, data...)
					modified = append(modified, data...)
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER:
					// pass, as the sampler has been 'read' before the switch
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
					VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
					data := vb.getImageData(ctx, bh, d.img)
					read(ctx, bh, data...)
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
					VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
					data := vb.getBufferData(ctx, bh, d.buf, uint64(d.bufOffset), uint64(d.bufRng))
					modify(ctx, bh, data...)
					modified = append(modified, data...)
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
					if doi < len(dynamicOffsets) {
						data := vb.getBufferData(ctx, bh, d.buf,
							uint64(dynamicOffsets[doi])+uint64(d.bufOffset), uint64(d.bufRng))
						doi++
						modify(ctx, bh, data...)
						modified = append(modified, data...)
					} else {
						log.E(ctx, "FootprintBuilder: DescriptorSet: %v has more dynamic descriptors than reserved dynamic offsets", *ds)
					}
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
					VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
					data := vb.getBufferData(ctx, bh, d.buf, uint64(d.bufOffset), uint64(d.bufRng))
					read(ctx, bh, data...)
				case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
					if doi < len(dynamicOffsets) {
						data := vb.getBufferData(ctx, bh, d.buf,
							uint64(dynamicOffsets[doi])+uint64(d.bufOffset), uint64(d.bufRng))
						doi++
						read(ctx, bh, data...)
					} else {
						log.E(ctx, "FootprintBuilder: DescriptorSet: %v has more dynamic descriptors than reserved dynamic offsets", *ds)
					}
				}
			}
		}
	}
	return modified
}

func (ds *descriptorSet) writeDescriptors(ctx context.Context,
	cmd api.Cmd, s *api.GlobalState, vb *FootprintBuilder,
	bh *dependencygraph.Behavior,
	write VkWriteDescriptorSet) {
	l := s.MemoryLayout
	dstElm := uint64(write.DstArrayElement())
	count := uint64(write.DescriptorCount())
	dstBinding := uint64(write.DstBinding())
	updateDstForOverflow := func() {
		if dstElm >= ds.descriptorCounts[dstBinding] {
			dstBinding++
			dstElm = uint64(0)
		}
	}
	switch write.DescriptorType() {
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
		for _, imageInfo := range write.PImageInfo().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			updateDstForOverflow()
			sampler := vkNullHandle
			vkImg := VkImage(0)
			if write.DescriptorType() != VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER &&
				read(ctx, bh, vkHandle(imageInfo.ImageView())) {
				vkView := imageInfo.ImageView()
				vkImg = GetState(s).ImageViews().Get(vkView).Image().VulkanHandle()
			}
			if (write.DescriptorType() == VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER ||
				write.DescriptorType() == VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER) &&
				read(ctx, bh, vkHandle(imageInfo.Sampler())) {
				sampler = vkHandle(imageInfo.Sampler())
			}
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType(),
				vkImg, sampler, VkBuffer(0), 0, 0)
			dstElm++
		}
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
		for _, bufferInfo := range write.PBufferInfo().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			updateDstForOverflow()
			vkBuf := bufferInfo.Buffer()
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType(), VkImage(0),
				vkNullHandle, vkBuf, bufferInfo.Offset(), bufferInfo.Range())
			dstElm++
		}
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
		for _, bufferInfo := range write.PBufferInfo().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			updateDstForOverflow()
			vkBuf := bufferInfo.Buffer()
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType(), VkImage(0),
				vkNullHandle, vkBuf, bufferInfo.Offset(), bufferInfo.Range())
			dstElm++
		}
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
		for _, vkBufView := range write.PTexelBufferView().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			updateDstForOverflow()
			read(ctx, bh, vkHandle(vkBufView))
			bufView := GetState(s).BufferViews().Get(vkBufView)
			vkBuf := GetState(s).BufferViews().Get(vkBufView).Buffer().VulkanHandle()
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType(),
				VkImage(0), vkNullHandle, vkBuf, bufView.Offset(), bufView.Range())
			dstElm++
		}
	}
}

func (ds *descriptorSet) copyDescriptors(ctx context.Context,
	cmd api.Cmd, s *api.GlobalState, bh *dependencygraph.Behavior,
	srcDs *descriptorSet, copy VkCopyDescriptorSet) {
	dstElm := uint64(copy.DstArrayElement())
	srcElm := uint64(copy.SrcArrayElement())
	dstBinding := uint64(copy.DstBinding())
	srcBinding := uint64(copy.SrcBinding())
	updateDstAndSrcForOverflow := func() {
		if dstElm >= ds.descriptorCounts[dstBinding] {
			dstBinding++
			dstElm = uint64(0)
		}
		if srcElm >= srcDs.descriptorCounts[srcBinding] {
			srcBinding++
			srcElm = uint64(0)
		}
	}
	for i := uint64(0); i < uint64(copy.DescriptorCount()); i++ {
		updateDstAndSrcForOverflow()
		srcD := srcDs.getDescriptor(ctx, bh, srcBinding, srcElm)
		ds.setDescriptor(ctx, bh, dstBinding, dstElm, srcD.ty,
			srcD.img, srcD.sampler, srcD.buf, srcD.bufOffset, srcD.bufRng)
		srcElm++
		dstElm++
	}
}

type boundDescriptorSet struct {
	descriptorSet  *descriptorSet
	dynamicOffsets []uint32
}

func newBoundDescriptorSet(ctx context.Context, bh *dependencygraph.Behavior,
	ds *descriptorSet, getDynamicOffset func() uint32) *boundDescriptorSet {
	bds := &boundDescriptorSet{descriptorSet: ds}
	bds.dynamicOffsets = make([]uint32, ds.dynamicDescriptorCount)
	for i := range bds.dynamicOffsets {
		bds.dynamicOffsets[i] = getDynamicOffset()
	}
	write(ctx, bh, bds)
	return bds
}

func (*boundDescriptorSet) DefUseVariable() {}

type sparseImageMemoryBinding struct {
	backingData dependencygraph.DefUseVariable
}

func newSparseImageMemoryBinding(ctx context.Context, bh *dependencygraph.Behavior,
	memory VkDeviceMemory,
	memoryOffset, size uint64) *sparseImageMemoryBinding {
	b := &sparseImageMemoryBinding{
		backingData: memorySpan{
			span: interval.U64Span{
				Start: memoryOffset,
				End:   memoryOffset + size,
			},
			memory: memory,
		}}
	write(ctx, bh, b)
	return b
}

func (*sparseImageMemoryBinding) DefUseVariable() {}

type imageLayoutAndData struct {
	layout     label
	opaqueData resBindingList
	sparseData map[VkImageAspectFlags]map[uint32]map[uint32]map[uint64]*sparseImageMemoryBinding
}

func newImageLayoutAndData(ctx context.Context,
	bh *dependencygraph.Behavior) *imageLayoutAndData {
	d := &imageLayoutAndData{layout: newLabel()}
	d.sparseData = map[VkImageAspectFlags]map[uint32]map[uint32]map[uint64]*sparseImageMemoryBinding{}
	write(ctx, bh, d.layout)
	return d
}

// FootprintBuilder implements the FootprintBuilder interface and builds
// Footprint for Vulkan commands.
type FootprintBuilder struct {
	machine *vulkanMachine

	// commands
	commands map[VkCommandBuffer][]*commandBufferCommand

	// coherent memory mapping
	mappedCoherentMemories map[VkDeviceMemory]DeviceMemoryObjectʳ

	// Vulkan handle states
	semaphoreSignals map[VkSemaphore]label
	fences           map[VkFence]*fence
	events           map[VkEvent]*event
	querypools       map[VkQueryPool]*queryPool
	commandBuffers   map[VkCommandBuffer]*commandBuffer
	images           map[VkImage]*imageLayoutAndData
	buffers          map[VkBuffer]resBindingList
	descriptorSets   map[VkDescriptorSet]*descriptorSet

	// execution info
	executionStates map[VkQueue]*queueExecutionState
	submitInfos     map[api.CmdID] /*ID of VkQueueSubmit*/ *queueSubmitInfo
	submitIDs       map[*VkQueueSubmit]api.CmdID

	// presentation info
	swapchainImageAcquired  map[VkSwapchainKHR][]label
	swapchainImagePresented map[VkSwapchainKHR][]label
}

// getImageData records a read operation of the Vulkan image handle, a read
// operation of the image layout, a read operation of the image bindings, then
// returns the underlying data.
func (vb *FootprintBuilder) getImageData(ctx context.Context,
	bh *dependencygraph.Behavior, vkImg VkImage) []dependencygraph.DefUseVariable {
	if bh != nil {
		if !read(ctx, bh, vkHandle(vkImg)) {
			return []dependencygraph.DefUseVariable{}
		}
		if !read(ctx, bh, vb.images[vkImg].layout) {
			return []dependencygraph.DefUseVariable{}
		}
	}
	data := vb.images[vkImg].opaqueData.getBoundData(ctx, bh, 0, vkWholeSize)
	for _, aspecti := range vb.images[vkImg].sparseData {
		for _, layeri := range aspecti {
			for _, leveli := range layeri {
				for _, blocki := range leveli {
					if bh != nil {
						read(ctx, bh, blocki)
					}
					data = append(data, blocki.backingData)
				}
			}
		}
	}
	return data
}

// getImageOpaqueData records a read operation of the Vulkan image handle, a
// read operation of the image layout, a read operation of the overlapping
// bindings, then returns the underlying data. This only works for opaque image
// bindings (non-sparse-resident bindings), and the image must NOT be created
// by swapchains.
func (vb *FootprintBuilder) getImageOpaqueData(ctx context.Context,
	bh *dependencygraph.Behavior, vkImg VkImage, offset, size uint64) []dependencygraph.DefUseVariable {
	read(ctx, bh, vkHandle(vkImg))
	data := vb.images[vkImg].opaqueData.getBoundData(ctx, bh, offset, size)
	return data
}

func (vb *FootprintBuilder) getSparseImageBindData(ctx context.Context,
	cmd api.Cmd, id api.CmdID, s *api.GlobalState, bh *dependencygraph.Behavior,
	vkImg VkImage, bind VkSparseImageMemoryBind) []dependencygraph.DefUseVariable {
	data := []dependencygraph.DefUseVariable{}
	vb.visitBlocksInVkSparseImageMemoryBind(ctx, cmd, id, s, bh, vkImg, bind, func(
		aspects VkImageAspectFlags, layer, level uint32, blockIndex, memoryOffset uint64) {
		if _, ok := vb.images[vkImg].sparseData[aspects]; !ok {
			return
		}
		if _, ok := vb.images[vkImg].sparseData[aspects][layer]; !ok {
			return
		}
		if _, ok := vb.images[vkImg].sparseData[aspects][layer][level]; !ok {
			return
		}
		if _, ok := vb.images[vkImg].sparseData[aspects][layer][level][blockIndex]; !ok {
			return
		}
		data = append(data, vb.images[vkImg].sparseData[aspects][layer][level][blockIndex].backingData)
	})
	return data
}

// getImageLayoutAndData records a read operation of the Vulkan handle, a read
// operation of the image binding, but not the image layout. Then returns the
// image layout label and underlying data.
func (vb *FootprintBuilder) getImageLayoutAndData(ctx context.Context,
	bh *dependencygraph.Behavior, vkImg VkImage) (label, []dependencygraph.DefUseVariable) {
	read(ctx, bh, vkHandle(vkImg))
	return vb.images[vkImg].layout, vb.getImageData(ctx, bh, vkImg)
}

func (vb *FootprintBuilder) addOpaqueImageMemBinding(ctx context.Context,
	bh *dependencygraph.Behavior, vkImg VkImage, vkMem VkDeviceMemory, resOffset,
	size, memOffset uint64) {
	vb.images[vkImg].opaqueData = addResBinding(ctx, vb.images[vkImg].opaqueData,
		newSpanResBinding(ctx, bh, vkMem, resOffset, size, memOffset))
}

func (vb *FootprintBuilder) addSwapchainImageMemBinding(ctx context.Context,
	bh *dependencygraph.Behavior, vkImg VkImage) {
	vb.images[vkImg].opaqueData = addResBinding(ctx, vb.images[vkImg].opaqueData,
		newNonSpanResBinding(ctx, bh, vkWholeSize))
}

// Traverse through the blocks covered by the given bind.
func (vb *FootprintBuilder) visitBlocksInVkSparseImageMemoryBind(ctx context.Context,
	cmd api.Cmd, id api.CmdID, s *api.GlobalState, bh *dependencygraph.Behavior,
	vkImg VkImage, bind VkSparseImageMemoryBind, cb func(aspects VkImageAspectFlags,
		layer, level uint32, blockIndex, memOffset uint64)) {
	imgObj := GetState(s).Images().Get(vkImg)

	aspect := bind.Subresource().AspectMask()
	layer := bind.Subresource().ArrayLayer()
	level := bind.Subresource().MipLevel()

	gran, found := sparseImageMemoryBindGranularity(ctx, imgObj, bind)
	if found {
		width, _ := subGetMipSize(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, imgObj.Info().Extent().Width(), level)
		height, _ := subGetMipSize(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, imgObj.Info().Extent().Height(), level)
		wb, _ := subRoundUpTo(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, width, gran.Width())
		hb, _ := subRoundUpTo(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, height, gran.Height())
		xe, _ := subRoundUpTo(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, bind.Extent().Width(), gran.Width())
		ye, _ := subRoundUpTo(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, bind.Extent().Height(), gran.Height())
		ze, _ := subRoundUpTo(ctx, cmd, id, nil, s, nil, cmd.Thread(), nil, bind.Extent().Depth(), gran.Depth())
		blockSize := uint64(imgObj.MemoryRequirements().Alignment())
		for zi := uint32(0); zi < ze; zi++ {
			for yi := uint32(0); yi < ye; yi++ {
				for xi := uint32(0); xi < xe; xi++ {
					loc := xi + yi*wb + zi*wb*hb
					memoryOffset := uint64(bind.MemoryOffset()) + uint64(loc)*blockSize
					cb(aspect, layer, level, uint64(loc), memoryOffset)
				}
			}
		}
	}
}

func (vb *FootprintBuilder) addSparseImageMemBinding(ctx context.Context,
	cmd api.Cmd, id api.CmdID,
	s *api.GlobalState, bh *dependencygraph.Behavior, vkImg VkImage,
	bind VkSparseImageMemoryBind) {
	blockSize := GetState(s).Images().Get(vkImg).MemoryRequirements().Alignment()
	vb.visitBlocksInVkSparseImageMemoryBind(ctx, cmd, id, s, bh, vkImg, bind,
		func(aspects VkImageAspectFlags, layer, level uint32, blockIndex, memoryOffset uint64) {
			if _, ok := vb.images[vkImg].sparseData[aspects]; !ok {
				vb.images[vkImg].sparseData[aspects] = map[uint32]map[uint32]map[uint64]*sparseImageMemoryBinding{}
			}
			if _, ok := vb.images[vkImg].sparseData[aspects][layer]; !ok {
				vb.images[vkImg].sparseData[aspects][layer] = map[uint32]map[uint64]*sparseImageMemoryBinding{}
			}
			if _, ok := vb.images[vkImg].sparseData[aspects][layer][level]; !ok {
				vb.images[vkImg].sparseData[aspects][layer][level] = map[uint64]*sparseImageMemoryBinding{}
			}
			vb.images[vkImg].sparseData[aspects][layer][level][blockIndex] = newSparseImageMemoryBinding(
				ctx, bh, bind.Memory(), memoryOffset, uint64(blockSize))
		})
}

func (vb *FootprintBuilder) getBufferData(ctx context.Context,
	bh *dependencygraph.Behavior, vkBuf VkBuffer,
	offset, size uint64) []dependencygraph.DefUseVariable {
	read(ctx, bh, vkHandle(vkBuf))
	for _, bb := range vb.buffers[vkBuf].resBindings() {
		read(ctx, bh, bb)
	}
	return vb.buffers[vkBuf].getBoundData(ctx, bh, offset, size)
}

func (vb *FootprintBuilder) addBufferMemBinding(ctx context.Context,
	bh *dependencygraph.Behavior, vkBuf VkBuffer,
	vkMem VkDeviceMemory, resOffset, size, memOffset uint64) {
	vb.buffers[vkBuf] = addResBinding(ctx, vb.buffers[vkBuf],
		newSpanResBinding(ctx, bh, vkMem, resOffset, size, memOffset))
}

func (vb *FootprintBuilder) newCommand(ctx context.Context,
	bh *dependencygraph.Behavior, vkCb VkCommandBuffer) *commandBufferCommand {
	cbc := &commandBufferCommand{}
	read(ctx, bh, vkHandle(vkCb))
	read(ctx, bh, vb.commandBuffers[vkCb].begin)
	write(ctx, bh, cbc)
	vb.commands[vkCb] = append(vb.commands[vkCb], cbc)
	return cbc
}

func newFootprintBuilder() *FootprintBuilder {
	return &FootprintBuilder{
		machine:                 newVulkanMachine(),
		commands:                map[VkCommandBuffer][]*commandBufferCommand{},
		mappedCoherentMemories:  map[VkDeviceMemory]DeviceMemoryObjectʳ{},
		semaphoreSignals:        map[VkSemaphore]label{},
		fences:                  map[VkFence]*fence{},
		events:                  map[VkEvent]*event{},
		querypools:              map[VkQueryPool]*queryPool{},
		commandBuffers:          map[VkCommandBuffer]*commandBuffer{},
		images:                  map[VkImage]*imageLayoutAndData{},
		buffers:                 map[VkBuffer]resBindingList{},
		descriptorSets:          map[VkDescriptorSet]*descriptorSet{},
		executionStates:         map[VkQueue]*queueExecutionState{},
		submitInfos:             map[api.CmdID]*queueSubmitInfo{},
		submitIDs:               map[*VkQueueSubmit]api.CmdID{},
		swapchainImageAcquired:  map[VkSwapchainKHR][]label{},
		swapchainImagePresented: map[VkSwapchainKHR][]label{},
	}
}

func (vb *FootprintBuilder) rollOutExecuted(ctx context.Context,
	ft *dependencygraph.Footprint,
	executedCommands []api.SubCmdIdx) {
	for _, executedFCI := range executedCommands {
		submitID := executedFCI[0]
		submitinfo := vb.submitInfos[api.CmdID(submitID)]
		if !submitinfo.began {
			bh := dependencygraph.NewBehavior(api.SubCmdIdx{submitID}, vb.machine)
			for _, sp := range submitinfo.waitSemaphores {
				if read(ctx, bh, vkHandle(sp)) {
					modify(ctx, bh, vb.semaphoreSignals[sp])
				}
			}
			// write(ctx, bh, submitinfo.queued)
			ft.AddBehavior(ctx, bh)
			submitinfo.began = true
		}
		submittedCmd := submitinfo.pendingCommands[0]
		if executedFCI.Equals(submittedCmd.id) {
			execInfo := vb.executionStates[submitinfo.queue]
			execInfo.currentSubmitInfo = submitinfo
			execInfo.updateCurrentCommand(ctx, executedFCI)
			submittedCmd.runCommand(ctx, ft, vb.machine, execInfo)
		} else {
			log.E(ctx, "FootprintBuilder: Execution order differs from submission order. "+
				"Index of executed command: %v, Index of submitted command: %v",
				executedFCI, submittedCmd.id)
			return
		}
		// Remove the front submitted command.
		submitinfo.pendingCommands =
			submitinfo.pendingCommands[1:]
		// After the last command of the submit, we need to add a behavior for
		// semaphore and fence signaling.
		if len(submitinfo.pendingCommands) == 0 {
			bh := dependencygraph.NewBehavior(api.SubCmdIdx{
				executedFCI[0]}, vb.machine)
			// add writes to the semaphores and fences
			read(ctx, bh, submitinfo.queued)
			write(ctx, bh, submitinfo.done)
			for _, sp := range submitinfo.signalSemaphores {
				if read(ctx, bh, vkHandle(sp)) {
					write(ctx, bh, vb.semaphoreSignals[sp])
				}
			}
			if read(ctx, bh, vkHandle(submitinfo.signalFence)) {
				write(ctx, bh, vb.fences[submitinfo.signalFence].signal)
			}
			ft.AddBehavior(ctx, bh)
		}
	}
}

func (vb *FootprintBuilder) recordReadsWritesModifies(
	ctx context.Context, ft *dependencygraph.Footprint, bh *dependencygraph.Behavior,
	vkCb VkCommandBuffer, reads []dependencygraph.DefUseVariable,
	writes []dependencygraph.DefUseVariable, modifies []dependencygraph.DefUseVariable) {
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand, execInfo *queueExecutionState) {
		cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
		read(ctx, cbh, reads...)
		write(ctx, cbh, writes...)
		modify(ctx, cbh, modifies...)
		ft.AddBehavior(ctx, cbh)
	}
}

func (vb *FootprintBuilder) recordModifingDynamicStates(
	ctx context.Context, ft *dependencygraph.Footprint, bh *dependencygraph.Behavior,
	vkCb VkCommandBuffer) {
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand, execInfo *queueExecutionState) {
		cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
		modify(ctx, cbh, execInfo.currentCmdBufState.dynamicState)
		ft.AddBehavior(ctx, cbh)
	}
}

func (vb *FootprintBuilder) useBoundDescriptorSets(ctx context.Context,
	bh *dependencygraph.Behavior,
	cmdBufState *commandBufferExecutionState) []dependencygraph.DefUseVariable {
	modified := []dependencygraph.DefUseVariable{}
	for _, bds := range cmdBufState.descriptorSets {
		read(ctx, bh, bds)
		ds := bds.descriptorSet
		modified = append(modified, ds.useDescriptors(ctx, vb, bh, bds.dynamicOffsets)...)
	}
	return modified
}

func (vb *FootprintBuilder) draw(ctx context.Context,
	bh *dependencygraph.Behavior, execInfo *queueExecutionState) {
	read(ctx, bh, execInfo.subpass)
	read(ctx, bh, execInfo.currentCmdBufState.pipeline)
	read(ctx, bh, execInfo.currentCmdBufState.dynamicState)
	subpassI := execInfo.subpass.val
	for _, b := range execInfo.currentCmdBufState.vertexBufferResBindings {
		read(ctx, bh, b.getBoundData(ctx, bh, 0, vkWholeSize)...)
	}
	modifiedDs := vb.useBoundDescriptorSets(ctx, bh, execInfo.currentCmdBufState)
	execInfo.subpasses[execInfo.subpass.val].modifiedDescriptorData = append(
		execInfo.subpasses[execInfo.subpass.val].modifiedDescriptorData,
		modifiedDs...)
	if execInfo.currentCmdBufState.indexBufferResBindings != nil {
		read(ctx, bh, execInfo.currentCmdBufState.indexBufferResBindings.getBoundData(
			ctx, bh, 0, vkWholeSize)...)
	}
	for _, input := range execInfo.subpasses[subpassI].inputAttachments {
		read(ctx, bh, input.data...)
	}
	for _, color := range execInfo.subpasses[subpassI].colorAttachments {
		modify(ctx, bh, color.data...)
	}
	if execInfo.subpasses[subpassI].depthStencilAttachment != nil {
		dsAtt := execInfo.subpasses[subpassI].depthStencilAttachment
		modify(ctx, bh, dsAtt.data...)
	}
}

func (vb *FootprintBuilder) keepSubmittedCommandAlive(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behavior,
	vkCb VkCommandBuffer) {
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand, execInfo *queueExecutionState) {
		cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
		cbh.Alive = true
		ft.AddBehavior(ctx, cbh)
	}
}

func (t VkIndexType) size() int {
	switch t {
	case VkIndexType_VK_INDEX_TYPE_UINT16:
		return 2
	case VkIndexType_VK_INDEX_TYPE_UINT32:
		return 4
	default:
		return 0
	}
	return 0
}

func (vb *FootprintBuilder) readBoundIndexBuffer(ctx context.Context,
	bh *dependencygraph.Behavior, execInfo *queueExecutionState, cmd api.Cmd) {
	indexSize := uint64(execInfo.currentCmdBufState.indexType.size())
	if indexSize == uint64(0) {
		log.E(ctx, "FootprintBuilder: Invalid size of the indices of bound index buffer. IndexType: %v",
			execInfo.currentCmdBufState.indexType)
	}
	offset := uint64(0)
	size := vkWholeSize
	switch cmd := cmd.(type) {
	case *VkCmdDrawIndexed:
		size = uint64(cmd.IndexCount()) * indexSize
		offset += uint64(cmd.FirstIndex()) * indexSize
	case *VkCmdDrawIndexedIndirect:
	}
	dataToRead := execInfo.currentCmdBufState.indexBufferResBindings.getBoundData(
		ctx, bh, offset, size)
	read(ctx, bh, dataToRead...)
}

func (vb *FootprintBuilder) recordBarriers(ctx context.Context,
	s *api.GlobalState, ft *dependencygraph.Footprint, cmd api.Cmd,
	bh *dependencygraph.Behavior, vkCb VkCommandBuffer, memoryBarrierCount uint32,
	bufferBarrierCount uint32, pBufferBarriers VkBufferMemoryBarrierᶜᵖ,
	imageBarrierCount uint32, pImageBarriers VkImageMemoryBarrierᶜᵖ,
	attachedReads []dependencygraph.DefUseVariable) {
	l := s.MemoryLayout
	touchedData := []dependencygraph.DefUseVariable{}
	if memoryBarrierCount > 0 {
		// touch all buffer and image backing data
		for i := range vb.images {
			touchedData = append(touchedData, vb.getImageData(ctx, bh, i)...)
		}
		for b := range vb.buffers {
			touchedData = append(touchedData, vb.getBufferData(ctx, bh, b, 0, vkWholeSize)...)
		}
	} else {
		for _, barrier := range pBufferBarriers.Slice(0,
			uint64(bufferBarrierCount), l).MustRead(ctx, cmd, s, nil) {
			touchedData = append(touchedData, vb.getBufferData(ctx, bh, barrier.Buffer(),
				uint64(barrier.Offset()), uint64(barrier.Size()))...)
		}
		for _, barrier := range pImageBarriers.Slice(0,
			uint64(imageBarrierCount), l).MustRead(ctx, cmd, s, nil) {
			imgLayout, imgData := vb.getImageLayoutAndData(ctx, bh, barrier.Image())
			touchedData = append(touchedData, imgLayout)
			touchedData = append(touchedData, imgData...)
		}
	}
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand,
		execInfo *queueExecutionState) {
		for _, d := range touchedData {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, attachedReads...)
			modify(ctx, cbh, d)
			ft.AddBehavior(ctx, cbh)
		}
	}
}

// BuildFootprint incrementally builds the given Footprint with the given
// command specified with api.CmdID and api.Cmd.
func (vb *FootprintBuilder) BuildFootprint(ctx context.Context,
	s *api.GlobalState, ft *dependencygraph.Footprint, id api.CmdID, cmd api.Cmd) {

	l := s.MemoryLayout

	// Records the mapping from queue submit to command ID, so the
	// HandleSubcommand callback can use it.
	if qs, isSubmit := cmd.(*VkQueueSubmit); isSubmit {
		vb.submitIDs[qs] = id
	}
	// Register callback function to record only the truly executed
	// commandbuffer commands.
	executedCommands := []api.SubCmdIdx{}
	GetState(s).PostSubcommand = func(a interface{}) {
		queueSubmit, isQs := (GetState(s).CurrentSubmission).(*VkQueueSubmit)
		if !isQs {
			log.E(ctx, "FootprintBuilder: CurrentSubmission command in State is not a VkQueueSubmit")
		}
		fci := api.SubCmdIdx{uint64(vb.submitIDs[queueSubmit])}
		fci = append(fci, GetState(s).SubCmdIdx...)
		executedCommands = append(executedCommands, fci)
	}

	// Register callback function to track sparse bindings
	sparseBindingInfo := []QueuedSparseBinds{}
	GetState(s).postBindSparse = func(binds QueuedSparseBindsʳ) {
		sparseBindingInfo = append(sparseBindingInfo, binds.Get())
	}

	// Mutate
	if err := cmd.Mutate(ctx, id, s, nil); err != nil {
		// Continue the footprint building without emitting errors here. It is the
		// following mutate() calls' responsibility to catch the error.
		return
	}

	bh := dependencygraph.NewBehavior(
		api.SubCmdIdx{uint64(id)}, vb.machine)

	// Records the current last draw framebuffer image data, so that later when
	// the user request a command, we can always guarantee that the last draw
	// framebuffer is alive.
	if GetState(s).LastSubmission() == LastSubmissionType_SUBMIT {
		lastBoundQueue := GetState(s).LastBoundQueue()
		if !lastBoundQueue.IsNil() {
			if GetState(s).LastDrawInfos().Contains(lastBoundQueue.VulkanHandle()) {
				lastDrawInfo := GetState(s).LastDrawInfos().Get(lastBoundQueue.VulkanHandle())
				if !lastDrawInfo.Framebuffer().IsNil() {
					for _, view := range lastDrawInfo.Framebuffer().ImageAttachments().All() {
						if view.IsNil() || view.Image().IsNil() {
							continue
						}
						img := view.Image()
						data := vb.getImageData(ctx, nil, img.VulkanHandle())
						vb.machine.lastBoundFramebufferImageData[id] = append(
							vb.machine.lastBoundFramebufferImageData[id], data...)
					}
				}
			}
		}
	}

	// The main switch
	switch cmd := cmd.(type) {
	// device memory
	case *VkAllocateMemory:
		vkMem := cmd.PMemory().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkMem))
	case *VkFreeMemory:
		vkMem := cmd.Memory()
		read(ctx, bh, vkHandle(vkMem))
		bh.Alive = true
	case *VkMapMemory:
		modify(ctx, bh, vkHandle(cmd.Memory()))
		memObj := GetState(s).DeviceMemories().Get(cmd.Memory())
		isCoherent, _ := subIsMemoryCoherent(ctx, cmd, id, nil, s, GetState(s),
			cmd.Thread(), nil, memObj)
		if isCoherent {
			vb.mappedCoherentMemories[cmd.Memory()] = memObj
		}
		bh.Alive = true
	case *VkUnmapMemory:
		modify(ctx, bh, vkHandle(cmd.Memory()))
		vb.writeCoherentMemoryData(ctx, cmd, bh)
		if _, mappedCoherent := vb.mappedCoherentMemories[cmd.Memory()]; mappedCoherent {
			delete(vb.mappedCoherentMemories, cmd.Memory())
		}
	case *VkFlushMappedMemoryRanges:
		coherentMemDone := false
		count := uint64(cmd.MemoryRangeCount())
		for _, rng := range cmd.PMemoryRanges().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(rng.Memory()))
			mem := GetState(s).DeviceMemories().Get(rng.Memory())
			if mem.IsNil() {
				continue
			}
			isCoherent, _ := subIsMemoryCoherent(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, mem)
			if isCoherent {
				if !coherentMemDone {
					vb.writeCoherentMemoryData(ctx, cmd, bh)
					coherentMemDone = true
				}
				continue
			}
			offset := uint64(rng.Offset())
			size := uint64(rng.Size())
			ms := memorySpan{
				span:   interval.U64Span{Start: offset, End: offset + size},
				memory: rng.Memory(),
			}
			write(ctx, bh, ms)
		}
	case *VkInvalidateMappedMemoryRanges:
		count := uint64(cmd.MemoryRangeCount())
		for _, rng := range cmd.PMemoryRanges().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(rng.Memory()))
			offset := uint64(rng.Offset())
			size := uint64(rng.Size())
			ms := memorySpan{
				span:   interval.U64Span{Start: offset, End: offset + size},
				memory: rng.Memory(),
			}
			read(ctx, bh, ms)
		}

	// image
	case *VkCreateImage:
		vkImg := cmd.PImage().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkImg))
		vb.images[vkImg] = newImageLayoutAndData(ctx, bh)
	case *VkDestroyImage:
		vkImg := cmd.Image()
		if read(ctx, bh, vkHandle(vkImg)) {
			delete(vb.images, vkImg)
		}
		bh.Alive = true
	case *VkGetImageMemoryRequirements:
		// TODO: Once the memory requirements are moved out from the image object,
		// drop the 'modify' on the image handle, replace it with another proper
		// representation of the cached data.
		modify(ctx, bh, vkHandle(cmd.Image()))
	case *VkGetImageSparseMemoryRequirements:
		// TODO: Once the memory requirements are moved out from the image object,
		// drop the 'modify' on the image handle, replace it with another proper
		// representation of the cached data.
		modify(ctx, bh, vkHandle(cmd.Image()))

	case *VkBindImageMemory:
		read(ctx, bh, vkHandle(cmd.Image()))
		read(ctx, bh, vkHandle(cmd.Memory()))
		offset := uint64(cmd.MemoryOffset())
		inferredSize, err := subInferImageSize(ctx, cmd, id, nil, s, nil, cmd.Thread(),
			nil, GetState(s).Images().Get(cmd.Image()))
		if err != nil {
			log.E(ctx, "FootprintBuilder: Cannot get inferred size of image: %v", cmd.Image())
			log.E(ctx, "FootprintBuilder: Command %v %v: %v", id, cmd, err)
			bh.Aborted = true
		}
		size := uint64(inferredSize)
		vb.addOpaqueImageMemBinding(ctx, bh, cmd.Image(), cmd.Memory(), 0, size, offset)

	case *VkCreateImageView:
		write(ctx, bh, vkHandle(cmd.PView().MustRead(ctx, cmd, s, nil)))
		img := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil).Image()
		read(ctx, bh, vb.getImageData(ctx, bh, img)...)
	case *VkDestroyImageView:
		read(ctx, bh, vkHandle(cmd.ImageView()))
		bh.Alive = true

	// buffer
	case *VkCreateBuffer:
		vkBuf := cmd.PBuffer().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkBuf))
	case *VkDestroyBuffer:
		vkBuf := cmd.Buffer()
		if read(ctx, bh, vkHandle(vkBuf)) {
			delete(vb.buffers, vkBuf)
		}
		bh.Alive = true
	case *VkGetBufferMemoryRequirements:
		// TODO: Once the memory requirements are moved out from the buffer object,
		// drop the 'modify' on the buffer handle, replace it with another proper
		// representation of the cached data.
		modify(ctx, bh, vkHandle(cmd.Buffer()))

	case *VkBindBufferMemory:
		read(ctx, bh, vkHandle(cmd.Buffer()))
		read(ctx, bh, vkHandle(cmd.Memory()))
		offset := uint64(cmd.MemoryOffset())
		size := uint64(GetState(s).Buffers().Get(cmd.Buffer()).Info().Size())
		vb.addBufferMemBinding(ctx, bh, cmd.Buffer(), cmd.Memory(), 0, size, offset)
	case *VkCreateBufferView:
		write(ctx, bh, vkHandle(cmd.PView().MustRead(ctx, cmd, s, nil)))
	case *VkDestroyBufferView:
		read(ctx, bh, vkHandle(cmd.BufferView()))
		bh.Alive = true

	// swapchain
	case *VkCreateSwapchainKHR:
		vkSw := cmd.PSwapchain().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkSw))

	case *VkCreateSharedSwapchainsKHR:
		count := uint64(cmd.SwapchainCount())
		for _, vkSw := range cmd.PSwapchains().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			write(ctx, bh, vkHandle(vkSw))
		}

	case *VkGetSwapchainImagesKHR:
		read(ctx, bh, vkHandle(cmd.Swapchain()))
		if cmd.PSwapchainImages() == 0 {
			modify(ctx, bh, vkHandle(cmd.Swapchain()))
		} else {
			count := uint64(cmd.PSwapchainImageCount().MustRead(ctx, cmd, s, nil))
			for _, vkImg := range cmd.PSwapchainImages().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
				write(ctx, bh, vkHandle(vkImg))
				vb.images[vkImg] = newImageLayoutAndData(ctx, bh)
				vb.addSwapchainImageMemBinding(ctx, bh, vkImg)
				vb.swapchainImageAcquired[cmd.Swapchain()] = append(
					vb.swapchainImageAcquired[cmd.Swapchain()], newLabel())
				vb.swapchainImagePresented[cmd.Swapchain()] = append(
					vb.swapchainImagePresented[cmd.Swapchain()], newLabel())
			}
		}
	case *VkDestroySwapchainKHR:
		read(ctx, bh, vkHandle(cmd.Swapchain()))
		delete(vb.swapchainImageAcquired, cmd.Swapchain())
		delete(vb.swapchainImagePresented, cmd.Swapchain())
		bh.Alive = true

	// presentation engine
	case *VkAcquireNextImageKHR:
		if read(ctx, bh, vkHandle(cmd.Semaphore())) {
			write(ctx, bh, vb.semaphoreSignals[cmd.Semaphore()])
		}
		if read(ctx, bh, vkHandle(cmd.Fence())) {
			write(ctx, bh, vb.fences[cmd.Fence()].signal)
		}
		read(ctx, bh, vkHandle(cmd.Swapchain()))
		// The value of this imgId should have been written by the driver.
		imgID := cmd.PImageIndex().MustRead(ctx, cmd, s, nil)
		vkImg := GetState(s).Swapchains().Get(cmd.Swapchain()).SwapchainImages().Get(imgID).VulkanHandle()
		if read(ctx, bh, vkHandle(vkImg)) {
			imgLayout, imgData := vb.getImageLayoutAndData(ctx, bh, vkImg)
			write(ctx, bh, imgLayout)
			write(ctx, bh, imgData...)
		}
		write(ctx, bh, vb.swapchainImageAcquired[cmd.Swapchain()][imgID])
		read(ctx, bh, vb.swapchainImagePresented[cmd.Swapchain()][imgID])

	case *VkQueuePresentKHR:
		read(ctx, bh, vkHandle(cmd.Queue()))
		info := cmd.PPresentInfo().MustRead(ctx, cmd, s, nil)
		spCount := uint64(info.WaitSemaphoreCount())
		for _, vkSp := range info.PWaitSemaphores().Slice(0, spCount, l).MustRead(ctx, cmd, s, nil) {
			if read(ctx, bh, vkHandle(vkSp)) {
				read(ctx, bh, vb.semaphoreSignals[vkSp])
			}
		}
		swCount := uint64(info.SwapchainCount())
		imgIds := info.PImageIndices().Slice(0, swCount, l)
		for swi, vkSw := range info.PSwapchains().Slice(0, swCount, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(vkSw))
			imgID := imgIds.Index(uint64(swi)).MustRead(ctx, cmd, s, nil)[0]
			vkImg := GetState(s).Swapchains().Get(vkSw).SwapchainImages().Get(imgID).VulkanHandle()
			imgLayout, imgData := vb.getImageLayoutAndData(ctx, bh, vkImg)
			read(ctx, bh, imgLayout)
			read(ctx, bh, imgData...)

			// For each image to be presented, one extra behavior is requied to
			// track the acquire-present pair of the image state in the presentation
			// engine. And this extra behavior must be kept alive to prevent the
			// presentation engine from hang.
			extraBh := dependencygraph.NewBehavior(api.SubCmdIdx{uint64(id)}, vb.machine)
			for _, vkSp := range info.PWaitSemaphores().Slice(0, spCount, l).MustRead(ctx, cmd, s, nil) {
				read(ctx, extraBh, vkHandle(cmd.Queue()))
				if read(ctx, extraBh, vkHandle(vkSp)) {
					read(ctx, extraBh, vb.semaphoreSignals[vkSp])
				}
			}
			read(ctx, extraBh, vb.swapchainImageAcquired[vkSw][imgID])
			write(ctx, extraBh, vb.swapchainImagePresented[vkSw][imgID])
			extraBh.Alive = true
			ft.AddBehavior(ctx, extraBh)
		}

	// sampler
	case *VkCreateSampler:
		write(ctx, bh, vkHandle(cmd.PSampler().MustRead(ctx, cmd, s, nil)))
	case *VkDestroySampler:
		read(ctx, bh, vkHandle(cmd.Sampler()))
		bh.Alive = true

	// query pool
	case *VkCreateQueryPool:
		vkQp := cmd.PQueryPool().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkQp))
		vb.querypools[vkQp] = &queryPool{}
		count := uint64(cmd.PCreateInfo().MustRead(ctx, cmd, s, nil).QueryCount())
		for i := uint64(0); i < count; i++ {
			vb.querypools[vkQp].queries = append(vb.querypools[vkQp].queries, newQuery())
		}
	case *VkDestroyQueryPool:
		if read(ctx, bh, vkHandle(cmd.QueryPool())) {
			delete(vb.querypools, cmd.QueryPool())
		}
		bh.Alive = true
	case *VkGetQueryPoolResults:
		read(ctx, bh, vkHandle(cmd.QueryPool()))
		count := uint64(cmd.QueryCount())
		first := uint64(cmd.FirstQuery())
		for i := uint64(0); i < count; i++ {
			read(ctx, bh, vb.querypools[cmd.QueryPool()].queries[i+first].result)
		}

	// descriptor set
	case *VkCreateDescriptorSetLayout:
		write(ctx, bh, vkHandle(cmd.PSetLayout().MustRead(ctx, cmd, s, nil)))
		info := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
		bindings := info.PBindings().Slice(0, uint64(info.BindingCount()), l).MustRead(ctx, cmd, s, nil)
		for _, b := range bindings {
			if b.PImmutableSamplers() != memory.Nullptr {
				samplers := b.PImmutableSamplers().Slice(0, uint64(b.DescriptorCount()), l).MustRead(ctx, cmd, s, nil)
				for _, sam := range samplers {
					read(ctx, bh, vkHandle(sam))
				}
			}
		}
	case *VkDestroyDescriptorSetLayout:
		read(ctx, bh, vkHandle(cmd.DescriptorSetLayout()))
		bh.Alive = true
	case *VkAllocateDescriptorSets:
		info := cmd.PAllocateInfo().MustRead(ctx, cmd, s, nil)
		setCount := uint64(info.DescriptorSetCount())
		vkLayouts := info.PSetLayouts().Slice(0, setCount, l)
		for i, vkSet := range cmd.PDescriptorSets().Slice(0, setCount, l).MustRead(ctx, cmd, s, nil) {
			vkLayout := vkLayouts.Index(uint64(i)).MustRead(ctx, cmd, s, nil)[0]
			read(ctx, bh, vkHandle(vkLayout))
			layoutObj := GetState(s).DescriptorSetLayouts().Get(vkLayout)
			write(ctx, bh, vkHandle(vkSet))
			vb.descriptorSets[vkSet] = newDescriptorSet()
			for bi, bindingInfo := range layoutObj.Bindings().All() {
				for di := uint32(0); di < bindingInfo.Count(); di++ {
					vb.descriptorSets[vkSet].reserveDescriptor(uint64(bi), uint64(di))
				}
			}
		}
	case *VkUpdateDescriptorSets:
		writeCount := cmd.DescriptorWriteCount()
		if writeCount > 0 {
			for _, write := range cmd.PDescriptorWrites().Slice(0, uint64(writeCount),
				l).MustRead(ctx, cmd, s, nil) {
				read(ctx, bh, vkHandle(write.DstSet()))
				ds := vb.descriptorSets[write.DstSet()]
				ds.writeDescriptors(ctx, cmd, s, vb, bh, write)
			}
		}
		copyCount := cmd.DescriptorCopyCount()
		if copyCount > 0 {
			for _, copy := range cmd.PDescriptorCopies().Slice(0, uint64(copyCount),
				l).MustRead(ctx, cmd, s, nil) {
				read(ctx, bh, vkHandle(copy.SrcSet()))
				read(ctx, bh, vkHandle(copy.DstSet()))
				vb.descriptorSets[copy.DstSet()].copyDescriptors(ctx, cmd, s, bh,
					vb.descriptorSets[copy.SrcSet()], copy)
			}
		}

	case *VkFreeDescriptorSets:
		count := uint64(cmd.DescriptorSetCount())
		for _, vkSet := range cmd.PDescriptorSets().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(vkSet))
			delete(vb.descriptorSets, vkSet)
		}
		bh.Alive = true

	// pipelines
	case *VkCreatePipelineLayout:
		info := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(cmd.PPipelineLayout().MustRead(ctx, cmd, s, nil)))
		setCount := uint64(info.SetLayoutCount())
		for _, setLayout := range info.PSetLayouts().Slice(0, setCount, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(setLayout))
		}
	case *VkDestroyPipelineLayout:
		read(ctx, bh, vkHandle(cmd.PipelineLayout()))
		bh.Alive = true
	case *VkCreateGraphicsPipelines:
		read(ctx, bh, vkHandle(cmd.PipelineCache()))
		infoCount := uint64(cmd.CreateInfoCount())
		for _, info := range cmd.PCreateInfos().Slice(0, infoCount, l).MustRead(ctx, cmd, s, nil) {
			stageCount := uint64(info.StageCount())
			for _, stage := range info.PStages().Slice(0, stageCount, l).MustRead(ctx, cmd, s, nil) {
				module := stage.Module()
				read(ctx, bh, vkHandle(module))
			}
			read(ctx, bh, vkHandle(info.Layout()))
			read(ctx, bh, vkHandle(info.RenderPass()))
		}
		for _, vkPl := range cmd.PPipelines().Slice(0, infoCount, l).MustRead(ctx, cmd, s, nil) {
			write(ctx, bh, vkHandle(vkPl))
		}
	case *VkCreateComputePipelines:
		read(ctx, bh, vkHandle(cmd.PipelineCache()))
		infoCount := uint64(cmd.CreateInfoCount())
		for _, info := range cmd.PCreateInfos().Slice(0, infoCount, l).MustRead(ctx, cmd, s, nil) {
			stage := info.Stage()
			module := stage.Module()
			read(ctx, bh, vkHandle(module))
			read(ctx, bh, vkHandle(info.Layout()))
		}
		for _, vkPl := range cmd.PPipelines().Slice(0, infoCount, l).MustRead(ctx, cmd, s, nil) {
			write(ctx, bh, vkHandle(vkPl))
		}
	case *VkDestroyPipeline:
		read(ctx, bh, vkHandle(cmd.Pipeline()))
		bh.Alive = true

	case *VkCreatePipelineCache:
		write(ctx, bh, vkHandle(cmd.PPipelineCache().MustRead(ctx, cmd, s, nil)))
	case *VkDestroyPipelineCache:
		read(ctx, bh, vkHandle(cmd.PipelineCache()))
		bh.Alive = true
	case *VkGetPipelineCacheData:
		read(ctx, bh, vkHandle(cmd.PipelineCache()))
	case *VkMergePipelineCaches:
		modify(ctx, bh, vkHandle(cmd.DstCache()))
		srcCount := uint64(cmd.SrcCacheCount())
		for _, src := range cmd.PSrcCaches().Slice(0, srcCount, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(src))
		}

	// Shader module
	case *VkCreateShaderModule:
		write(ctx, bh, vkHandle(cmd.PShaderModule().MustRead(ctx, cmd, s, nil)))
	case *VkDestroyShaderModule:
		read(ctx, bh, vkHandle(cmd.ShaderModule()))
		bh.Alive = true

	// create/destroy renderpass
	case *VkCreateRenderPass:
		write(ctx, bh, vkHandle(cmd.PRenderPass().MustRead(ctx, cmd, s, nil)))
	case *VkDestroyRenderPass:
		read(ctx, bh, vkHandle(cmd.RenderPass()))
		bh.Alive = true

	// create/destroy framebuffer
	case *VkCreateFramebuffer:
		info := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
		read(ctx, bh, vkHandle(info.RenderPass()))
		attCount := uint64(info.AttachmentCount())
		for _, att := range info.PAttachments().Slice(0, attCount, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(att))
		}
		write(ctx, bh, vkHandle(cmd.PFramebuffer().MustRead(ctx, cmd, s, nil)))
	case *VkDestroyFramebuffer:
		read(ctx, bh, vkHandle(cmd.Framebuffer()))
		bh.Alive = true

	// debug marker name and tag setting commands. Always kept alive.
	case *VkDebugMarkerSetObjectTagEXT:
		read(ctx, bh, vkHandle(cmd.PTagInfo().MustRead(ctx, cmd, s, nil).Object()))
		bh.Alive = true
	case *VkDebugMarkerSetObjectNameEXT:
		read(ctx, bh, vkHandle(cmd.PNameInfo().MustRead(ctx, cmd, s, nil).Object()))
		bh.Alive = true

	// commandbuffer
	case *VkAllocateCommandBuffers:
		count := uint64(cmd.PAllocateInfo().MustRead(ctx, cmd, s, nil).CommandBufferCount())
		for _, vkCb := range cmd.PCommandBuffers().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			write(ctx, bh, vkHandle(vkCb))
			vb.commandBuffers[vkCb] = &commandBuffer{begin: newLabel(),
				end: newLabel(), renderPassBegin: newLabel()}
		}

	case *VkResetCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer()))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].begin)
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].end)
		vb.commands[cmd.CommandBuffer()] = []*commandBufferCommand{}

	case *VkFreeCommandBuffers:
		count := uint64(cmd.CommandBufferCount())
		for _, vkCb := range cmd.PCommandBuffers().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			if read(ctx, bh, vkHandle(vkCb)) {
				write(ctx, bh, vb.commandBuffers[vkCb].begin)
				write(ctx, bh, vb.commandBuffers[vkCb].end)
				delete(vb.commandBuffers, vkCb)
				delete(vb.commands, vkCb)
			}
		}
		bh.Alive = true

	case *VkBeginCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer()))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].begin)
		vb.commands[cmd.CommandBuffer()] = []*commandBufferCommand{}
	case *VkEndCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer()))
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].begin)
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].end)

	// copy, blit, resolve, clear, fill, update image and buffer
	case *VkCmdCopyImage:
		dst := vb.getImageData(ctx, bh, cmd.DstImage())
		src := vb.getImageData(ctx, bh, cmd.SrcImage())
		overwritten := false
		count := uint64(cmd.RegionCount())
		// TODO: check dst image coverage correctly
		for _, region := range cmd.PRegions().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images().Get(cmd.DstImage()),
				region.DstSubresource(), region.DstOffset(), region.Extent())
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, dst, emptyDefUseVars)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, emptyDefUseVars, dst)
		}

	case *VkCmdCopyBuffer:
		src := []dependencygraph.DefUseVariable{}
		dst := []dependencygraph.DefUseVariable{}
		count := uint64(cmd.RegionCount())
		for _, region := range cmd.PRegions().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			src = append(src, vb.getBufferData(ctx, bh, cmd.SrcBuffer(),
				uint64(region.SrcOffset()), uint64(region.Size()))...)
			dst = append(dst, vb.getBufferData(ctx, bh, cmd.DstBuffer(),
				uint64(region.DstOffset()), uint64(region.Size()))...)
		}
		vb.recordReadsWritesModifies(
			ctx, ft, bh, cmd.CommandBuffer(), src, dst, emptyDefUseVars)

	case *VkCmdCopyImageToBuffer:
		// TODO: calculate the ranges for the overwritten data
		dst := vb.getBufferData(ctx, bh, cmd.DstBuffer(), 0, vkWholeSize)
		src := vb.getImageData(ctx, bh, cmd.SrcImage())
		vb.recordReadsWritesModifies(
			ctx, ft, bh, cmd.CommandBuffer(), src, emptyDefUseVars, dst)

	case *VkCmdCopyBufferToImage:
		// TODO: calculate the ranges for the source data
		src := vb.getBufferData(ctx, bh, cmd.SrcBuffer(), 0, vkWholeSize)
		dst := vb.getImageData(ctx, bh, cmd.DstImage())
		overwritten := false
		count := uint64(cmd.RegionCount())
		for _, region := range cmd.PRegions().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images().Get(cmd.DstImage()),
				region.ImageSubresource(), region.ImageOffset(), region.ImageExtent())
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, dst, emptyDefUseVars)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, emptyDefUseVars, dst)
		}

	case *VkCmdBlitImage:
		src := vb.getImageData(ctx, bh, cmd.SrcImage())
		dst := vb.getImageData(ctx, bh, cmd.DstImage())
		overwritten := false
		count := uint64(cmd.RegionCount())
		for _, region := range cmd.PRegions().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			overwritten = overwritten || blitFullyCoverImage(
				GetState(s).Images().Get(cmd.DstImage()),
				region.DstSubresource(),
				region.DstOffsets().Get(0), region.DstOffsets().Get(1))
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, dst, emptyDefUseVars)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, emptyDefUseVars, dst)
		}

	case *VkCmdResolveImage:
		src := vb.getImageData(ctx, bh, cmd.SrcImage())
		dst := vb.getImageData(ctx, bh, cmd.DstImage())
		overwritten := false
		count := uint64(cmd.RegionCount())
		for _, region := range cmd.PRegions().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images().Get(cmd.DstImage()),
				region.DstSubresource(), region.DstOffset(), region.Extent())
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, dst, emptyDefUseVars)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer(), src, emptyDefUseVars, dst)
		}

	case *VkCmdFillBuffer:
		dst := vb.getBufferData(ctx, bh, cmd.DstBuffer(), uint64(cmd.DstOffset()), uint64(cmd.Size()))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(),
			emptyDefUseVars, dst, emptyDefUseVars)

	case *VkCmdUpdateBuffer:
		dst := vb.getBufferData(ctx, bh, cmd.DstBuffer(), uint64(cmd.DstOffset()), uint64(cmd.DataSize()))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(),
			emptyDefUseVars, dst, emptyDefUseVars)

	case *VkCmdClearColorImage:
		dst := vb.getImageData(ctx, bh, cmd.Image())
		count := uint64(cmd.RangeCount())
		overwritten := false
		for _, rng := range cmd.PRanges().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			if subresourceRangeFullyCoverImage(GetState(s).Images().Get(cmd.Image()), rng) {
				overwritten = true
			}
		}
		if overwritten {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(),
				emptyDefUseVars, dst, emptyDefUseVars)
		} else {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(),
				emptyDefUseVars, emptyDefUseVars, dst)
		}

	case *VkCmdClearDepthStencilImage:
		dst := vb.getImageData(ctx, bh, cmd.Image())
		count := uint64(cmd.RangeCount())
		overwritten := false
		for _, rng := range cmd.PRanges().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			if subresourceRangeFullyCoverImage(GetState(s).Images().Get(cmd.Image()), rng) {
				overwritten = true
			}
		}
		if overwritten {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(),
				emptyDefUseVars, dst, emptyDefUseVars)
		} else {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(),
				emptyDefUseVars, emptyDefUseVars, dst)
		}

	// renderpass and subpass
	case *VkCmdBeginRenderPass:
		vkRp := cmd.PRenderPassBegin().MustRead(ctx, cmd, s, nil).RenderPass()
		read(ctx, bh, vkHandle(vkRp))
		vkFb := cmd.PRenderPassBegin().MustRead(ctx, cmd, s, nil).Framebuffer()
		read(ctx, bh, vkHandle(vkFb))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].renderPassBegin)
		rp := GetState(s).RenderPasses().Get(vkRp)
		fb := GetState(s).Framebuffers().Get(vkFb)
		read(ctx, bh, vkHandle(fb.RenderPass().VulkanHandle()))
		for _, ia := range fb.ImageAttachments().All() {
			if read(ctx, bh, vkHandle(ia.VulkanHandle())) {
				read(ctx, bh, vkHandle(ia.Image().VulkanHandle()))
			}
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			execInfo.beginRenderPass(ctx, vb, cbh, rp, fb)
			execInfo.renderPassBegin = newForwardPairedLabel(ctx, cbh)
			ft.AddBehavior(ctx, cbh)
			cbh.Alive = true // TODO(awoloszyn)(BUG:1158): Investigate why this is needed.
			// Without this, we drop some needed commands.
		}

	case *VkCmdNextSubpass:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			execInfo.nextSubpass(ctx, ft, cbh, sc, vb.machine)
			ft.AddBehavior(ctx, cbh)
			cbh.Alive = true // TODO(awoloszyn)(BUG:1158): Investigate why this is needed.
			// Without this, we drop some needed commands.
		}

	case *VkCmdEndRenderPass:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			execInfo.endRenderPass(ctx, ft, cbh, sc, vb.machine)
			read(ctx, cbh, execInfo.renderPassBegin)
			ft.AddBehavior(ctx, cbh)
			cbh.Alive = true // TODO(awoloszyn)(BUG:1158): Investigate why this is needed.
			// Without this, we drop some needed commands.
		}

	// bind vertex buffers, index buffer, pipeline and descriptors
	case *VkCmdBindVertexBuffers:
		count := uint64(cmd.BindingCount())
		offsets := cmd.POffsets().Slice(0, count, l).MustRead(ctx, cmd, s, nil)
		subBindings := []resBindingList{}
		for i, vkBuf := range cmd.PBuffers().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			subBindings = append(subBindings, vb.buffers[vkBuf].getSubBindingList(ctx, bh,
				uint64(offsets[i]), vkWholeSize))
		}
		firstBinding := cmd.FirstBinding()
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			for i, sb := range subBindings {
				binding := firstBinding + uint32(i)
				execInfo.currentCmdBufState.vertexBufferResBindings[binding] = sb
			}
			ft.AddBehavior(ctx, cbh)
		}
	case *VkCmdBindIndexBuffer:
		subBindings := vb.buffers[cmd.Buffer()].getSubBindingList(ctx, bh,
			uint64(cmd.Offset()), vkWholeSize)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			execInfo.currentCmdBufState.indexBufferResBindings = subBindings
			execInfo.currentCmdBufState.indexType = cmd.IndexType()
			ft.AddBehavior(ctx, cbh)
		}
	case *VkCmdBindPipeline:
		vkPi := cmd.Pipeline()
		read(ctx, bh, vkHandle(vkPi))
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, vkHandle(vkPi))
			write(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			ft.AddBehavior(ctx, cbh)
		}
	case *VkCmdBindDescriptorSets:
		read(ctx, bh, vkHandle(cmd.Layout()))
		count := uint64(cmd.DescriptorSetCount())
		dss := []*descriptorSet{}
		for _, vkSet := range cmd.PDescriptorSets().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(vkSet))
			dss = append(dss, vb.descriptorSets[vkSet])
		}
		firstSet := cmd.FirstSet()
		dOffsets := []uint32{}
		dOffsetsLeft := 0
		if cmd.DynamicOffsetCount() > uint32(0) {
			dOffsets = cmd.PDynamicOffsets().Slice(0, uint64(cmd.DynamicOffsetCount()),
				l).MustRead(ctx, cmd, s, nil)
			dOffsetsLeft = len(dOffsets)
		}
		getDynamicOffset := func() uint32 {
			if dOffsetsLeft > 0 {
				d := dOffsets[len(dOffsets)-dOffsetsLeft]
				dOffsetsLeft--
				return d
			}
			log.E(ctx, "FootprintBuilder: The number of dynamic offsets does not match with the number of dynamic descriptors")
			return 0
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			for i, ds := range dss {
				set := firstSet + uint32(i)
				execInfo.currentCmdBufState.descriptorSets[set] = newBoundDescriptorSet(ctx, cbh, ds, getDynamicOffset)
			}
			ft.AddBehavior(ctx, cbh)
		}

	// draw and dispatch
	case *VkCmdDraw:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			vb.draw(ctx, cbh, execInfo)
			ft.AddBehavior(ctx, cbh)
		}

	case *VkCmdDrawIndexed:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			vb.readBoundIndexBuffer(ctx, cbh, execInfo, cmd)
			vb.draw(ctx, cbh, execInfo)
			ft.AddBehavior(ctx, cbh)
		}

	case *VkCmdDrawIndirect:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].renderPassBegin)
		count := uint64(cmd.DrawCount())
		sizeOfDrawIndirectdCommand := uint64(4 * 4)
		offset := uint64(cmd.Offset())
		src := []dependencygraph.DefUseVariable{}
		for i := uint64(0); i < count; i++ {
			src = append(src, vb.getBufferData(ctx, bh, cmd.Buffer(), offset,
				sizeOfDrawIndirectdCommand)...)
			offset += uint64(cmd.Stride())
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			vb.draw(ctx, cbh, execInfo)
			read(ctx, cbh, src...)
			ft.AddBehavior(ctx, cbh)
		}

	case *VkCmdDrawIndexedIndirect:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer()].renderPassBegin)
		count := uint64(cmd.DrawCount())
		sizeOfDrawIndexedIndirectCommand := uint64(5 * 4)
		offset := uint64(cmd.Offset())
		src := []dependencygraph.DefUseVariable{}
		for i := uint64(0); i < count; i++ {
			src = append(src, vb.getBufferData(ctx, bh, cmd.Buffer(), offset,
				sizeOfDrawIndexedIndirectCommand)...)
			offset += uint64(cmd.Stride())
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			vb.readBoundIndexBuffer(ctx, cbh, execInfo, cmd)
			vb.draw(ctx, cbh, execInfo)
			read(ctx, cbh, src...)
			ft.AddBehavior(ctx, cbh)
		}

	case *VkCmdDispatch:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			modified := vb.useBoundDescriptorSets(ctx, cbh, execInfo.currentCmdBufState)
			modify(ctx, cbh, modified...)
			ft.AddBehavior(ctx, cbh)
		}

	case *VkCmdDispatchIndirect:
		sizeOfDispatchIndirectCommand := uint64(3 * 4)
		src := vb.getBufferData(ctx, bh, cmd.Buffer(), uint64(cmd.Offset()), sizeOfDispatchIndirectCommand)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			modified := vb.useBoundDescriptorSets(ctx, cbh, execInfo.currentCmdBufState)
			modify(ctx, cbh, modified...)
			read(ctx, cbh, src...)
			ft.AddBehavior(ctx, cbh)
		}

	// pipeline settings
	case *VkCmdPushConstants:
		read(ctx, bh, vkHandle(cmd.Layout()))
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetLineWidth:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetScissor:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetViewport:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetDepthBias:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetDepthBounds:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetBlendConstants:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetStencilCompareMask:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetStencilWriteMask:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdSetStencilReference:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer())

	// clear attachments
	case *VkCmdClearAttachments:
		attCount := uint64(cmd.AttachmentCount())
		atts := []VkClearAttachment{}
		rectCount := uint64(cmd.RectCount())
		rects := []VkClearRect{}
		for _, att := range cmd.PAttachments().Slice(0, attCount, l).MustRead(ctx, cmd, s, nil) {
			atts = append(atts, att)
		}
		for _, rect := range cmd.PRects().Slice(0, rectCount, l).MustRead(ctx, cmd, s, nil) {
			rects = append(rects, rect)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionState) {
			cbh := sc.cmd.newBehavior(ctx, sc, vb.machine, execInfo)
			for _, a := range atts {
				clearAttachmentData(ctx, cbh, execInfo, a, rects)
			}
			ft.AddBehavior(ctx, cbh)
		}

	// query pool commands
	case *VkCmdResetQueryPool:
		read(ctx, bh, vkHandle(cmd.QueryPool()))
		resetLabels := []dependencygraph.DefUseVariable{}
		count := uint64(cmd.QueryCount())
		first := uint64(cmd.FirstQuery())
		for i := uint64(0); i < count; i++ {
			resetLabels = append(resetLabels,
				vb.querypools[cmd.QueryPool()].queries[first+i].reset)
		}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), emptyDefUseVars,
			resetLabels, emptyDefUseVars)
	case *VkCmdBeginQuery:
		read(ctx, bh, vkHandle(cmd.QueryPool()))
		resetLabels := []dependencygraph.DefUseVariable{
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].reset}
		beginLabels := []dependencygraph.DefUseVariable{
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].begin}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), resetLabels,
			beginLabels, emptyDefUseVars)
	case *VkCmdEndQuery:
		read(ctx, bh, vkHandle(cmd.QueryPool()))
		endAndResultLabels := []dependencygraph.DefUseVariable{
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].end,
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].result,
		}
		beginLabels := []dependencygraph.DefUseVariable{
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].begin}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), beginLabels,
			endAndResultLabels, emptyDefUseVars)
	case *VkCmdWriteTimestamp:
		read(ctx, bh, vkHandle(cmd.QueryPool()))
		resetLabels := []dependencygraph.DefUseVariable{
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].reset}
		resultLabels := []dependencygraph.DefUseVariable{
			vb.querypools[cmd.QueryPool()].queries[cmd.Query()].result}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), resetLabels,
			resultLabels, emptyDefUseVars)
	case *VkCmdCopyQueryPoolResults:
		read(ctx, bh, vkHandle(cmd.QueryPool()))
		// TODO: calculate the range
		src := []dependencygraph.DefUseVariable{}
		dst := vb.getBufferData(ctx, bh, cmd.DstBuffer(), 0, vkWholeSize)
		count := uint64(cmd.QueryCount())
		first := uint64(cmd.FirstQuery())
		for i := uint64(0); i < count; i++ {
			src = append(src, vb.querypools[cmd.QueryPool()].queries[first+i].result)
		}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), src, emptyDefUseVars, dst)

	// debug marker extension commandbuffer commands. Those commands are kept
	// alive if they are submitted.
	case *VkCmdDebugMarkerBeginEXT:
		vb.keepSubmittedCommandAlive(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdDebugMarkerEndEXT:
		vb.keepSubmittedCommandAlive(ctx, ft, bh, cmd.CommandBuffer())
	case *VkCmdDebugMarkerInsertEXT:
		vb.keepSubmittedCommandAlive(ctx, ft, bh, cmd.CommandBuffer())

	// event commandbuffer commands
	case *VkCmdSetEvent:
		read(ctx, bh, vkHandle(cmd.Event()))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), emptyDefUseVars,
			[]dependencygraph.DefUseVariable{vb.events[cmd.Event()].signal}, emptyDefUseVars)
	case *VkCmdResetEvent:
		read(ctx, bh, vkHandle(cmd.Event()))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer(), emptyDefUseVars,
			[]dependencygraph.DefUseVariable{vb.events[cmd.Event()].unsignal}, emptyDefUseVars)
	case *VkCmdWaitEvents:
		eventLabels := []dependencygraph.DefUseVariable{}
		evCount := uint64(cmd.EventCount())
		for _, vkEv := range cmd.PEvents().Slice(0, evCount, l).MustRead(ctx, cmd, s, nil) {
			read(ctx, bh, vkHandle(vkEv))
			eventLabels = append(eventLabels, vb.events[vkEv].signal,
				vb.events[vkEv].unsignal)
		}
		vb.recordBarriers(ctx, s, ft, cmd, bh, cmd.CommandBuffer(), cmd.MemoryBarrierCount(),
			cmd.BufferMemoryBarrierCount(), cmd.PBufferMemoryBarriers(),
			cmd.ImageMemoryBarrierCount(), cmd.PImageMemoryBarriers(), eventLabels)

	// pipeline barrier
	case *VkCmdPipelineBarrier:
		vb.recordBarriers(ctx, s, ft, cmd, bh, cmd.CommandBuffer(), cmd.MemoryBarrierCount(),
			cmd.BufferMemoryBarrierCount(), cmd.PBufferMemoryBarriers(),
			cmd.ImageMemoryBarrierCount(), cmd.PImageMemoryBarriers(), emptyDefUseVars)

	// secondary command buffers
	case *VkCmdExecuteCommands:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer())
		cbc.isCmdExecuteCommands = true
		count := uint64(cmd.CommandBufferCount())
		for _, vkScb := range cmd.PCommandBuffers().Slice(0, count, l).MustRead(ctx, cmd, s, nil) {
			cbc.recordSecondaryCommandBuffer(vkScb)
			read(ctx, bh, vkHandle(vkScb))
		}
		cbc.behave = func(sc submittedCommand, execInfo *queueExecutionState) {}

	// execution triggering
	case *VkQueueSubmit:
		read(ctx, bh, vkHandle(cmd.Queue()))
		if _, ok := vb.executionStates[cmd.Queue()]; !ok {
			vb.executionStates[cmd.Queue()] = newQueueExecutionState(id)
		}
		vb.executionStates[cmd.Queue()].lastSubmitID = id
		// collect submission info and submitted commands
		vb.submitInfos[id] = &queueSubmitInfo{
			began:  false,
			queued: newLabel(),
			done:   newLabel(),
			queue:  cmd.Queue(),
		}
		submitCount := uint64(cmd.SubmitCount())
		hasCmd := false
		for i, submit := range cmd.PSubmits().Slice(0, submitCount, l).MustRead(ctx, cmd, s, nil) {
			commandBufferCount := uint64(submit.CommandBufferCount())
			for j, vkCb := range submit.PCommandBuffers().Slice(0, commandBufferCount, l).MustRead(ctx, cmd, s, nil) {
				read(ctx, bh, vkHandle(vkCb))
				read(ctx, bh, vb.commandBuffers[vkCb].end)
				for k, cbc := range vb.commands[vkCb] {
					if !hasCmd {
						hasCmd = true
					}
					fci := api.SubCmdIdx{uint64(id), uint64(i), uint64(j), uint64(k)}
					submittedCmd := newSubmittedCommand(fci, cbc, nil)
					vb.submitInfos[id].pendingCommands = append(vb.submitInfos[id].pendingCommands, submittedCmd)
					if cbc.isCmdExecuteCommands {
						for scbi, scb := range cbc.secondaryCommandBuffers {
							read(ctx, bh, vb.commandBuffers[scb].end)
							for sci, scbc := range vb.commands[scb] {
								fci := api.SubCmdIdx{uint64(id), uint64(i), uint64(j), uint64(k), uint64(scbi), uint64(sci)}
								submittedCmd := newSubmittedCommand(fci, scbc, cbc)
								vb.submitInfos[id].pendingCommands = append(vb.submitInfos[id].pendingCommands, submittedCmd)
							}
						}
					}
				}
			}
			waitSemaphoreCount := uint64(submit.WaitSemaphoreCount())
			for _, vkSp := range submit.PWaitSemaphores().Slice(0, waitSemaphoreCount, l).MustRead(ctx, cmd, s, nil) {
				vb.submitInfos[id].waitSemaphores = append(
					vb.submitInfos[id].waitSemaphores, vkSp)
			}
			signalSemaphoreCount := uint64(submit.SignalSemaphoreCount())
			for _, vkSp := range submit.PSignalSemaphores().Slice(0, signalSemaphoreCount, l).MustRead(ctx, cmd, s, nil) {
				vb.submitInfos[id].signalSemaphores = append(
					vb.submitInfos[id].signalSemaphores, vkSp)
			}
		}
		vb.submitInfos[id].signalFence = cmd.Fence()

		// queue execution begin
		vb.writeCoherentMemoryData(ctx, cmd, bh)
		if read(ctx, bh, vkHandle(cmd.Fence())) {
			read(ctx, bh, vb.fences[cmd.Fence()].unsignal)
			write(ctx, bh, vb.fences[cmd.Fence()].signal)
		}
		// If the submission does not contains commands, records the write
		// behavior here as we don't have any callbacks for those operations.
		// This is not exactly correct. If the whole submission is in pending
		// state, even if there is no command to submit, those semaphore/fence
		// signal/unsignal operations will be in pending, instead of being
		// carried out immediately.
		// TODO: Once we merge the dependency tree building process to mutate
		// calls, make sure the signal/unsignal operations in pending state
		// are handled correctly.
		write(ctx, bh, vb.submitInfos[id].queued)
		for _, sp := range vb.submitInfos[id].waitSemaphores {
			if read(ctx, bh, vkHandle(sp)) {
				if !hasCmd {
					modify(ctx, bh, vb.semaphoreSignals[sp])
				}
			}
		}
		for _, sp := range vb.submitInfos[id].signalSemaphores {
			if read(ctx, bh, vkHandle(sp)) {
				if !hasCmd {
					write(ctx, bh, vkHandle(sp))
				}
			}
		}
		if read(ctx, bh, vkHandle(cmd.Fence())) {
			if !hasCmd {
				write(ctx, bh, vb.fences[cmd.Fence()].signal)
			}
		}

	case *VkSetEvent:
		if read(ctx, bh, vkHandle(cmd.Event())) {
			write(ctx, bh, vb.events[cmd.Event()].signal)
			vb.writeCoherentMemoryData(ctx, cmd, bh)
			bh.Alive = true
		}

	case *VkQueueBindSparse:
		read(ctx, bh, vkHandle(cmd.Queue()))
		for _, bindInfo := range cmd.PBindInfo().Slice(0, uint64(cmd.BindInfoCount()), l).MustRead(
			ctx, cmd, s, nil) {
			for _, bufferBinds := range bindInfo.PBufferBinds().Slice(0,
				uint64(bindInfo.BufferBindCount()), l).MustRead(ctx, cmd, s, nil) {
				if read(ctx, bh, vkHandle(bufferBinds.Buffer())) {
					buf := bufferBinds.Buffer()
					binds := bufferBinds.PBinds().Slice(0, uint64(bufferBinds.BindCount()), l).MustRead(
						ctx, cmd, s, nil)
					for _, bind := range binds {
						if read(ctx, bh, vkHandle(bind.Memory())) {
							vb.addBufferMemBinding(ctx, bh, buf, bind.Memory(),
								uint64(bind.ResourceOffset()), uint64(bind.Size()), uint64(bind.MemoryOffset()))
						}
					}
				}
			}
			for _, opaqueBinds := range bindInfo.PImageOpaqueBinds().Slice(0,
				uint64(bindInfo.ImageOpaqueBindCount()), l).MustRead(ctx, cmd, s, nil) {
				if read(ctx, bh, vkHandle(opaqueBinds.Image())) {
					img := opaqueBinds.Image()
					binds := opaqueBinds.PBinds().Slice(0, uint64(opaqueBinds.BindCount()), l).MustRead(
						ctx, cmd, s, nil)
					for _, bind := range binds {
						if read(ctx, bh, vkHandle(bind.Memory())) {
							vb.addOpaqueImageMemBinding(ctx, bh, img, bind.Memory(),
								uint64(bind.ResourceOffset()), uint64(bind.Size()), uint64(bind.MemoryOffset()))
						}
					}
				}
			}
			for _, imageBinds := range bindInfo.PImageBinds().Slice(0,
				uint64(bindInfo.ImageBindCount()), l).MustRead(ctx, cmd, s, nil) {
				if read(ctx, bh, vkHandle(imageBinds.Image())) {
					img := imageBinds.Image()
					binds := imageBinds.PBinds().Slice(0, uint64(imageBinds.BindCount()), l).MustRead(
						ctx, cmd, s, nil)
					for _, bind := range binds {
						if read(ctx, bh, vkHandle(bind.Memory())) {
							vb.addSparseImageMemBinding(ctx, cmd, id, s, bh, img, bind)
						}
					}
				}
			}
		}

	// synchronization primitives
	case *VkResetEvent:
		if read(ctx, bh, vkHandle(cmd.Event())) {
			write(ctx, bh, vb.events[cmd.Event()].unsignal)
			bh.Alive = true
		}

	case *VkCreateSemaphore:
		vkSp := cmd.PSemaphore().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkSp))
		vb.semaphoreSignals[vkSp] = newLabel()
	case *VkDestroySemaphore:
		vkSp := cmd.Semaphore()
		if read(ctx, bh, vkHandle(vkSp)) {
			delete(vb.semaphoreSignals, vkSp)
			bh.Alive = true
		}

	case *VkCreateEvent:
		vkEv := cmd.PEvent().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkEv))
		vb.events[vkEv] = &event{signal: newLabel(), unsignal: newLabel()}
	case *VkGetEventStatus:
		vkEv := cmd.Event()
		if read(ctx, bh, vkHandle(vkEv)) {
			read(ctx, bh, vb.events[vkEv].signal)
			read(ctx, bh, vb.events[vkEv].unsignal)
			bh.Alive = true
		}
	case *VkDestroyEvent:
		vkEv := cmd.Event()
		if read(ctx, bh, vkHandle(vkEv)) {
			delete(vb.events, vkEv)
			bh.Alive = true
		}

	case *VkCreateFence:
		vkFe := cmd.PFence().MustRead(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkFe))
		vb.fences[vkFe] = &fence{signal: newLabel(), unsignal: newLabel()}
	case *VkGetFenceStatus:
		vkFe := cmd.Fence()
		if read(ctx, bh, vkHandle(vkFe)) {
			read(ctx, bh, vb.fences[vkFe].signal)
			read(ctx, bh, vb.fences[vkFe].unsignal)
			bh.Alive = true
		}
	case *VkWaitForFences:
		fenceCount := uint64(cmd.FenceCount())
		for _, vkFe := range cmd.PFences().Slice(0, fenceCount, l).MustRead(ctx, cmd, s, nil) {
			if read(ctx, bh, vkHandle(vkFe)) {
				read(ctx, bh, vb.fences[vkFe].signal)
				read(ctx, bh, vb.fences[vkFe].unsignal)
				bh.Alive = true
			}
		}
	case *VkResetFences:
		fenceCount := uint64(cmd.FenceCount())
		for _, vkFe := range cmd.PFences().Slice(0, fenceCount, l).MustRead(ctx, cmd, s, nil) {
			if read(ctx, bh, vkHandle(vkFe)) {
				write(ctx, bh, vb.fences[vkFe].unsignal)
				bh.Alive = true
			}
		}
	case *VkDestroyFence:
		vkFe := cmd.Fence()
		if read(ctx, bh, vkHandle(vkFe)) {
			delete(vb.fences, vkFe)
			bh.Alive = true
		}

	case *VkQueueWaitIdle:
		vkQu := cmd.Queue()
		if read(ctx, bh, vkHandle(vkQu)) {
			if _, ok := vb.executionStates[vkQu]; ok {
				bh.Alive = true
			}
		}

	case *VkDeviceWaitIdle:
		for _, qei := range vb.executionStates {
			lastSubmitInfo := vb.submitInfos[qei.lastSubmitID]
			read(ctx, bh, lastSubmitInfo.done)
			bh.Alive = true
		}

	// Property queries, can be dropped if they are not the requested command.
	case *VkGetDeviceMemoryCommitment:
		read(ctx, bh, vkHandle(cmd.Memory()))
	case *VkGetImageSubresourceLayout:
		read(ctx, bh, vkHandle(cmd.Image()))
	case *VkGetRenderAreaGranularity:
		read(ctx, bh, vkHandle(cmd.RenderPass()))
	case *VkEnumerateInstanceExtensionProperties,
		*VkEnumerateDeviceExtensionProperties,
		*VkEnumerateInstanceLayerProperties,
		*VkEnumerateDeviceLayerProperties:

	// Keep alive
	case *VkGetDeviceProcAddr,
		*VkGetInstanceProcAddr:
		bh.Alive = true
	case *VkCreateInstance:
		bh.Alive = true
	case *VkEnumeratePhysicalDevices:
		bh.Alive = true
	case *VkCreateDevice:
		bh.Alive = true
	case *VkGetDeviceQueue:
		bh.Alive = true
	case *VkCreateDescriptorPool,
		*VkDestroyDescriptorPool,
		*VkResetDescriptorPool:
		bh.Alive = true
	case *VkCreateAndroidSurfaceKHR,
		*VkCreateXlibSurfaceKHR,
		*VkCreateXcbSurfaceKHR,
		*VkCreateWaylandSurfaceKHR,
		*VkCreateMirSurfaceKHR,
		*VkCreateWin32SurfaceKHR,
		*VkDestroySurfaceKHR:
		bh.Alive = true
	case *VkCreateCommandPool,
		// TODO: ResetCommandPool should overwrite all the command buffers in this
		// pool.
		*VkResetCommandPool,
		*VkDestroyCommandPool:
		bh.Alive = true
	case *VkGetPhysicalDeviceXlibPresentationSupportKHR,
		*VkGetPhysicalDeviceXcbPresentationSupportKHR,
		*VkGetPhysicalDeviceWaylandPresentationSupportKHR,
		*VkGetPhysicalDeviceWin32PresentationSupportKHR,
		*VkGetPhysicalDeviceMirPresentationSupportKHR:
		bh.Alive = true
	case *VkGetPhysicalDeviceProperties,
		*VkGetPhysicalDeviceMemoryProperties,
		*VkGetPhysicalDeviceQueueFamilyProperties,
		*VkGetPhysicalDeviceDisplayPropertiesKHR,
		*VkGetPhysicalDeviceDisplayPlanePropertiesKHR,
		*VkGetPhysicalDeviceFeatures,
		*VkGetPhysicalDeviceFormatProperties,
		*VkGetPhysicalDeviceImageFormatProperties,
		*VkGetPhysicalDeviceSparseImageFormatProperties:
		bh.Alive = true
	case *VkGetPhysicalDeviceSurfaceSupportKHR,
		*VkGetPhysicalDeviceSurfaceCapabilitiesKHR,
		*VkGetPhysicalDeviceSurfaceFormatsKHR,
		*VkGetPhysicalDeviceSurfacePresentModesKHR:
		bh.Alive = true
	case *VkGetDisplayPlaneSupportedDisplaysKHR,
		*VkGetDisplayModePropertiesKHR,
		*VkGetDisplayPlaneCapabilitiesKHR,
		*VkCreateDisplayPlaneSurfaceKHR,
		*VkCreateDisplayModeKHR:
		bh.Alive = true
	// Unhandled, always keep alive
	default:
		log.W(ctx, "Command: %v is not handled in FootprintBuilder", cmd)
		bh.Alive = true
	}

	ft.AddBehavior(ctx, bh)

	// roll out the recorded reads and writes for queue submit and set event
	switch cmd.(type) {
	case *VkQueueSubmit:
		vb.rollOutExecuted(ctx, ft, executedCommands)
	case *VkSetEvent:
		vb.rollOutExecuted(ctx, ft, executedCommands)
	}
}

func (vb *FootprintBuilder) writeCoherentMemoryData(ctx context.Context,
	cmd api.Cmd, bh *dependencygraph.Behavior) {
	if cmd.Extras() == nil || cmd.Extras().Observations() == nil {
		return
	}
	for _, ro := range cmd.Extras().Observations().Reads {
		// Here we intersect all the memory observations with all the mapped
		// coherent memories. If any intersects are found, mark the behavior
		// as alive (explained in the loop below).
		// Another more intuitive way is to cache the observation here then, pull
		// the data later when rolling out the submitted commands, this way we only
		// record 'write' operation for the observations that are actually used in
		// the submitted commands. But it actually does not help, because without
		// the permit to modify api.Cmd, the coherent memory observations can only
		// be 'alive' or 'dead' altogether. Postponing the recording of 'write'
		// operation does not save any data.
		for vkDm, mm := range vb.mappedCoherentMemories {
			mappedRng := memory.Range{
				Base: uint64(mm.MappedLocation().Address()),
				Size: uint64(mm.MappedSize()),
			}
			if ro.Range.Overlaps(mappedRng) {

				// Dirty hack. If there are coherent memory observation attached on
				// this vkQueueSubmit, we need to keep, even if all the commands in
				// this submission are useless. This is because the observed pages
				// might be shared with other following commands in future queue
				// submissions. As we are not going to modify api.Cmd here to pass the
				// observations, we need those observation being called with
				// ApplyReads(). So we need to keep such vkQueueSubmit. vkUnmapMemory
				// has the same issue.
				bh.Alive = true

				intersect := ro.Range.Intersect(mappedRng)
				offset := uint64(mm.MappedOffset()) + intersect.Base - mm.MappedLocation().Address()
				ms := memorySpan{
					span:   interval.U64Span{Start: offset, End: offset + intersect.Size},
					memory: vkDm,
				}
				write(ctx, bh, ms)
			}
		}
	}
}

// helper functions
func debug(ctx context.Context, fmt string, args ...interface{}) {
	if config.DebugDeadCodeElimination {
		log.D(ctx, fmt, args...)
	}
}

func read(ctx context.Context, bh *dependencygraph.Behavior,
	cs ...dependencygraph.DefUseVariable) bool {
	allSucceeded := true
	for _, c := range cs {
		switch c := c.(type) {
		case vkHandle:
			if c == vkNullHandle {
				debug(ctx, "Read to VK_NULL_HANDLE is ignored")
				allSucceeded = false
				continue
			}
		case *forwardPairedLabel:
			c.labelReadBehaviors = append(c.labelReadBehaviors, bh)
		}
		bh.Read(c)
		debug(ctx, "<Behavior: %v, Read: %v>", bh, c)
	}
	return allSucceeded
}

func write(ctx context.Context, bh *dependencygraph.Behavior,
	cs ...dependencygraph.DefUseVariable) bool {
	allSucceeded := true
	for _, c := range cs {
		switch c := c.(type) {
		case vkHandle:
			if c == vkNullHandle {
				debug(ctx, "Write to VK_NULL_HANDLE is ignored")
				allSucceeded = false
				continue
			}
		}
		bh.Write(c)
		debug(ctx, "<Behavior: %v, Write: %v>", bh, c)
	}
	return allSucceeded
}

func modify(ctx context.Context, bh *dependencygraph.Behavior,
	cs ...dependencygraph.DefUseVariable) bool {
	allSucceeded := true
	for _, c := range cs {
		switch c := c.(type) {
		case vkHandle:
			if c == vkNullHandle {
				debug(ctx, "Write to VK_NULL_HANDLE is ignored")
				allSucceeded = false
				continue
			}
		}
		bh.Modify(c)
		debug(ctx, "<Behavior: %v, Modify: %v>", bh, c)
	}
	return allSucceeded
}

func framebufferPortCoveredByClearRect(fb FramebufferObjectʳ, r VkClearRect) bool {
	if r.BaseArrayLayer() == uint32(0) &&
		r.LayerCount() == fb.Layers() &&
		r.Rect().Offset().X() == 0 && r.Rect().Offset().Y() == 0 &&
		r.Rect().Extent().Width() == fb.Width() &&
		r.Rect().Extent().Height() == fb.Height() {
		return true
	}
	return false
}

func clearAttachmentData(ctx context.Context, bh *dependencygraph.Behavior,
	execInfo *queueExecutionState, a VkClearAttachment, rects []VkClearRect) {
	subpass := &execInfo.subpasses[execInfo.subpass.val]
	if a.AspectMask() == VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) ||
		a.AspectMask() == VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) {
		if subpass.depthStencilAttachment != nil {
			modify(ctx, bh, subpass.depthStencilAttachment.data...)
			return
		}
	} else if a.AspectMask() == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT|
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) {
		if subpass.depthStencilAttachment != nil {
			overwritten := false
			for _, r := range rects {
				if framebufferPortCoveredByClearRect(execInfo.framebuffer, r) {
					overwritten = true
				}
			}
			if overwritten && subpass.depthStencilAttachment.fullImageData {
				write(ctx, bh, subpass.depthStencilAttachment.data...)
				return
			}
			modify(ctx, bh, subpass.depthStencilAttachment.data...)
			return
		}
	} else {
		if a.ColorAttachment() != vkAttachmentUnused {
			overwritten := false
			for _, r := range rects {
				if framebufferPortCoveredByClearRect(execInfo.framebuffer, r) {
					overwritten = true
				}
			}
			att := subpass.colorAttachments[a.ColorAttachment()]
			if overwritten && att.fullImageData {
				write(ctx, bh, att.data...)
				return
			}
			modify(ctx, bh, att.data...)
			return
		}
	}
}

func subresourceLayersFullyCoverImage(img ImageObjectʳ, layers VkImageSubresourceLayers,
	offset VkOffset3D, extent VkExtent3D) bool {
	if offset.X() != 0 || offset.Y() != 0 || offset.Z() != 0 {
		return false
	}
	if extent.Width() != img.Info().Extent().Width() ||
		extent.Height() != img.Info().Extent().Height() ||
		extent.Depth() != img.Info().Extent().Depth() {
		return false
	}
	if layers.BaseArrayLayer() != uint32(0) {
		return false
	}
	if layers.LayerCount() != img.Info().ArrayLayers() {
		return false
	}
	// Be conservative, only returns true if both the depth and the stencil
	// bits are set.
	if layers.AspectMask() == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT|
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) {
		return true
	}
	// For color images, returns true
	if layers.AspectMask() == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT) {
		return true
	}
	return false
}

func subresourceRangeFullyCoverImage(img ImageObjectʳ, rng VkImageSubresourceRange) bool {
	if rng.BaseArrayLayer() != 0 || rng.BaseMipLevel() != 0 {
		return false
	}
	if rng.LayerCount() != img.Info().ArrayLayers() || rng.LevelCount() != img.Info().MipLevels() {
		return false
	}
	// Be conservative, only returns true if both the depth and the stencil bits
	// are set.
	if rng.AspectMask() == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT|
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) ||
		rng.AspectMask() == VkImageAspectFlags(
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT) {
		return true
	}
	return false
}

func blitFullyCoverImage(img ImageObjectʳ, layers VkImageSubresourceLayers,
	offset1 VkOffset3D, offset2 VkOffset3D) bool {

	tmpArena := arena.New()
	defer tmpArena.Dispose()

	if offset1.X() == 0 && offset1.Y() == 0 && offset1.Z() == 0 {
		offset := offset1
		extent := NewVkExtent3D(tmpArena,
			uint32(offset2.X()-offset1.X()),
			uint32(offset2.Y()-offset1.Y()),
			uint32(offset2.Z()-offset1.Z()),
		)
		return subresourceLayersFullyCoverImage(img, layers, offset, extent)
	} else if offset2.X() == 0 && offset2.Y() == 0 && offset2.Z() == 0 {
		offset := offset2
		extent := NewVkExtent3D(tmpArena,
			uint32(offset1.X()-offset2.X()),
			uint32(offset1.Y()-offset2.Y()),
			uint32(offset1.Z()-offset2.Z()),
		)
		return subresourceLayersFullyCoverImage(img, layers, offset, extent)
	} else {
		return false
	}
}

func getSubBufferData(bufData memorySpan, offset, size uint64) memorySpan {
	start := offset + bufData.span.Start
	end := bufData.span.End
	if size != vkWholeSize {
		end = start + size
	}
	return memorySpan{
		span:   interval.U64Span{Start: start, End: end},
		memory: bufData.memory,
	}
}

func sparseImageMemoryBindGranularity(ctx context.Context, imgObj ImageObjectʳ,
	bind VkSparseImageMemoryBind) (VkExtent3D, bool) {
	for _, r := range imgObj.SparseMemoryRequirements().All() {
		if r.FormatProperties().AspectMask() == bind.Subresource().AspectMask() {
			return r.FormatProperties().ImageGranularity(), true
		}
	}
	log.E(ctx, "Sparse image granularity not found for VkImage: %v, "+
		"with AspectMask: %v", imgObj.VulkanHandle(), bind.Subresource().AspectMask())
	return VkExtent3D{}, false
}
