package vulkan

import "github.com/google/gapid/gapis/memory"

type descriptorSetPoolReservation struct {
	valid    bool
	descSets []VkDescriptorSet
	owner    flushableResource
}

func (res *descriptorSetPoolReservation) IsValid() bool {
	return res.valid
}

func (res *descriptorSetPoolReservation) Owner() flushableResource {
	return res.owner
}

func (res *descriptorSetPoolReservation) DescriptorSets() []VkDescriptorSet {
	return res.descSets
}

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

type descriptorSetLayoutBindingInfo struct {
	descriptorType VkDescriptorType
	count          uint32
	stages         VkShaderStageFlags
}

type descriptorSetLayoutInfo struct {
	bindings map[uint32]descriptorSetLayoutBindingInfo
}

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

func (dss *homoDescriptorSetPool) AddUser(user flushableResourceUser) {
	dss.users[user] = struct{}{}
}

func (dss *homoDescriptorSetPool) DropUser(user flushableResourceUser) {
	if _, ok := dss.users[user]; ok {
		delete(dss.users, user)
	}
}

func (dss *homoDescriptorSetPool) Free(sb *stateBuilder) {
	dss.layout = VkDescriptorSetLayout(0)
	dss.flush(sb)
	dss.descSets = nil
}

type naiveImageViewPool struct {
	dev   VkDevice
	views map[ipImageViewInfo]VkImageView
}

func newNaiveImageViewPool(dev VkDevice) *naiveImageViewPool {
	return &naiveImageViewPool{
		dev:   dev,
		views: map[ipImageViewInfo]VkImageView{},
	}
}

func (p *naiveImageViewPool) getOrCreateImageView(sb *stateBuilder, nm debugMarkerName, info ipImageViewInfo) VkImageView {
	if v, ok := p.views[info]; ok {
		return v
	}
	handle := ipCreateImageView(sb, nm, p.dev, info)
	p.views[info] = handle
	return handle
}

func (p *naiveImageViewPool) Free(sb *stateBuilder) {
	for _, v := range p.views {
		sb.write(sb.cb.VkDestroyImageView(p.dev, v, memory.Nullptr))
	}
	p.views = map[ipImageViewInfo]VkImageView{}
	return
}

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

func (p *naiveShaderModulePool) Free(sb *stateBuilder) {
	for _, s := range p.shaders {
		sb.write(sb.cb.VkDestroyShaderModule(p.dev, s, memory.Nullptr))
	}
	p.shaders = map[ipShaderModuleInfo]VkShaderModule{}
	return
}
