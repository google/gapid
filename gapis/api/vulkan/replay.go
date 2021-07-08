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
	"flag"
	"fmt"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/commandGenerator"
	"github.com/google/gapid/gapis/api/controlFlowGenerator"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

var (
	lenientDeviceMatching = flag.Bool("lenient-device-matching", false, "_Makes replay device matching more lenient")
)

// GetInitialPayload creates a replay that emits instructions for
// state priming of a capture.
func (a API) GetInitialPayload(ctx context.Context,
	c *path.Capture,
	device *device.Instance,
	out transform.Writer) error {

	initialCmds, im, _ := initialcmds.InitialCommands(ctx, c)
	out.State().Allocator.ReserveRanges(im)
	cmdGenerator := commandGenerator.NewLinearCommandGenerator(initialCmds, nil)

	var transforms []transform.Transform
	transforms = append(transforms, newMakeAttachmentReadable(false))
	transforms = append(transforms, newDropInvalidDestroy("GetInitialPayload"))
	transforms = append(transforms, newExternalMemory())
	if config.LogInitialCmdsToCapture {
		if c, err := capture.ResolveGraphicsFromPath(ctx, c); err == nil {
			transforms = append(transforms, newCaptureLog(ctx, c, "initial_cmds.gfxtrace"))
		}
	}

	chain := transform.CreateTransformChain(ctx, cmdGenerator, transforms, out)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	if err := controlFlow.TransformAll(ctx); err != nil {
		log.E(ctx, "[GetInitialPayload] Error: %v", err)
		return err
	}

	return nil
}

// CleanupResources creates a replay that emits instructions for
// destroying resources at a given stateg
func (a API) CleanupResources(ctx context.Context, device *device.Instance, out transform.Writer) error {
	cmdGenerator := commandGenerator.NewLinearCommandGenerator(nil, nil)
	transforms := []transform.Transform{
		newDestroyResourcesAtEOS(),
	}
	chain := transform.CreateTransformChain(ctx, cmdGenerator, transforms, out)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	if err := controlFlow.TransformAll(ctx); err != nil {
		log.E(ctx, "[CleanupResources] Error: %v", err)
		return err
	}

	return nil
}

func (a API) Replay(
	ctx context.Context,
	intent replay.Intent,
	cfg replay.Config,
	dependentPayload string,
	rrs []replay.RequestAndResult,
	device *device.Instance,
	c *capture.GraphicsCapture,
	out transform.Writer) error {
	priority, incompatibleReason := a.GetReplayPriority(ctx, device, c.Header)
	if priority == 0 {
		return log.Errf(ctx, nil, "Cannot replay Vulkan commands on device '%v', reason: %v", device.Name, incompatibleReason)
	}

	if len(rrs) == 0 {
		return log.Errf(ctx, nil, "No request has been found for the replay")
	}

	firstRequest := rrs[0].Request
	replayType := getReplayTypeName(firstRequest)

	if len(rrs) > 1 {
		if _, ok := firstRequest.(framebufferRequest); !ok {
			// Only the framebuffer requests are batched
			panic(fmt.Sprintf("Batched request is not supported for %v", replayType))
		}
	}

	_, isProfileRequest := firstRequest.(profileRequest)

	var transforms []transform.Transform
	transforms = append(transforms, newMakeAttachmentReadable(isProfileRequest))
	transforms = append(transforms, newDropInvalidDestroy(replayType))
	transforms = append(transforms, newExternalMemory())

	// Melih TODO: DCE probably should be here
	initialCmds := getInitialCmds(ctx, dependentPayload, intent, out)
	numOfInitialCmds := api.CmdID(len(initialCmds))

	// Due to how replay system works, different types of replays cannot be batched
	switch firstRequest.(type) {
	case framebufferRequest:
		framebufferTransforms, err := getFramebufferTransforms(ctx, numOfInitialCmds, intent, cfg, rrs)
		if err != nil {
			log.E(ctx, "%v Error: %v", replayType, err)
			return err
		}
		transforms = append(transforms, framebufferTransforms...)
	case issuesRequest:
		transforms = append(transforms, getIssuesTransforms(ctx, c, numOfInitialCmds, &rrs[0])...)
	case profileRequest:
		profileTransforms, err := getProfileTransforms(ctx, numOfInitialCmds, device, &rrs[0])
		if err != nil {
			log.E(ctx, "%v Error: %v", replayType, err)
			return err
		}
		transforms = append(transforms, profileTransforms...)
	case timestampsRequest:
		transforms = append(transforms, getTimestampTransforms(ctx, &rrs[0])...)
	default:
		panic("Unknown request type")
	}

	transforms = append(transforms, newDestroyResourcesAtEOS())
	transforms = appendLogTransforms(ctx, replayType, c, transforms)

	cmdGenerator := commandGenerator.NewLinearCommandGenerator(initialCmds, c.Commands)

	// Handle this if it's a profile request and return
	if request, ok := firstRequest.(profileRequest); ok {

		loopStart := numOfInitialCmds
		loopEnd := api.CmdID(len(initialCmds) + len(c.Commands) - 1)
		nullWriterObj := nullWriter{state: cloneStateWithSharedAllocator(ctx, c, out.State())}
		chain := transform.CreateTransformChain(ctx, cmdGenerator, transforms, nullWriterObj)
		loopCallbacks := getPerfettoLoopCallbacks(request.traceOptions, request.handler, request.buffer)
		controlFlow := NewLoopingVulkanControlFlowGenerator(ctx, chain, out, c, loopStart, loopEnd, request.loopCount, loopCallbacks)

		if err := controlFlow.TransformAll(ctx); err != nil {
			log.E(ctx, "%v Error: %v", replayType, err)
			return err
		}

	} else {

		// Handle all other types of request in the normal way.
		chain := transform.CreateTransformChain(ctx, cmdGenerator, transforms, out)
		controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)

		if err := controlFlow.TransformAll(ctx); err != nil {
			log.E(ctx, "%v Error: %v", replayType, err)
			return err
		}
	}

	return nil
}

