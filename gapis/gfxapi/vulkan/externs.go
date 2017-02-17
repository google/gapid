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
	"reflect"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
)

type externs struct {
	ctx log.Context // Allowed because the externs struct is only a parameter proxy for a single call
	a   atom.Atom
	s   *gfxapi.State
	b   *rb.Builder
}

func (e externs) hasDynamicProperty(info VkPipelineDynamicStateCreateInfoᶜᵖ,
	state VkDynamicState) bool {
	if (info) == (VkPipelineDynamicStateCreateInfoᶜᵖ{}) {
		return false
	}
	dynamic_state_info := info.Slice(uint64(0), uint64(1), e.s).Index(uint64(0), e.s).Read(e.ctx, e.a, e.s, e.b)
	states := dynamic_state_info.PDynamicStates.Slice(uint64(uint32(0)), uint64(dynamic_state_info.DynamicStateCount), e.s).Read(e.ctx, e.a, e.s, e.b)
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func (e externs) mapMemory(value Voidᵖᵖ, slice slice) {
	ctx := e.ctx
	if b := e.b; b != nil {
		switch e.a.(type) {
		case *VkMapMemory:
			b.Load(protocol.Type_AbsolutePointer, value.value(e.b, e.a, e.s))
			b.MapMemory(slice.Range(e.s))
		default:
			ctx.Error().V("atom", e.a).Log("mapBuffer extern called for unsupported atom")
		}
	}
}

func (e externs) addCmd(commandBuffer VkCommandBuffer, recreate_data interface{}, data interface{}, functionToCall interface{}) {
	args := []reflect.Value{
		reflect.ValueOf(e.ctx),
		reflect.ValueOf(e.a),
		reflect.ValueOf(&atom.Observations{}),
		reflect.ValueOf(e.s),
		reflect.ValueOf(GetState(e.s)),
		reflect.ValueOf(e.b),
		reflect.ValueOf(data),
	}
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)

	o.Commands = append(o.Commands, CommandBufferCommand{func() {
		reflect.ValueOf(functionToCall).Call(args)
	}, &e.a})
}

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	o.Commands = []CommandBufferCommand{}
}

func (e externs) execCommands(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	for _, command := range o.Commands {
		command.function()
	}
}

func (e externs) createUpdateBufferData(buffer VkBuffer, offset VkDeviceSize, size VkDeviceSize, data interface{}) *RecreateVkCommandBuffer {
	return nil
}

func (e externs) addWords(module VkShaderModule, numBytes interface{}, data interface{}) {
}

func (e externs) setSpecData(module *SpecializationInfo, numBytes interface{}, data interface{}) {
}

func (e externs) unmapMemory(slice slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(slice.Range(e.s))
	}
}
