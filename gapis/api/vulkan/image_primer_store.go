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
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory"
)

const (
	ipStoreOutputImageBinding           = 0
	ipStoreInputImageBinding            = 1
	ipStoreMaxComputeGroupCountX        = 65536
	ipStoreMaxComputeGroupCountY        = 65536
	ipStoreMaxComputeGroupCountZ        = 65536
	ipStoreInitialDescriptorSetPoolSize = 16
	ipStoreImageLayout                  = VkImageLayout_VK_IMAGE_LAYOUT_GENERAL
)

var (
	descriptorSetLayoutInfoForStore ipDescriptorSetLayoutInfo
)

func init() {
	descriptorSetLayoutInfoForStore.bindings = map[uint32]ipDescriptorSetLayoutBindingInfo{}
	descriptorSetLayoutInfoForStore.bindings[ipStoreOutputImageBinding] = ipDescriptorSetLayoutBindingInfo{
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
		1, VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
	}
	descriptorSetLayoutInfoForStore.bindings[ipStoreInputImageBinding] = ipDescriptorSetLayoutBindingInfo{
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
		1, VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
	}
}

// ipStoreRecipe describes how a subresource region data of a storage image
// to be primed by imageLoad/Store operation.
type ipStoreRecipe struct {
	inputImage   VkImage
	inputAspect  VkImageAspectFlagBits
	outputImage  VkImage
	outputAspect VkImageAspectFlagBits
	layer        uint32
	level        uint32
	wordIndex    uint32
	extentWidth  uint32
	extentHeight uint32
	extentDepth  uint32
	offsetX      int32
	offsetY      int32
	offsetZ      int32
}

// ipStoreKitBuilder builds the kit used to generate commands to prime image
// data by imageLoad/Store operation.
type ipStoreKitBuilder struct {
	nm                  debugMarkerName
	dev                 VkDevice
	descriptorSetLayout VkDescriptorSetLayout
	pipelineLayout      VkPipelineLayout
	descSetPool         *homoDescriptorSetPool
	shaderModulePool    *naiveShaderModulePool
	imageViewPool       *naiveImageViewPool
	// Use a pair of map + slice to free them in order
	pipelinePoolIndex map[ipStorePipelineInfo]int
	pipelinePool      []VkPipeline
}

func newImagePrimerStoreKitBuilder(sb *stateBuilder, dev VkDevice) *ipStoreKitBuilder {
	builder := &ipStoreKitBuilder{
		nm:                debugMarkerName(fmt.Sprintf("store kit builder dev: %v", dev)),
		dev:               dev,
		shaderModulePool:  newNaiveShaderModulePool(dev),
		imageViewPool:     newNaiveImageViewPool(dev),
		pipelinePoolIndex: map[ipStorePipelineInfo]int{},
		pipelinePool:      []VkPipeline{},
	}
	builder.descriptorSetLayout = ipCreateDescriptorSetLayout(sb, builder.nm, dev, descriptorSetLayoutInfoForStore)
	builder.descSetPool = newHomoDescriptorSetPool(sb, builder.nm, dev, builder.descriptorSetLayout, ipStoreInitialDescriptorSetPoolSize, false)
	builder.pipelineLayout = ipCreatePipelineLayout(sb, builder.nm, dev,
		[]VkDescriptorSetLayout{builder.descriptorSetLayout},
		VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
		4*4)
	return builder
}

// Free frees all the resources used for all the kits built by this builder.
func (kb *ipStoreKitBuilder) Free(sb *stateBuilder) {
	if kb.descSetPool != nil {
		kb.descSetPool.Free(sb)
		kb.descSetPool = nil
	}
	if kb.imageViewPool != nil {
		kb.imageViewPool.Free(sb)
		kb.imageViewPool = nil
	}
	if kb.shaderModulePool != nil {
		kb.shaderModulePool.Free(sb)
		kb.shaderModulePool = nil
	}
	for _, p := range kb.pipelinePool {
		sb.write(sb.cb.VkDestroyPipeline(kb.dev, p, memory.Nullptr))
	}
	kb.pipelinePoolIndex = map[ipStorePipelineInfo]int{}
	kb.pipelinePool = []VkPipeline{}
	if kb.pipelineLayout != VkPipelineLayout(0) {
		sb.write(sb.cb.VkDestroyPipelineLayout(kb.dev, kb.pipelineLayout, memory.Nullptr))
		kb.pipelineLayout = VkPipelineLayout(0)
	}
	if kb.descriptorSetLayout != VkDescriptorSetLayout(0) {
		sb.write(sb.cb.VkDestroyDescriptorSetLayout(
			kb.dev, kb.descriptorSetLayout, memory.Nullptr))
		kb.descriptorSetLayout = VkDescriptorSetLayout(0)
	}
}

