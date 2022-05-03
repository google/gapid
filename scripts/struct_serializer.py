import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_enc(enc):
    enc.print(standard.HEADER)
    enc.print('#include <functional>')
    enc.line()
    enc.print('#include "bind_first.h"')
    enc.print('#include "encoder.h"')
    enc.print('#include "custom.h"')
    enc.print('namespace gapid2 {')


def output_union_enc(x, enc):
    pass


def output_member_enc(x, struct_name, memberid, idx, vop, enc):
    tp = x.type
    while(type(tp) == vulkan.const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            enc.print(
                f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);")
            enc.enter_scope(f"while (baseStruct) {{")
            enc.print(
                f"enc->template encode<uint32_t>(baseStruct->sType);")
            enc.enter_scope(f"switch (baseStruct->sType) {{")
            for y in x.extended_by:
                enc.enter_scope(f"case {y[0]}:")
                enc.enter_scope(
                    f"if (baseStruct->pNext != nullptr) {{")
                enc.print(
                    f"{y[1].name} _tmp = *reinterpret_cast<const {y[1].name}*>(baseStruct);")
                enc.print(f"_tmp.pNext = nullptr;")
                prms = [f"_tmp", "enc"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params()])
                enc.print(
                    f"serialize_{y[1].name}(state_block_, {', '.join(prms)});")
                enc.leave_enter_scope(f"}} else {{")
                prms = [
                    f"*reinterpret_cast<const {y[1].name}*>(baseStruct)", "enc"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params()])
                enc.print(
                    f"serialize_{y[1].name}(state_block_, {', '.join(prms)});")
                enc.leave_scope(f"}}")
                enc.post_leave_scope(f"break;")
            if (struct_name == 'VkDeviceCreateInfo'):
                enc.enter_scope(
                    f"case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO:")
                enc.post_leave_scope(f"break;")
            enc.enter_scope(f"default:")
            enc.post_leave_scope(f'GAPID2_ERROR("Unexpected pNext");')
            enc.leave_scope(f"}}")
            enc.print(f"baseStruct = baseStruct->pNext;")
            enc.leave_scope(f"}}")
            enc.print(
                f"enc->template encode<uint32_t>(0);  // No more pNext")
        elif (struct_name == "VkInstanceCreateInfo"):
            enc.print(
                f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);")
            enc.enter_scope(f"while (baseStruct) {{")
            enc.enter_scope(f"switch (baseStruct->sType) {{")
            enc.enter_scope(
                f"case VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO:")
            enc.post_leave_scope(f"break;")
            enc.enter_scope(f"default:")
            enc.post_leave_scope(f'GAPID2_ERROR("Unexpected pNext");')
            enc.leave_scope(f"}}")
            enc.print(f"baseStruct = baseStruct->pNext;")
            enc.leave_scope(f"}}")
            enc.print(
                f"enc->template encode<uint32_t>(0);  // No more pNext")
        else:
            enc.enter_scope(f'if ({vop}pNext) {{')
            enc.print('GAPID2_ERROR("Unexpected pNext");')
            enc.leave_scope('}')
            enc.print(f"enc->template encode<uint32_t>(0);  // pNext")
    elif type(tp) == vulkan.union:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params()])
        enc.print(
            f"_custom_serialize_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.struct:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params()])
        enc.print(
            f"serialize_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        enc.print(
            f"enc->template encode<{enc_type}>({vop}{x.name}{idx});")
    elif type(tp) == vulkan.platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        enc.print(
            f"enc->template encode<{enc_type}>({vop}{x.name}{idx});")
    elif type(tp) == vulkan.pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["val", "enc"]
            enc.print(
                f"_{struct_name}_{x.name}_serialize({', '.join(prms)});")
            return
        if type(tp.pointee) == vulkan.external_type:
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                vulkan.error("Expected null-terminated char list")
            # encode
            enc.enter_scope(f"if ({vop}{x.name}{idx}) {{")
            enc.print(
                f"uint64_t len = strlen({vop}{x.name}{idx});")
            enc.print(f"enc->template encode<uint64_t>(len + 1);")
            enc.print(
                f"enc->template encode_primitive_array<char>({vop}{x.name}{idx}, len + 1);")
            enc.leave_enter_scope(f"}} else {{")
            enc.print(f"enc->template encode<uint64_t>(0);")
            enc.leave_scope(f"}}")
            return

        if x.noautovalidity:
            enc.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(val)) {{")
            enc.print(f"enc->template encode<char>(1);")
        elif x.optional:
            enc.enter_scope(f"if ({vop}{x.name}{idx}) {{")
            enc.print(f"enc->template encode<char>(1);")

        mem_idx = f'{idx}[0]'
        if x.len:
            ll = f"{vop}{x.len.split(',')[0]}"
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                ll = f"_{struct_name}_{x.name}_length(val)"
            enc.print(
                f"enc->template encode<uint64_t>({ll});  // array_len")
            ii = enc.get_depth()
            enc.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {ll}; ++i_{ii}) {{")
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee

        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_member_enc(xm, struct_name, memberid + 1,
                          mem_idx, f"{vop}", enc)

        if x.len:
            enc.leave_scope(f"}}")
        if x.noautovalidity:
            enc.leave_enter_scope(f"}} else {{")
            enc.print(f"enc->template encode<char>(0);")
            enc.leave_scope(f"}}")
        elif x.optional:
            enc.leave_enter_scope(f"}} else {{")
            enc.print(f"enc->template encode<char>(0);")
            enc.leave_scope(f"}}")

    elif type(tp) == vulkan.enum:
        enc.print(
            f"enc->template encode<{tp.base_type.name}>({vop}{x.name}{idx});")
    elif type(tp) == vulkan.handle:
        if tp.dispatch == vulkan.handle.DISPATCHABLE:
            enc.print(
                f"enc->template encode<uint64_t>(reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));")
        else:
            enc.print(
                f"enc->template encode<uint64_t>(reinterpret_cast<uint64_t>({vop}{x.name}{idx}));")
    elif type(tp) == vulkan.array_type:
        ii = enc.get_depth()
        enc.enter_scope(
            f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{")
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_member_enc(xm, struct_name, memberid + 1,
                          f"[i_{ii}]", vop, enc)
        enc.leave_scope(f"}}")
    

def output_struct_enc(x, enc):

    memberid = 0

    serialize_params = [f"const {x.name}& val", "encoder* enc"]
    serialize_params.extend([y.function()
                            for y in x.get_serialization_params()])
    enc.enter_scope(
        f"inline void serialize_{x.name}(state_block* state_block_, {', '.join(serialize_params)}) {{")

    for y in x.members:
        output_member_enc(y, x.name, memberid, "", "val.", enc)
        memberid += 1
    enc.leave_scope("}\n")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "struct_serialization.h"), mode="w") as fenc:
        enc = generator.generator(fenc)

        out_enc(enc)

        for x in definition.get_sorted_structs():
            if type(x) == vulkan.union:
                output_union_enc(x, enc)
            elif type(x) == vulkan.struct:
                output_struct_enc(x, enc)
        enc.leave_scope('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
