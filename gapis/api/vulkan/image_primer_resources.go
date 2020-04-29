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

import "github.com/google/gapid/gapis/memory"

// descriptorSetPoolReservation is the result of descriptor set reservation
// request to homoDescriptorSetPool. It contains the descriptor sets reserved
// for the requester to use. It implements the flushablePiece interface.
type descriptorSetPoolReservation struct {
	valid    bool
	descSets []VkDescriptorSet
	owner    flushableResource
}

// IsValid implements the flushablePiece interface, to show if the descriptor
// set reserved in this reservation are still valid to use.
func (res *descriptorSetPoolReservation) IsValid() bool {
	return res.valid
}

// Owner implements the flushablePiece interface, returns the descriptor set
// pool from where this reservation was made.
func (res *descriptorSetPoolReservation) Owner() flushableResource {
	return res.owner
}

// DescriptorSets returns a list of reserved descriptor sets.
func (res *descriptorSetPoolReservation) DescriptorSets() []VkDescriptorSet {
	return res.descSets
}

// homoDescriptorSetPool is a pool of homogeneous descriptor sets allocated with
// exact same layout and from same descriptor pool. It implements the
// flushableResource interface so all the descriptor sets reserved from it have
// their life time managed by this descriptor set pool.
type homoDescriptorSetPool struct {
	name              debugMarkerName
	dev               VkDevice
	layout            VkDescriptorSetLayout
	next              uint32
	capacity          uint32
	noFlushUntilFree  bool
	pools             []VkDescriptorPool
	descSets          []VkDescriptorSet
	users             map[flushableResourceUser]struct{}
	validReservations []*descriptorSetPoolReservation
}

// newHomoDescriptorSetPool creates a new homoDescriptorPool for the given
// descriptor set layout, and a pool with the given initial count of descriptor
// sets. If notFlushUntilFree is true, the pool will not invalidate any
// descriptor sets reserved from this pool, otherwise, only the descriptor sets
// reserved in the last reservation request is guanranteed to be valid to use,
// a new incoming reservation request may trigger a flush and invalidate
// previously reserved descriptor sets.
func newHomoDescriptorSetPool(sb *stateBuilder, nm debugMarkerName, dev VkDevice, layout VkDescriptorSetLayout, initialSetCount uint32, noFlushUntilFree bool) *homoDescriptorSetPool {
	p := &homoDescriptorSetPool{
		name:              nm,
		dev:               dev,
		layout:            layout,
		next:              0,
		capacity:          initialSetCount,
		noFlushUntilFree:  noFlushUntilFree,
		pools:             []VkDescriptorPool{},
		descSets:          []VkDescriptorSet{},
		users:             map[flushableResourceUser]struct{}{},
		validReservations: []*descriptorSetPoolReservation{},
	}
	p.pools = append(p.pools, p.createDescriptorPool(sb, p.capacity))
	p.descSets = append(p.descSets, p.allocateDescriptorSets(sb, p.pools[0], p.capacity)...)
	return p
}

// ReserveDescriptorSets reserves descriptor sets from this
// homoDescriptorSetPool for a specific count, returns a reservation that
// contains the reserved descriptor sets, and error. If the
// homoDescriptorSetPool was created with noFlushUntilFree set to false, call
// to this function may trigger a flush and all the previously reserved
// descriptor sets might be flushed.
func (dss *homoDescriptorSetPool) ReserveDescriptorSets(sb *stateBuilder, count uint32) (*descriptorSetPoolReservation, error) {
	if count+dss.next > dss.capacity {
		if dss.noFlushUntilFree {
			newDescSetCount := count + dss.next - dss.capacity
			pool := dss.createDescriptorPool(sb, newDescSetCount)
			newDescSets := dss.allocateDescriptorSets(sb, pool, newDescSetCount)
			dss.pools = append(dss.pools, pool)
			dss.descSets = append(dss.descSets, newDescSets...)
			dss.capacity += newDescSetCount
		} else {
			if count > dss.capacity {
				dss.capacity = count
			}
			dss.flush(sb)
		}
		return dss.ReserveDescriptorSets(sb, count)
	}
	current := dss.next
	dss.next += count
	reservation := &descriptorSetPoolReservation{
		valid:    true,
		descSets: dss.descSets[current : current+count],
		owner:    dss,
	}
	return reservation, nil
}