func getFramebufferTransforms(ctx context.Context,
	numOfInitialCmds api.CmdID,
	intent replay.Intent,
	cfg replay.Config,
	rrs []replay.RequestAndResult) ([]transform.Transform, error) {

	shouldRenderWired := false
	doDisplayToSurface := false
	shouldOverDraw := false

	vulkanTerminator, err := newVulkanTerminator(ctx, intent.Capture, numOfInitialCmds)
	if err != nil {
		log.E(ctx, "Vulkan terminator failed: %v", err)
		return nil, err
	}

	splitterTransform := NewCommandSplitter(ctx)
	readFramebufferTransform := newReadFramebuffer(ctx)
	overdrawTransform := NewStencilOverdraw()

	for _, rr := range rrs {
		request := rr.Request.(framebufferRequest)

		cfg := cfg.(drawConfig)
		cmdID := request.after[0]

		if cfg.drawMode == path.DrawMode_OVERDRAW {
			// TODO(subcommands): Add subcommand support here
			if err := vulkanTerminator.Add(ctx, api.CmdID(cmdID), request.after[1:]); err != nil {
				log.E(ctx, "Vulkan terminator error on Cmd(%v) : %v", cmdID, err)
				return nil, err
			}
			shouldOverDraw = true
			overdrawTransform.add(ctx, request.after, intent.Capture, rr.Result)
			break
		}

		if err := vulkanTerminator.Add(ctx, api.CmdID(cmdID), api.SubCmdIdx{}); err != nil {
			log.E(ctx, "Vulkan terminator error on Cmd(%v) : %v", cmdID, err)
			return nil, err
		}

		subIdx := append(api.SubCmdIdx{}, request.after...)
		splitterTransform.Split(ctx, subIdx)

		switch cfg.drawMode {
		case path.DrawMode_WIREFRAME_ALL:
			shouldRenderWired = true
		case path.DrawMode_WIREFRAME_OVERLAY:
			return nil, fmt.Errorf("Overlay wireframe view is not currently supported")
			// Overdraw is handled above, since it breaks out of the normal read flow.
		}

		switch request.attachment {
		case api.FramebufferAttachmentType_OutputDepth, api.FramebufferAttachmentType_InputDepth:
			readFramebufferTransform.Depth(ctx, subIdx, request.width, request.height, request.framebufferIndex, rr.Result)
		case api.FramebufferAttachmentType_OutputColor, api.FramebufferAttachmentType_InputColor:
			readFramebufferTransform.Color(ctx, subIdx, request.width, request.height, request.framebufferIndex, rr.Result)
		default:
			return nil, fmt.Errorf("Stencil attachments are not currently supported")
		}

		if request.displayToSurface {
			doDisplayToSurface = true
		}
	}

	transforms := make([]transform.Transform, 0)

	if shouldRenderWired {
		transforms = append(transforms, newWireframeTransform())
	}

	if doDisplayToSurface {
		transforms = append(transforms, newDisplayToSurface())
	}

	transforms = append(transforms, vulkanTerminator)

	if shouldOverDraw {
		transforms = append(transforms, overdrawTransform)
	}

	transforms = append(transforms, splitterTransform)
	transforms = append(transforms, readFramebufferTransform)
	return transforms, nil
}