// BuildStoreKits takes a list of store recipes and return a list of store kits
// for priming image data stored in another storage image by imageLoad/Store
// operation.
func (kb *ipStoreKitBuilder) BuildStoreKits(sb *stateBuilder, recipes ...ipStoreRecipe) ([]ipStoreKit, error) {
	var err error
	storeCount := uint32(len(recipes))
	kits := make([]ipStoreKit, storeCount)
	// reserve and update descriptor sets
	descSetReservation, err := kb.descSetPool.ReserveDescriptorSets(sb, storeCount)
	if err != nil {
		return []ipStoreKit{}, log.Errf(sb.ctx, err, "failed at reserving %v descriptor sets", storeCount)
	}
	descSets := descSetReservation.DescriptorSets()
	if len(descSets) != len(recipes) {
		return []ipStoreKit{}, fmt.Errorf("not enough reserved descriptor sets")
	}
	for i := range kits {
		kits[i].dependentPieces = append(kits[i].dependentPieces, descSetReservation)
		des := descSets[i]
		inputView := kb.imageViewPool.getOrCreateImageView(sb, kb.nm, ipImageViewInfo{
			image:  recipes[i].inputImage,
			aspect: recipes[i].inputAspect,
			layer:  recipes[i].layer,
			level:  recipes[i].level,
		})
		outputView := kb.imageViewPool.getOrCreateImageView(sb, kb.nm, ipImageViewInfo{
			image:  recipes[i].outputImage,
			aspect: recipes[i].outputAspect,
			layer:  recipes[i].layer,
			level:  recipes[i].level,
		})
		writeDescriptorSet(sb, kb.dev, des, ipStoreInputImageBinding, 0,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
			[]VkDescriptorImageInfo{
				NewVkDescriptorImageInfo(sb.ta,
					0,                  // sampler
					inputView,          // imageView
					ipStoreImageLayout, // layout
				),
			}, []VkDescriptorBufferInfo{}, []VkBufferView{},
		)
		writeDescriptorSet(sb, kb.dev, des, ipStoreOutputImageBinding, 0,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
			[]VkDescriptorImageInfo{
				NewVkDescriptorImageInfo(sb.ta,
					0,                  // sampler
					outputView,         // imageView
					ipStoreImageLayout, // layout
				),
			}, []VkDescriptorBufferInfo{}, []VkBufferView{},
		)
		kits[i].descriptorSet = des
	}
	for i := range kits {
		inputImageObj := GetState(sb.newState).Images().Get(recipes[i].inputImage)
		outputImageObj := GetState(sb.newState).Images().Get(recipes[i].outputImage)
		// set pipeline
		kits[i].pipelineLayout = kb.pipelineLayout
		kits[i].pipeline = kb.getOrCreatePipeline(sb, ipStorePipelineInfo{
			inputFormat:    inputImageObj.Info().Fmt(),
			inputAspect:    recipes[i].inputAspect,
			outputFormat:   outputImageObj.Info().Fmt(),
			outputAspect:   recipes[i].outputAspect,
			outputType:     outputImageObj.Info().ImageType(),
			pipelineLayout: kb.pipelineLayout,
		})
		// set block offset and extent
		kits[i].storeOffsetX = uint32(recipes[i].offsetX)
		kits[i].storeOffsetY = uint32(recipes[i].offsetY)
		kits[i].storeOffsetZ = uint32(recipes[i].offsetZ)
		kits[i].storeExtentWidth = recipes[i].extentWidth
		kits[i].storeExtentHeight = recipes[i].extentHeight
		kits[i].storeExtentDepth = recipes[i].extentDepth
		// set word index (reserved for 64 bit per channel or wider formats)
		kits[i].wordIndex = recipes[i].wordIndex
		// set name
		kits[i].name = debugMarkerName(
			fmt.Sprintf("Store to img: %v, aspect: %v, layer: %v, level: %v",
				recipes[i].outputImage, recipes[i].outputAspect, recipes[i].layer, recipes[i].level))
	}
	return kits, nil
}

