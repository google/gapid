import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_clone(clone):
    clone.print(standard.HEADER)
    clone.print('#include <vulkan/vk_layer.h>')
    clone.print('#include <functional>')
    clone.line()
    clone.print('#include "bind_first.h"')
    clone.print('#include "custom.h"')
    clone.print('#include "state_block.h"')
    clone.print('#include "temporary_allocator.h"')
    clone.print('namespace gapid2 {')


def output_union_clone(x, clone):
    pass


def output_member_clone(x, xdst, struct_name, memberid, idx, vop, dop, clone):
    tp = x.type
    dt = xdst.type
    while(type(tp) == vulkan.const_type):
        tp = tp.child
    while(type(dt) == vulkan.const_type):
        dt = dt.child
    if x.name == "pNext":
        if len(x.extended_by):
            clone.print(
                f"auto srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>({vop}pNext);")
            clone.print(f"void* dstBaseStructBase = nullptr;")
            clone.print(
                f"auto dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&dstBaseStructBase);")
            clone.enter_scope(f"while(srcBaseStruct) {{")
            clone.enter_scope(f"switch(srcBaseStruct->sType) {{")
            for y in x.extended_by:
                clone.enter_scope(f"case {y[0]}: {{")
                clone.print(
                    f"{y[1].name}* _val = mem->get_typed_memory<{y[1].name}>(1);")
                clone.enter_scope(
                    f"if (srcBaseStruct->pNext != nullptr) {{")
                clone.print(
                    f"auto srctmp_ = *reinterpret_cast<const {y[1].name}*>(srcBaseStruct);")
                clone.print(
                    f"srctmp_.pNext = nullptr;")
                prms = [f"srctmp_", "*_val", "mem"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, src)' for z in y[1].get_serialization_params(vulkan.CLONE)])
                clone.print(
                    f"clone(state_block_, {', '.join(prms)});")
                clone.print(
                    f"*dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);")
                clone.leave_enter_scope(f"}} else {{")
                prms = [
                    f"*reinterpret_cast<const {y[1].name}*>(srcBaseStruct)", "*_val", "mem"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, src)' for z in y[1].get_serialization_params(vulkan.CLONE)])
                clone.print(
                    f"clone(state_block_, {', '.join(prms)});")
                clone.print(
                    f"*dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);")
                clone.leave_scope(f"}}")
                clone.print(f"break;")
                clone.leave_scope(f"}}")
            if (struct_name == 'VkDeviceCreateInfo'):
                clone.enter_scope(
                    f"case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO: {{")
                clone.print(
                    f"VkLayerDeviceCreateInfo* _val = mem->get_typed_memory<VkLayerDeviceCreateInfo>(1);")
                clone.print(
                    f"memcpy(_val, srcBaseStruct, sizeof(VkLayerDeviceCreateInfo));")
                clone.print(
                    f"*dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);")
                clone.post_leave_scope(f"break;")
                clone.leave_scope(f"}}")
            clone.enter_scope(f"default:")
            clone.post_leave_scope(f'GAPID2_ERROR("Unexpected pNext");')
            clone.leave_scope(f"}}")
            clone.print(
                f"dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&((*dstBaseStruct)->pNext));")
            clone.print(
                f"srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>(srcBaseStruct->pNext);")
            clone.leave_scope(f"}}")
            clone.print(f"{dop}pNext = dstBaseStructBase;")
        elif (struct_name == "VkInstanceCreateInfo"):
            clone.print(
                f"auto srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>({vop}pNext);")
            clone.print(f"void* dstBaseStructBase = nullptr;")
            clone.print(
                f"auto dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&dstBaseStructBase);")
            clone.enter_scope(f"while(srcBaseStruct) {{")
            clone.enter_scope(f"switch(srcBaseStruct->sType) {{")
            clone.enter_scope(
                f"case VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO: {{")
            clone.print(
                f"VkLayerInstanceCreateInfo* _val = mem->get_typed_memory<VkLayerInstanceCreateInfo>(1);")
            clone.print(
                f"memcpy(_val, srcBaseStruct, sizeof(VkLayerInstanceCreateInfo));")
            clone.print(
                f"*dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);")
            clone.post_leave_scope(f"break;")
            clone.leave_scope(f"}}")
            clone.enter_scope(f"    default:")
            clone.post_leave_scope(f'GAPID2_ERROR("Unexpected pNext");')
            clone.leave_scope(f"}}")
            clone.print(
                f"dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&((*dstBaseStruct)->pNext));")
            clone.print(
                f"  srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>(srcBaseStruct->pNext);")
            clone.leave_scope(f"}}")
            clone.print(f"{dop}pNext = dstBaseStructBase;")
        else:
            clone.enter_scope(
                f'if({vop}pNext) {{')
            clone.print('GAPID2_ERROR("Unexpected pNext");')
            clone.leave_scope('}')
            clone.print(f"{dop}pNext = nullptr;")
    elif type(tp) == vulkan.union:
        if x.noautovalidity:
            clone.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(src)) {{")
        prms = [f"{vop}{x.name}{idx}", f"{dop}{xdst.name}{idx}", "mem"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, src)' for z in tp.get_serialization_params(vulkan.CLONE)])
        clone.print(
            f"_custom_clone_{tp.name}(state_block_, {', '.join(prms)});")
        if x.noautovalidity:
            clone.leave_scope(f"}}")
    elif type(tp) == vulkan.struct:
        if x.noautovalidity:
            clone.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(src)) {{")
        prms = [f"{vop}{x.name}{idx}", f"{dop}{xdst.name}{idx}", "mem"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, src)' for z in tp.get_serialization_params(vulkan.CLONE)])
        clone.print(
            f"clone(state_block_, {', '.join(prms)});")
        if x.noautovalidity:
            clone.leave_scope(f"}}")
    elif type(tp) == vulkan.basetype:
        if x.noautovalidity:
            clone.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(src)) {{")
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        clone.print(
            f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};")
        if x.noautovalidity:
            clone.leave_scope(f"}}")
    elif type(tp) == vulkan.platform_type:
        if x.noautovalidity:
            clone.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(src)) {{")
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        clone.print(
            f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};")
        if x.noautovalidity:
            clone.leave_scope(f"}}")
    elif type(tp) == vulkan.pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["src", "dst", "mem"]
            clone.print(
                f"_{struct_name}_{x.name}_clone({', '.join(prms)});")
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                vulkan.error("Expected null-terminated char list")
            # encode
            clone.enter_scope(f"if ({vop}{x.name}{idx}) {{")
            clone.print(f"uint64_t _len = strlen({vop}{x.name}{idx});")
            clone.enter_scope(f"{{")
            clone.print(f"char* __tmp = mem->get_typed_memory<char>(_len+1);")
            clone.print(f"memcpy(__tmp, {vop}{x.name}{idx}, _len+1);")
            clone.print(f"{dop}{xdst.name}{idx} = __tmp;")
            clone.leave_scope(f"}}")
            clone.leave_enter_scope(f"}} else {{")
            clone.print(f"{dop}{xdst.name}{idx} = nullptr;")
            clone.leave_scope(f"}}")
            return
        ct = "1"
        if x.noautovalidity:
            clone.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(src)) {{")
        elif x.optional:
            clone.enter_scope(f"if ({vop}{x.name}{idx}) {{")

        mem_idx = f'{idx}[0]'
        if x.len:
            ll = f"{vop}{x.len.split(',')[0]}"
            ct = ll
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                clone.print(
                    f"uint64_t _len{memberid} = _{struct_name}_{x.name}_length(src);")
                ct = f"_len{memberid}"

        tmp = f'temp{memberid}'
        clone.print(
            f"{tp.get_non_const_member(tmp)} = mem->get_typed_memory<{tp.pointee.get_non_const_member('')[0:-1]}>({ct});")

        if x.len:
            ii = clone.get_depth()
            clone.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {ct}; ++i_{ii}) {{")
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        xm.name = f'{vop}{xm.name}'
        xm.noautovalidity = False
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        x2 = copy.deepcopy(xdst)
        x2.type = dt.pointee
        x2.name = tmp
        x2.noautovalidity = False

        output_member_clone(xm, x2, struct_name, memberid +
                            1, mem_idx, f"", f"", clone)
        if x.len:
            clone.leave_scope(f"}}")
        clone.print(f"{dop}{xdst.name}{idx} = {tmp};")
        if x.optional or x.noautovalidity:
            clone.leave_enter_scope(f"}} else {{")
            clone.print(f"{dop}{xdst.name}{idx} = nullptr;")
            clone.leave_scope(f"}}")
    elif type(tp) == vulkan.enum:
        clone.print(
            f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};")
    elif type(tp) == vulkan.handle:
        if x.noautovalidity:
            clone.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(src)) {{")
        clone.print(
            f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};")
        if x.noautovalidity:
            clone.leave_scope(f"}}")

    elif type(tp) == vulkan.array_type:
        ii = clone.get_depth()
        clone.enter_scope(
            f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{")
        xm = copy.deepcopy(x)
        xm.type = tp.child
        xm.noautovalidity = False
        x2 = copy.deepcopy(xdst)
        x2.type = dt.child
        x2.noautovalidity = False
        output_member_clone(xm, x2, struct_name, memberid + 1,
                            f"[i_{ii}]", vop, dop, clone)
        clone.leave_scope(f"}}")


def output_struct_clone(x, clone):
    memberid = 0
    serialize_params = ["state_block* state_block_",
                        f"const {x.name}& src", f"{x.name}& dst", "temporary_allocator* mem"]
    serialize_params.extend([y.function()
                            for y in x.get_serialization_params(vulkan.CLONE)])

    clone.enter_scope(
        f"inline void clone({', '.join(serialize_params)}) {{")

    for y in x.members:
        output_member_clone(y, y, x.name, memberid, "",
                            "src.", "dst.", clone)
        memberid += 1
    clone.leave_scope("}\n")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "struct_clone.h"), mode="w") as fclone:
        clone = generator.generator(fclone)

        out_clone(clone)

        for x in definition.get_sorted_structs():
            if type(x) == vulkan.union:
                output_union_clone(x, clone)
            elif type(x) == vulkan.struct:
                output_struct_clone(x, clone)
        clone.leave_scope('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