func (dss *homoDescriptorSetPool) createDescriptorPool(sb *stateBuilder, setCount uint32) VkDescriptorPool {
	layoutObj := GetState(sb.newState).DescriptorSetLayouts().Get(dss.layout)
	countPerType := map[VkDescriptorType]uint32{}
	for _, info := range layoutObj.Bindings().All() {
		countPerType[info.Type()] += info.Count()
	}
	for t := range countPerType {
		countPerType[t] *= setCount
	}
	poolSizes := []VkDescriptorPoolSize{}
	for t, c := range countPerType {
		poolSizes = append(poolSizes, NewVkDescriptorPoolSize(sb.ta, t, c))
	}

	handle := VkDescriptorPool(newUnusedID(true, func(x uint64) bool {
		return GetState(sb.newState).DescriptorPools().Contains(VkDescriptorPool(x)) || GetState(sb.oldState).DescriptorPools().Contains(VkDescriptorPool(x))
	}))
	sb.write(sb.cb.VkCreateDescriptorPool(
		dss.dev,
		sb.MustAllocReadData(NewVkDescriptorPoolCreateInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO, // sType
			0, // pNext
			VkDescriptorPoolCreateFlags(
				VkDescriptorPoolCreateFlagBits_VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT), // flags
			setCount,               // maxSets
			uint32(len(poolSizes)), // poolSizeCount
			NewVkDescriptorPoolSizeᶜᵖ(sb.MustAllocReadData(poolSizes).Ptr()), // pPoolSizes
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	if len(dss.name) > 0 {
		attachDebugMarkerName(sb, dss.name, dss.dev, handle)
	}
	return handle
}

func (dss *homoDescriptorSetPool) destroyAllDescriptorPool(sb *stateBuilder) {
	for _, pool := range dss.pools {
		sb.write(sb.cb.VkDestroyDescriptorPool(dss.dev, pool, memory.Nullptr))
	}
	dss.pools = []VkDescriptorPool{}
}

func (dss *homoDescriptorSetPool) allocateDescriptorSets(sb *stateBuilder, pool VkDescriptorPool, count uint32) []VkDescriptorSet {
	newSets := make([]VkDescriptorSet, count)
	for i := range newSets {
		newSets[i] = VkDescriptorSet(
			newUnusedID(true, func(x uint64) bool {
				return GetState(sb.newState).DescriptorSets().Contains(VkDescriptorSet(x))
			}))
	}
	layoutSlice := make([]VkDescriptorSetLayout, count)
	for i := range layoutSlice {
		layoutSlice[i] = dss.layout
	}
	sb.write(sb.cb.VkAllocateDescriptorSets(
		dss.dev,
		sb.MustAllocReadData(NewVkDescriptorSetAllocateInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO, // sType
			0,     // pNext
			pool,  // descriptorPool
			count, // descriptorSetCount
			NewVkDescriptorSetLayoutᶜᵖ(sb.MustAllocReadData(layoutSlice).Ptr()), // pSetLayouts
		)).Ptr(),
		sb.MustAllocWriteData(newSets).Ptr(),
		VkResult_VK_SUCCESS,
	))
	if len(dss.name) > 0 {
		for _, set := range newSets {
			attachDebugMarkerName(sb, dss.name, dss.dev, set)
		}
	}
	return newSets
}

// flush implements the flushableResource interface.
func (dss *homoDescriptorSetPool) flush(sb *stateBuilder) {
	for u := range dss.users {
		u.OnResourceFlush(sb, dss)
		delete(dss.users, u)
	}
	for _, res := range dss.validReservations {
		res.valid = false
	}
	dss.validReservations = []*descriptorSetPoolReservation{}
	dss.destroyAllDescriptorPool(sb)
	if dss.layout != VkDescriptorSetLayout(0) {
		pool := dss.createDescriptorPool(sb, dss.capacity)
		dss.pools = append(dss.pools, pool)
		dss.descSets = dss.allocateDescriptorSets(sb, pool, dss.capacity)
	}
	dss.next = 0
}

// AddUser implements the flushableResource interface. It registers a user of
// the descriptor sets reserved from this pool.
func (dss *homoDescriptorSetPool) AddUser(user flushableResourceUser) {
	dss.users[user] = struct{}{}
}

// DropUser implements the flushableResource interface. It removes a user from
// the user list of this pool.
func (dss *homoDescriptorSetPool) DropUser(user flushableResourceUser) {
	if _, ok := dss.users[user]; ok {
		delete(dss.users, user)
	}
}

// Free flushes all the descriptor sets reserved in this pool.
func (dss *homoDescriptorSetPool) Free(sb *stateBuilder) {
	dss.layout = VkDescriptorSetLayout(0)
	dss.flush(sb)
	dss.descSets = nil
}

// naiveImageViewPool is a simple map based pool of VkImageView.
type naiveImageViewPool struct {
	dev VkDevice
	// Use a pair of map + slice to free them in order
	viewsIndex map[ipImageViewInfo]int
	views      []VkImageView
}

func newNaiveImageViewPool(dev VkDevice) *naiveImageViewPool {
	return &naiveImageViewPool{
		dev:        dev,
		viewsIndex: map[ipImageViewInfo]int{},
		views:      []VkImageView{},
	}
}

func (p *naiveImageViewPool) getOrCreateImageView(sb *stateBuilder, nm debugMarkerName, info ipImageViewInfo) VkImageView {
	if i, ok := p.viewsIndex[info]; ok {
		return p.views[i]
	}
	handle := ipCreateImageView(sb, nm, p.dev, info)
	p.viewsIndex[info] = len(p.views)
	p.views = append(p.views, handle)
	return handle
}

// Free destroyes all the image views in this pool.
func (p *naiveImageViewPool) Free(sb *stateBuilder) {
	for _, v := range p.views {
		sb.write(sb.cb.VkDestroyImageView(p.dev, v, memory.Nullptr))
	}
	p.viewsIndex = map[ipImageViewInfo]int{}
	p.views = []VkImageView{}
	return
}

// naiveShaderModulePool is a simple map based pool of VkShaderModule
type naiveShaderModulePool struct {
	dev     VkDevice
	shaders map[ipShaderModuleInfo]VkShaderModule
}

func newNaiveShaderModulePool(dev VkDevice) *naiveShaderModulePool {
	return &naiveShaderModulePool{
		dev:     dev,
		shaders: map[ipShaderModuleInfo]VkShaderModule{},
	}
}

func (p *naiveShaderModulePool) getOrCreateShaderModule(sb *stateBuilder, nm debugMarkerName, info ipShaderModuleInfo) VkShaderModule {
	if s, ok := p.shaders[info]; ok {
		return s
	}
	handle, err := ipCreateShaderModule(sb, nm, p.dev, info)
	if err != nil {
		panic(err)
	}
	p.shaders[info] = handle
	return handle
}

// Free destroyes all the shader modules in this pool
func (p *naiveShaderModulePool) Free(sb *stateBuilder) {
	for _, s := range p.shaders {
		sb.write(sb.cb.VkDestroyShaderModule(p.dev, s, memory.Nullptr))
	}
	p.shaders = map[ipShaderModuleInfo]VkShaderModule{}
	return
}
