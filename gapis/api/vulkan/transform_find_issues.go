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
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service"
)

const (
	// Since Android NDK r21, the VK_LAYER_KHRONOS_validation meta layer
	// is available on both desktop and Android.
	validationMetaLayer  = "VK_LAYER_KHRONOS_validation"
	debugReportExtension = "VK_EXT_debug_report"
)

var _ transform.Transform = &findIssues{}

// findIssues is a command transform that detects issues when replaying the
// stream of commands. Any issues that are found are written to all the chans in
// the slice out. Once the last issue is sent (if any) all the chans in out are
// closed.
// NOTE: right now this transform is just used to close chans passed in requests.
type findIssues struct {
	endOfReplay
	state           *api.GlobalState
	issues          []replay.Issue
	reportCallbacks map[VkInstance]VkDebugReportCallbackEXT
	allocations     *allocationTracker
	realCmdOffset   api.CmdID
}

func newFindIssues(ctx context.Context, c *capture.GraphicsCapture, realCmdOffset api.CmdID) *findIssues {
	t := &findIssues{
		state:           c.NewState(ctx),
		reportCallbacks: map[VkInstance]VkDebugReportCallbackEXT{},
		allocations:     nil,
		realCmdOffset:   realCmdOffset,
	}

	t.state.OnError = func(err interface{}) {
		if issue, ok := err.(replay.Issue); ok {
			t.filterAndAppendIssues(ctx, issue)
		}
	}
	return t
}

func (issueTransform *findIssues) RequiresAccurateState() bool {
	return false
}

func (issueTransform *findIssues) RequiresInnerStateMutation() bool {
	return false
}

func (issueTransform *findIssues) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (issueTransform *findIssues) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	issueTransform.allocations = NewAllocationTracker(inputState)
	return nil
}

func (issueTransform *findIssues) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	ctx = log.Enter(ctx, "findIssues")

	outputCmds := make([]api.Cmd, 0, len(inputCommands))

	for _, cmd := range inputCommands {
		mutateErr := cmd.Mutate(ctx, id.GetID(), issueTransform.state, nil /* no builder */, nil /* no watcher */)
		if mutateErr != nil {
			// Ignore since downstream transform layers can only consume valid commands
			return outputCmds, mutateErr
		}

		if destroyInstanceCommand, ok := cmd.(*VkDestroyInstance); ok {
			// Before an instance is to be destroyed, check if it has debug report callback
			// created by us, if so, destroy the callback handle.
			newCmd := issueTransform.destroyDebugReportCallback(destroyInstanceCommand, inputState)
			if newCmd != nil {
				outputCmds = append(outputCmds, newCmd)
			}
		}

		if createInstanceCmd, ok := cmd.(*VkCreateInstance); ok {
			// Modify the vkCreateInstance to first remove any validation layers,
			// and then insert the meta validation layer. Also enable the
			// VK_EXT_debug_report extension.
			newCmd := issueTransform.modifyVkCreateInstance(ctx, createInstanceCmd, inputState)
			outputCmds = append(outputCmds, newCmd)
		} else {
			outputCmds = append(outputCmds, cmd)
		}

		// After an instance is created, try to create a debug report call back handle
		// for it. The create info is not completed, the device side code should complete
		// the create info before calling the underlying Vulkan command.
		if createInstanceCommand, ok := cmd.(*VkCreateInstance); ok {
			debugCmd := issueTransform.createDebugReportCallback(ctx, createInstanceCommand, inputState)
			if debugCmd != nil {
				outputCmds = append(outputCmds, cmd)
			}
		}
	}

	return outputCmds, nil
}

func (issueTransform *findIssues) ClearTransformResources(ctx context.Context) {
	issueTransform.allocations.FreeAllocations()
}

func (issueTransform *findIssues) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	cmds := make([]api.Cmd, 0)

	commandBuilder := CommandBuilder{Thread: 0}
	for instance, callback := range issueTransform.reportCallbacks {
		newCmd := commandBuilder.ReplayDestroyVkDebugReportCallback(instance, callback)
		if newCmd != nil {
			cmds = append(cmds, newCmd)
		}
		// It is safe to delete keys in loop in Go
		delete(issueTransform.reportCallbacks, instance)
	}

	registerNotificationReader := commandBuilder.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		return b.RegisterNotificationReader(builder.IssuesNotificationID, func(notification gapir.Notification) {
			issueTransform.notificationReader(ctx, notification)
		})
	})

	notifyInstruction := issueTransform.CreateNotifyInstruction(ctx, func() interface{} {
		return issueTransform.issues
	})

	cmds = append(cmds, registerNotificationReader, notifyInstruction)
	return cmds, nil
}

func (issueTransform *findIssues) notificationReader(ctx context.Context, n gapir.Notification) {
	vkApi := API{}
	eMsg := n.GetErrorMsg()
	if eMsg == nil {
		return
	}
	if uint8(eMsg.GetApiIndex()) != vkApi.Index() {
		return
	}

	msg := eMsg.GetMsg()
	label := eMsg.GetLabel()

	var issue replay.Issue
	issue.Command = api.CmdID(label)
	issue.Severity = service.Severity(uint32(eMsg.GetSeverity()))
	issue.Error = fmt.Errorf("%s", msg)
	issueTransform.filterAndAppendIssues(ctx, issue)
}