// ipStoreKit contains all the necessary information to start a compute pipeline
// to prime image data by imageLoad/Store operations.
type ipStoreKit struct {
	name              debugMarkerName
	storeOffsetX      uint32
	storeOffsetY      uint32
	storeOffsetZ      uint32
	wordIndex         uint32
	storeExtentWidth  uint32
	storeExtentHeight uint32
	storeExtentDepth  uint32
	descriptorSet     VkDescriptorSet
	pipeline          VkPipeline
	pipelineLayout    VkPipelineLayout
	dependentPieces   []flushablePiece
}

// BuildStoreCommands generates a queue command batch, which when being
// committed to a queue command handler, will record command buffer commands
// to bind compute pipeline and use the compute shader with imageLoad/Store
// operations to copy image data from one storage image to another storage image.
func (kit ipStoreKit) BuildStoreCommands(sb *stateBuilder) *queueCommandBatch {
	cmdBatch := newQueueCommandBatch(kit.name.String())
	cmdBatch.RecordCommandsOnCommit(func(commandBuffer VkCommandBuffer) {
		sb.write(sb.cb.VkCmdBindPipeline(
			commandBuffer,
			VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_COMPUTE,
			kit.pipeline,
		))
		sb.write(sb.cb.VkCmdBindDescriptorSets(
			commandBuffer,
			VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_COMPUTE,
			kit.pipelineLayout,
			0, 1, sb.MustAllocReadData(kit.descriptorSet).Ptr(),
			0, NewU32ᶜᵖ(memory.Nullptr),
		))
		metaData := make([]uint32, 0, 6)
		metaData = append(metaData,
			kit.storeOffsetX,
			kit.storeOffsetY,
			kit.storeOffsetZ,
			kit.wordIndex,
		)
		var db bytes.Buffer
		binary.Write(&db, binary.LittleEndian, metaData)
		sb.write(sb.cb.VkCmdPushConstants(
			commandBuffer,
			kit.pipelineLayout,
			VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
			0,
			uint32(len(db.Bytes())),
			NewCharᶜᵖ(sb.MustAllocReadData(db.Bytes()).Ptr()),
		))
		sb.write(sb.cb.VkCmdDispatch(commandBuffer,
			kit.storeExtentWidth, kit.storeExtentHeight, kit.storeExtentDepth))
	})
	cmdBatch.DoOnCommit(func(qch *queueCommandHandler) {
		qch.AddDependentFlushablePieces(kit.dependentPieces...)
	})
	return cmdBatch
}

type ipStorePipelineInfo struct {
	inputFormat    VkFormat
	inputAspect    VkImageAspectFlagBits
	outputFormat   VkFormat
	outputAspect   VkImageAspectFlagBits
	outputType     VkImageType
	pipelineLayout VkPipelineLayout
}

func (kb *ipStoreKitBuilder) getOrCreatePipeline(sb *stateBuilder, info ipStorePipelineInfo) VkPipeline {
	if i, ok := kb.pipelinePoolIndex[info]; ok {
		return kb.pipelinePool[i]
	}
	cs := kb.shaderModulePool.getOrCreateShaderModule(sb, kb.nm, ipShaderModuleInfo{
		stage:        VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT,
		inputFormat:  info.inputFormat,
		inputAspect:  info.inputAspect,
		outputFormat: info.outputFormat,
		outputAspect: info.outputAspect,
		outputType:   info.outputType,
	})

	createInfo := NewVkComputePipelineCreateInfo(sb.ta,
		VkStructureType_VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO, // sType
		0, // pNext
		0, // flags
		NewVkPipelineShaderStageCreateInfo(sb.ta, // stage
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, // sType
			0, // pNext
			0, // flags
			VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT, // stage
			cs, // module
			NewCharᶜᵖ(sb.MustAllocReadData("main").Ptr()), // pName
			NewVkSpecializationInfoᶜᵖ(memory.Nullptr),     // pSpecializationInfo
		),
		info.pipelineLayout, // layout
		0,                   // basePipelineHandle
		0,                   // basePipelineIndex
	)

	handle := VkPipeline(newUnusedID(true, func(x uint64) bool {
		return GetState(sb.newState).ComputePipelines().Contains(VkPipeline(x)) ||
			GetState(sb.oldState).ComputePipelines().Contains(VkPipeline(x))
	}))
	sb.write(sb.cb.VkCreateComputePipelines(
		kb.dev, VkPipelineCache(0), uint32(1),
		sb.MustAllocReadData(createInfo).Ptr(),
		memory.Nullptr, sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	kb.pipelinePoolIndex[info] = len(kb.pipelinePool)
	kb.pipelinePool = append(kb.pipelinePool, handle)
	return handle
}