func getIssuesTransforms(ctx context.Context,
	c *capture.GraphicsCapture,
	numOfInitialCmds api.CmdID,
	requestAndResult *replay.RequestAndResult) []transform.Transform {
	transforms := make([]transform.Transform, 0)

	issuesTransform := newFindIssues(ctx, c, numOfInitialCmds)
	issuesTransform.AddResult(requestAndResult.Result)
	transforms = append(transforms, issuesTransform)

	request := requestAndResult.Request.(issuesRequest)
	if request.displayToSurface {
		transforms = append(transforms, newDisplayToSurface())
	}

	return transforms
}

func getProfileTransforms(ctx context.Context,
	numOfInitialCmds api.CmdID,
	device *device.Instance,
	requestAndResult *replay.RequestAndResult) ([]transform.Transform, error) {
	var layerName string
	if device.GetConfiguration().GetPerfettoCapability().GetGpuProfiling().GetHasRenderStageProducerLayer() {
		layerName = "VkRenderStagesProducer"
	}

	request := requestAndResult.Request.(profileRequest)

	profileTransform := newEndOfReplay()
	profileTransform.AddResult(requestAndResult.Result)

	transforms := make([]transform.Transform, 0)
	transforms = append(transforms, newProfilingLayers(layerName))
	transforms = append(transforms, newMappingExporter(ctx, uint64(numOfInitialCmds), request.handleMappings))

	if request.experiments.DisableAnisotropicFiltering {
		transforms = append(transforms, newAfDisablerTransform())
	}

	var err error
	if len(request.experiments.DisabledCmds) > 0 {
		disablerTransform := newCommandDisabler(ctx, uint64(numOfInitialCmds))
		for _, disabledCmdID := range request.experiments.DisabledCmds {
			subIdx := append(api.SubCmdIdx{}, disabledCmdID...)
			err = disablerTransform.remove(ctx, subIdx)
		}
		transforms = append(transforms, disablerTransform)
	}

	transforms = append(transforms, profileTransform)
	return transforms, err
}

func getTimestampTransforms(ctx context.Context,
	requestAndResult *replay.RequestAndResult) []transform.Transform {
	request := requestAndResult.Request.(timestampsRequest)
	timestampTransform := newQueryTimestamps(ctx, request.handler)
	timestampTransform.AddResult(requestAndResult.Result)
	return []transform.Transform{timestampTransform}
}

func appendLogTransforms(ctx context.Context, tag string, capture *capture.GraphicsCapture, transforms []transform.Transform) []transform.Transform {
	if config.LogTransformsToFile {
		newTransforms := make([]transform.Transform, 0)
		newTransforms = append(newTransforms, newFileLog(ctx, "0_original_cmds"))
		for i, t := range transforms {
			var name string
			if n, ok := t.(interface {
				Name() string
			}); ok {
				name = n.Name()
			} else {
				name = strings.Replace(fmt.Sprintf("%T", t), "*", "", -1)
			}
			newTransforms = append(newTransforms, t, newFileLog(ctx, fmt.Sprintf("%v_cmds_after_%v", i+1, name)))
		}
		transforms = newTransforms
	}

	if config.LogTransformsToCapture {
		transforms = append(transforms, newCaptureLog(ctx, capture, tag+"_replay_log.gfxtrace"))
	}

	if config.LogMappingsToFile {
		transforms = append(transforms, newMappingExporterWithPrint(ctx, tag+"_mappings.txt"))
	}

	return transforms
}

func getInitialCmds(ctx context.Context,
	dependentPayload string,
	intent replay.Intent,
	out transform.Writer) []api.Cmd {

	if dependentPayload == "" {
		cmds, im, _ := initialcmds.InitialCommands(ctx, intent.Capture)
		out.State().Allocator.ReserveRanges(im)
		return cmds
	}

	return []api.Cmd{}
}