func (issueTransform *findIssues) filterAndAppendIssues(ctx context.Context, issue replay.Issue) {
	if issue.Command < issueTransform.realCmdOffset {
		// TODO: Fix all the errors reported for initial commands.
		log.E(ctx, "Error in state rebuilding command : [%v]: %s", issue.Command, issue.Error)
	} else {
		issue.Command = issue.Command - issueTransform.realCmdOffset
		issueTransform.issues = append(issueTransform.issues, issue)
	}
}

func (issueTransform *findIssues) modifyVkCreateInstance(ctx context.Context, cmd *VkCreateInstance, inputState *api.GlobalState) api.Cmd {
	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())
	info := cmd.PCreateInfo().MustRead(ctx, cmd, inputState, nil)

	layers := []Charᶜᵖ{}
	validationMetaLayerData := issueTransform.allocations.AllocDataOrPanic(ctx, validationMetaLayer)
	layers = append(layers, NewCharᶜᵖ(validationMetaLayerData.Ptr()))
	layersData := issueTransform.allocations.AllocDataOrPanic(ctx, layers)

	extCount := info.EnabledExtensionCount()
	exts := info.PpEnabledExtensionNames().Slice(0, uint64(extCount), inputState.MemoryLayout).MustRead(ctx, cmd, inputState, nil)
	var debugReportExtNameData api.AllocResult
	hasDebugReport := false
	for _, e := range exts {
		// TODO(chrisforbes): provide a better way of getting the contents of the string
		if debugReportExtension == strings.TrimRight(string(memory.CharToBytes(e.StringSlice(ctx, inputState).MustRead(ctx, cmd, inputState, nil))), "\x00") {
			hasDebugReport = true
		}
	}
	if !hasDebugReport {
		debugReportExtNameData = issueTransform.allocations.AllocDataOrPanic(ctx, debugReportExtension)
		exts = append(exts, NewCharᶜᵖ(debugReportExtNameData.Ptr()))
	}
	extsData := issueTransform.allocations.AllocDataOrPanic(ctx, exts)

	info.SetEnabledLayerCount(uint32(len(layers)))
	info.SetPpEnabledLayerNames(NewCharᶜᵖᶜᵖ(layersData.Ptr()))
	info.SetEnabledExtensionCount(uint32(len(exts)))
	info.SetPpEnabledExtensionNames(NewCharᶜᵖᶜᵖ(extsData.Ptr()))
	infoData := issueTransform.allocations.AllocDataOrPanic(ctx, info)

	commandBuilder := CommandBuilder{Thread: cmd.Thread()}
	newCmd := commandBuilder.VkCreateInstance(infoData.Ptr(), cmd.PAllocator(), cmd.PInstance(), cmd.Result())
	newCmd.AddRead(
		validationMetaLayerData.Data(),
	).AddRead(
		debugReportExtNameData.Data(),
	).AddRead(
		infoData.Data(),
	).AddRead(
		layersData.Data(),
	).AddRead(
		extsData.Data(),
	)
	// Also add back all the other read/write observations of the original vkCreateInstance
	for _, r := range cmd.Extras().Observations().Reads {
		newCmd.AddRead(r.Range, r.ID)
	}
	for _, w := range cmd.Extras().Observations().Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}

	return newCmd
}

func (issueTransform *findIssues) createDebugReportCallback(ctx context.Context, cmd *VkCreateInstance, inputState *api.GlobalState) api.Cmd {
	instance := cmd.PInstance().MustRead(ctx, cmd, inputState, nil)
	callbackHandle := VkDebugReportCallbackEXT(newUnusedID(true, func(x uint64) bool {
		for _, callback := range issueTransform.reportCallbacks {
			if uint64(callback) == x {
				return true
			}
		}
		return false
	}))
	issueTransform.reportCallbacks[instance] = callbackHandle

	callbackHandleData := issueTransform.allocations.AllocDataOrPanic(ctx, callbackHandle)
	callbackCreateInfo := issueTransform.allocations.AllocDataOrPanic(
		ctx, NewVkDebugReportCallbackCreateInfoEXT(
			VkStructureType_VK_STRUCTURE_TYPE_DEBUG_REPORT_CREATE_INFO_EXT, // sType
			0, // pNext
			VkDebugReportFlagsEXT((VkDebugReportFlagBitsEXT_VK_DEBUG_REPORT_DEBUG_BIT_EXT<<1)-1), // flags
			0, // pfnCallback
			0, // pUserData
		))

	commandBuilder := CommandBuilder{Thread: cmd.Thread()}
	return commandBuilder.ReplayCreateVkDebugReportCallback(
		instance,
		callbackCreateInfo.Ptr(),
		callbackHandleData.Ptr(),
		true,
	).AddRead(
		callbackCreateInfo.Data(),
	).AddWrite(
		callbackHandleData.Data(),
	)
}

func (issueTransform *findIssues) destroyDebugReportCallback(cmd *VkDestroyInstance, inputState *api.GlobalState) api.Cmd {
	instance := cmd.Instance()
	callbackHandle, ok := issueTransform.reportCallbacks[instance]
	if !ok {
		return nil
	}

	commandBuilder := CommandBuilder{Thread: cmd.Thread()}
	newCmd := commandBuilder.ReplayDestroyVkDebugReportCallback(instance, callbackHandle)
	delete(issueTransform.reportCallbacks, instance)
	return newCmd
}
