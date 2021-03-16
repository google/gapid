// Copyright (C) 2020 Google Inc.
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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

var _ transform.Transform = &displayToSurface{}

// displayToSurface is a transformation that enables rendering during replay to
// the original surface.
type displayToSurface struct {
	surfaceTypes map[uint64]uint32
}

func newDisplayToSurface() *displayToSurface {
	return &displayToSurface{
		surfaceTypes: map[uint64]uint32{},
	}
}

func (surfaceTransform *displayToSurface) RequiresAccurateState() bool {
	return false
}

func (surfaceTransform *displayToSurface) RequiresInnerStateMutation() bool {
	return false
}

func (surfaceTransform *displayToSurface) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (surfaceTransform *displayToSurface) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	return nil
}

func (surfaceTransform *displayToSurface) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (surfaceTransform *displayToSurface) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}

func (surfaceTransform *displayToSurface) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	for i, cmd := range inputCommands {
		modifiedCmd, err := surfaceTransform.modifySurface(ctx, cmd, inputState)
		if err != nil {
			return nil, err
		}
		if modifiedCmd != nil {
			inputCommands[i] = modifiedCmd
		}
	}

	return inputCommands, nil
}

func (surfaceTransform *displayToSurface) modifySurface(ctx context.Context, cmd api.Cmd, inputState *api.GlobalState) (api.Cmd, error) {
	if swapchainCmd, ok := cmd.(*VkCreateSwapchainKHR); ok {
		newCmd := swapchainCmd.clone()
		newCmd.extras = api.CmdExtras{}
		// Add an extra to indicate to custom_replay to add a flag to
		// the virtual swapchain pNext
		newCmd.extras = append(api.CmdExtras{surfaceTransform}, swapchainCmd.Extras().All()...)
		return newCmd, nil
	}

	switch c := cmd.(type) {
	case *VkCreateAndroidSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
		surface, err := c.PSurface().Read(ctx, cmd, inputState, nil)
		if err != nil {
			return nil, err
		}
		surfaceTransform.surfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR)
	case *VkCreateWaylandSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
		surface, err := c.PSurface().Read(ctx, cmd, inputState, nil)
		if err != nil {
			return nil, err
		}
		surfaceTransform.surfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_WAYLAND_SURFACE_CREATE_INFO_KHR)
	case *VkCreateWin32SurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
		surface, err := c.PSurface().Read(ctx, cmd, inputState, nil)
		if err != nil {
			return nil, err
		}
		surfaceTransform.surfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR)
	case *VkCreateXcbSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
		surface, err := c.PSurface().Read(ctx, cmd, inputState, nil)
		if err != nil {
			return nil, err
		}
		surfaceTransform.surfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_XCB_SURFACE_CREATE_INFO_KHR)
	case *VkCreateXlibSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
		surface, err := c.PSurface().Read(ctx, cmd, inputState, nil)
		if err != nil {
			return nil, err
		}
		surfaceTransform.surfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_XLIB_SURFACE_CREATE_INFO_KHR)
	case *VkCreateMacOSSurfaceMVK:
		cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
		surface, err := c.PSurface().Read(ctx, cmd, inputState, nil)
		if err != nil {
			return nil, err
		}
		surfaceTransform.surfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_MACOS_SURFACE_CREATE_INFO_MVK)
	default:
		return nil, nil
	}

	return cmd, nil
}