func getReplayTypeName(request replay.Request) string {
	switch request.(type) {
	case framebufferRequest:
		return "Framebuffer Replay"
	case issuesRequest:
		return "Issues Replay"
	case profileRequest:
		return "Profile Replay"
	case timestampsRequest:
		return "Timestamp Replay"
	}

	panic("Unknown replay type")
	return "Unknown Replay Type"
}

// GetReplayPriority implements the replay.Support interface
func (a API) GetReplayPriority(ctx context.Context, i *device.Instance, h *capture.Header) (uint32, *stringtable.Msg) {
	devConf := i.GetConfiguration()
	devAbis := devConf.GetABIs()
	devVkDriver := devConf.GetDrivers().GetVulkan()
	traceVkDriver := h.GetDevice().GetConfiguration().GetDrivers().GetVulkan()

	// Trace has no Vulkan information
	if traceVkDriver == nil {
		log.E(ctx, "Vulkan trace does not contain VulkanDriver info")
		return 0, messages.ReplayCompatibilityIncompatibleApi()
	}

	// The device does not support Vulkan
	if devVkDriver == nil {
		return 0, messages.ReplayCompatibilityIncompatibleApi()
	}

	// OSKind must match
	devOSKind := devConf.GetOS().GetKind()
	traceOSKind := h.GetABI().GetOS()
	if devOSKind != traceOSKind {
		return 0, messages.ReplayCompatibilityIncompatibleOs(devOSKind.String(), traceOSKind.String())
	}

	var reason *stringtable.Msg
	for _, abi := range devAbis {

		// Architecture must match
		if abi.GetArchitecture() != h.GetABI().GetArchitecture() {
			continue
		}

		// If there is no physical devices, the trace must not contain
		// vkCreateInstance, any ABI compatible Vulkan device should be able to
		// replay.
		if len(traceVkDriver.GetPhysicalDevices()) == 0 {
			return 1, messages.ReplayCompatibilityCompatible()
		}
		// Requires same GPU vendor, GPU device, Vulkan driver and Vulkan API version.
		for _, devPhyInfo := range devVkDriver.GetPhysicalDevices() {
			for _, tracePhyInfo := range traceVkDriver.GetPhysicalDevices() {
				if devPhyInfo.GetVendorId() != tracePhyInfo.GetVendorId() {
					reason = messages.ReplayCompatibilityIncompatibleGpu(devPhyInfo.GetDeviceName(), tracePhyInfo.GetDeviceName())
					continue
				}
				if devPhyInfo.GetDeviceId() != tracePhyInfo.GetDeviceId() {
					reason = messages.ReplayCompatibilityIncompatibleGpu(devPhyInfo.GetDeviceName(), tracePhyInfo.GetDeviceName())
					continue
				}
				if !*lenientDeviceMatching {
					if devPhyInfo.GetDriverVersion() != tracePhyInfo.GetDriverVersion() {
						reason = messages.ReplayCompatibilityIncompatibleDriverVersion(devPhyInfo.GetDriverVersion(), tracePhyInfo.GetDriverVersion())
						continue
					}
					// Ignore the API patch level (bottom 12 bits) when comparing the API version.
					if (devPhyInfo.GetApiVersion() & ^uint32(0xfff)) != (tracePhyInfo.GetApiVersion() & ^uint32(0xfff)) {
						reason = messages.ReplayCompatibilityIncompatibleApiVersion(devPhyInfo.GetApiVersion(), tracePhyInfo.GetApiVersion())
						continue
					}
				}
				return 1, messages.ReplayCompatibilityCompatible()
			}
		}
	}

	if reason == nil {
		// None of the device ABI architecture has matched
		reason = messages.ReplayCompatibilityIncompatibleArchitecture(h.GetABI().GetArchitecture().String())
	}
	return 0, reason
}

// nullWriter conforms to the the transformer.Writer interface, it just updates a state object and does nothing with the commands
type nullWriter struct {
	state *api.GlobalState
}

func (w nullWriter) State() *api.GlobalState {
	return w.state
}

func (w nullWriter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	return cmd.Mutate(ctx, id, w.state, nil, nil)
}

func cloneStateWithSharedAllocator(ctx context.Context, capture *capture.GraphicsCapture, state *api.GlobalState) *api.GlobalState {

	clone := capture.NewUninitializedStateSharingAllocator(ctx, state)
	clone.Memory = state.Memory.Clone()

	for apiState, graphicsApi := range state.APIs {

		clonedState := graphicsApi.Clone()
		clonedState.SetupInitialState(ctx, clone)

		clone.APIs[apiState] = clonedState
	}

	return clone
}
