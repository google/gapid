import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_dec(dec):
    dec.print(standard.HEADER)
    dec.print('#include <functional>')
    dec.line()
    dec.print('#include "decoder.h"')
    dec.print('#include "helpers.h"')
    dec.print('#include "forwards.h"')
    dec.print('#include "custom.h"')
    dec.print('namespace gapid2 {')


def output_union_dec(x, dec):
    pass


def output_member_dec(x, struct_name, memberid, idx, vop, dec):
    tp = x.type
    while(type(tp) == vulkan.const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            dec.print(f"uint32_t pNext{memberid} = 0;")
            dec.print(f"{vop}{x.name}{idx} = nullptr;")
            dec.print(
                f"const void** ppNext{memberid} = const_cast<const void**>(&{vop}{x.name}{idx});")
            dec.enter_scope(
                f"while ((pNext{memberid} = dec->decode<uint32_t>())) {{")

            dec.enter_scope(f"switch (pNext{memberid}) {{")
            for y in x.extended_by:
                dec.enter_scope(f"case {y[0]}: {{")
                dec.print(
                    f"{y[1].name}* pn = dec->get_typed_memory<{y[1].name}>(1);")
                prms = [f"*pn", "dec"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params(vulkan.DESERIALIZE)])
                dec.print(
                    f"deserialize_{y[1].name}(state_block_, {', '.join(prms)});")
                dec.print(f"*ppNext{memberid} = pn;")
                dec.print(
                    f"ppNext{memberid} = const_cast<const void**>(&pn->pNext);")
                dec.print(f"break;")
                dec.leave_scope(f"}}")
            if (struct_name == 'VkDeviceCreateInfo'):
                dec.enter_scope(
                    f"case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO:")
                dec.post_leave_scope(f"break;")
            dec.enter_scope(f"default:")
            dec.post_leave_scope(f'GAPID2_ERROR("Unexpected pNext");')
            dec.leave_scope(f"}}")
            dec.leave_scope(f"}}")
        else:
            dec.enter_scope('if (dec->decode<uint32_t>()) {')
            dec.print('GAPID2_ERROR("Unexpected pNext");')
            dec.leave_scope('}')
            dec.print(f'{vop}{x.name}{idx} = nullptr;')
    elif type(tp) == vulkan.union:
        dec.print(
            f"_custom_deserialize_{tp.name}(state_block_, {vop}{x.name}{idx}, dec);")
    elif type(tp) == vulkan.struct:
        prms = [f"{vop}{x.name}{idx}", "dec"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params(vulkan.DESERIALIZE)])
        dec.print(f"deserialize_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        dec.print(f"dec->decode<{enc_type}>(&{vop}{x.name}{idx});")
    elif type(tp) == vulkan.platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        dec.print(f"dec->decode<{enc_type}>(&{vop}{x.name}{idx});")
    elif type(tp) == vulkan.pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["val", "dec"]
            dec.print(
                f"_{struct_name}_{x.name}_deserialize({', '.join(prms)});")
            return
        if type(tp.pointee) == vulkan.external_type:
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                vulkan.error("Expected null-terminated char list")

            # decode
            dec.print(f"uint64_t length{memberid} = dec->decode<uint64_t>();")
            dec.enter_scope(f"if (length{memberid}) {{")
            dec.print(
                f"char* tmp_ = static_cast<char*>(dec->get_memory(length{memberid}));")
            dec.print(
                f"dec->decode_primitive_array(tmp_, length{memberid});")
            dec.print(f"{vop}{x.name}{idx} = tmp_;")
            dec.leave_enter_scope(f"}} else {{")
            dec.print(f"{vop}{x.name}{idx} = nullptr;")
            dec.leave_scope(f"}}")
            return

        ct = "1"
        if x.optional or x.noautovalidity:
            dec.enter_scope(f"if (dec->decode<char>()) {{")

        if x.len:
            ct = f'temp_len{memberid}'
            dec.print(f"uint64_t {ct} = dec->decode<uint64_t>();")

        tmp = f'temp{memberid}'
        dec.print(
            f"{tp.get_non_const_member(tmp)} = dec->get_typed_memory<{tp.pointee.get_non_const_member('')[0:-1]}>({ct});")

        mem_idx = f'[0]'
        if x.len:
            ii = dec.get_depth()
            dec.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {ct}; ++i_{ii}) {{")
            mem_idx = f'[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        xm.name = tmp
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_member_dec(xm, struct_name, memberid +
                          1, mem_idx, f"", dec)

        if x.len:
            dec.leave_scope(f"}}")
        dec.print(f"{vop}{x.name}{idx} = {tmp};")
        if x.optional or x.noautovalidity:
            dec.leave_enter_scope(f"}} else {{")
            dec.print(f"{vop}{x.name}{idx} = nullptr;")
            dec.leave_scope(f"}}")

    elif type(tp) == vulkan.enum:
        dec.print(
            f"dec->decode<{tp.base_type.name}>(&{vop}{x.name}{idx});")
    elif type(tp) == vulkan.handle:
        if tp.dispatch == vulkan.handle.DISPATCHABLE:
            dec.print(
                f"dec->decode<uint64_t>(reinterpret_cast<uintptr_t*>(&{vop}{x.name}{idx}));")
        else:
            dec.print(
                f"dec->decode<uint64_t>(reinterpret_cast<uint64_t*>(&{vop}{x.name}{idx}));")
    elif type(tp) == vulkan.array_type:
        ii = dec.get_depth()
        dec.enter_scope(
            f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{")
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_member_dec(xm, struct_name, memberid + 1,
                          f"[i_{ii}]", vop, dec)
        dec.leave_scope(f"}}")


def output_struct_dec(x, dec):

    memberid = 0

    deserialize_params = [f"{x.name}& val", "decoder* dec"]
    deserialize_params.extend([y.function()
                              for y in x.get_serialization_params(vulkan.DESERIALIZE)])

    dec.enter_scope(
        f"inline void deserialize_{x.name}(state_block* state_block_, {', '.join(deserialize_params)}) {{")
    for y in x.members:
        output_member_dec(y, x.name, memberid, "", "val.", dec)
        memberid += 1
    dec.leave_scope("}\n")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "struct_deserialization.h"), mode="w") as fdec:
        dec = generator.generator(fdec)

        out_dec(dec)

        for x in definition.get_sorted_structs():
            if type(x) == vulkan.union:
                output_union_dec(x, dec)
            elif type(x) == vulkan.struct:
                output_struct_dec(x, dec)
        dec.leave_scope('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
