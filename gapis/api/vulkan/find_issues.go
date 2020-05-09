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
	"github.com/google/gapid/gapis/config"
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

// findIssues is a command transform that detects issues when replaying the
// stream of commands. Any issues that are found are written to all the chans in
// the slice out. Once the last issue is sent (if any) all the chans in out are
// closed.
// NOTE: right now this transform is just used to close chans passed in requests.
type findIssues struct {
	replay.EndOfReplay
	state           *api.GlobalState
	issues          []replay.Issue
	reportCallbacks map[VkInstance]VkDebugReportCallbackEXT
}

func newFindIssues(ctx context.Context, c *capture.GraphicsCapture) *findIssues {
	t := &findIssues{
		state:           c.NewState(ctx),
		reportCallbacks: map[VkInstance]VkDebugReportCallbackEXT{},
	}
	t.state.OnError = func(err interface{}) {
		if issue, ok := err.(replay.Issue); ok {
			t.issues = append(t.issues, issue)
		}
	}
	return t
}

func (t *findIssues) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	ctx = log.Enter(ctx, "findIssues")

	mutateErr := cmd.Mutate(ctx, id, t.state, nil /* no builder */, nil /* no watcher */)
	if mutateErr != nil {
		// Ignore since downstream transform layers can only consume valid commands
		return nil
	}

	s := out.State()
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: out.State().Arena}
	l := s.MemoryLayout
	allocated := []api.AllocResult{}
	defer func() {
		for _, d := range allocated {
			d.Free()
		}
	}()
	mustAlloc := func(ctx context.Context, v ...interface{}) api.AllocResult {
		res := s.AllocDataOrPanic(ctx, v...)
		allocated = append(allocated, res)
		return res
	}

	// Before an instance is to be destroyed, check if it has debug report callback
	// created by us, if so, destory the call back handle.
	if di, ok := cmd.(*VkDestroyInstance); ok {
		inst := di.Instance()
		if ch, ok := t.reportCallbacks[inst]; ok {
			out.MutateAndWrite(ctx, api.CmdNoID, cb.ReplayDestroyVkDebugReportCallback(inst, ch))
			delete(t.reportCallbacks, inst)
		}
	}

	switch cmd := cmd.(type) {
	// Modify the vkCreateInstance to first remove any validation layers,
	// and then insert the meta validation layer. Also enable the
	// VK_EXT_debug_report extension.
	case *VkCreateInstance:
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		info := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
		layers := []Charᶜᵖ{}

		validationMetaLayerData := mustAlloc(ctx, validationMetaLayer)
		layers = append(layers, NewCharᶜᵖ(validationMetaLayerData.Ptr()))
		layersData := mustAlloc(ctx, layers)

		extCount := info.EnabledExtensionCount()
		exts := info.PpEnabledExtensionNames().Slice(0, uint64(extCount), l).MustRead(ctx, cmd, s, nil)
		var debugReportExtNameData api.AllocResult
		hasDebugReport := false
		for _, e := range exts {
			// TODO(chrisforbes): provide a better way of getting the contents of the string
			if debugReportExtension == strings.TrimRight(string(memory.CharToBytes(e.StringSlice(ctx, s).MustRead(ctx, cmd, s, nil))), "\x00") {
				hasDebugReport = true
			}
		}
		if !hasDebugReport {
			debugReportExtNameData = mustAlloc(ctx, debugReportExtension)
			exts = append(exts, NewCharᶜᵖ(debugReportExtNameData.Ptr()))
		}
		extsData := mustAlloc(ctx, exts)

		info.SetEnabledLayerCount(uint32(len(layers)))
		info.SetPpEnabledLayerNames(NewCharᶜᵖᶜᵖ(layersData.Ptr()))
		info.SetEnabledExtensionCount(uint32(len(exts)))
		info.SetPpEnabledExtensionNames(NewCharᶜᵖᶜᵖ(extsData.Ptr()))
		infoData := mustAlloc(ctx, info)

		newCmd := cb.VkCreateInstance(infoData.Ptr(), cmd.PAllocator(), cmd.PInstance(), cmd.Result())
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
		out.MutateAndWrite(ctx, id, newCmd)

	default:
		out.MutateAndWrite(ctx, id, cmd)

	}

	// After an instance is created, try to create a debug report call back handle
	// for it. The create info is not completed, the device side code should complete
	// the create info before calling the underlying Vulkan command.
	if ci, ok := cmd.(*VkCreateInstance); ok {
		inst := ci.PInstance().MustRead(ctx, cmd, s, nil)
		callbackHandle := VkDebugReportCallbackEXT(newUnusedID(true, func(x uint64) bool {
			for _, cb := range t.reportCallbacks {
				if uint64(cb) == x {
					return true
				}
			}
			return false
		}))
		callbackHandleData := mustAlloc(ctx, callbackHandle)
		callbackCreateInfo := mustAlloc(
			ctx, NewVkDebugReportCallbackCreateInfoEXT(
				s.Arena,
				VkStructureType_VK_STRUCTURE_TYPE_DEBUG_REPORT_CREATE_INFO_EXT, // sType
				0, // pNext
				VkDebugReportFlagsEXT((VkDebugReportFlagBitsEXT_VK_DEBUG_REPORT_DEBUG_BIT_EXT<<1)-1), // flags
				0, // pfnCallback
				0, // pUserData
			))
		out.MutateAndWrite(ctx, api.CmdNoID, cb.ReplayCreateVkDebugReportCallback(
			inst,
			callbackCreateInfo.Ptr(),
			callbackHandleData.Ptr(),
			true,
		).AddRead(
			callbackCreateInfo.Data(),
		).AddWrite(
			callbackHandleData.Data(),
		))
		t.reportCallbacks[inst] = callbackHandle
	}
	return nil
}

func (t *findIssues) Flush(ctx context.Context, out transform.Writer) error {
	cb := CommandBuilder{Thread: 0, Arena: out.State().Arena}
	for inst, ch := range t.reportCallbacks {
		if err := out.MutateAndWrite(ctx, api.CmdNoID, cb.ReplayDestroyVkDebugReportCallback(inst, ch)); err != nil {
			return err
		}
		// It is safe to delete keys in loop in Go
		delete(t.reportCallbacks, inst)
	}
	err := out.MutateAndWrite(ctx, api.CmdNoID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		return b.RegisterNotificationReader(builder.IssuesNotificationID, func(n gapir.Notification) {
			vkApi := API{}
			eMsg := n.GetErrorMsg()
			if eMsg == nil {
				return
			}
			if uint8(eMsg.GetApiIndex()) != vkApi.Index() {
				return
			}
			var issue replay.Issue
			msg := eMsg.GetMsg()
			label := eMsg.GetLabel()
			issue.Command = api.CmdID(label)
			issue.Severity = service.Severity(uint32(eMsg.GetSeverity()))

			if issue.Command == api.CmdNoID {
				// TODO: Fix all the errors reported for initial commands.
				if config.LogInitialCmdsIssues {
					log.E(ctx, "Error in state rebuilding command : %s", msg)
				}
			} else {
				// The debug report is issued for a trace command
				issue.Error = fmt.Errorf("%s", msg)
				t.issues = append(t.issues, issue)
			}
		})
	}))
	if err != nil {
		return err
	}
	t.AddNotifyInstruction(ctx, out, func() interface{} { return t.issues })
	return nil
}
