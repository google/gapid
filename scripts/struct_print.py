import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def output_print_member(x, struct_name, memberid, idx, vop, p):
    nm = f"\"{x.name}\""
    if idx != "" and idx != "[0]":
        nm = "\"\""
    tp = x.type
    while(type(tp) == vulkan.const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            p.print(f'printer->begin_array("pNext");')
            p.print(
                f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);")
            p.enter_scope(f"while(baseStruct) {{")
            p.print(f"switch(baseStruct->sType) {{")
            for y in x.extended_by:
                p.enter_scope(f"case {y[0]}:")
                p.enter_scope(f"if (baseStruct->pNext != nullptr) {{")
                p.print(
                    f"{y[1].name} _tmp = *reinterpret_cast<const {y[1].name}*>(baseStruct);")
                p.print(f"_tmp.pNext = nullptr;")
                prms = [f"_tmp", "printer"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params()])
                p.print(
                    f"print_{y[1].name}(\"\", state_block_, {', '.join(prms)});")
                p.leave_enter_scope(f"}} else {{")
                prms = [
                    f"*reinterpret_cast<const {y[1].name}*>(baseStruct)", "printer"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params()])
                p.print(
                    f"print_{y[1].name}(\"\", state_block_, {', '.join(prms)});")
                p.leave_scope(f"}}")
                p.post_leave_scope(f"break;")
            if (struct_name == 'VkDeviceCreateInfo'):
                p.enter_scope(
                    f"case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO:")
                p.post_leave_scope(f"      break;")
            p.enter_scope(f"default:")
            p.print(f'GAPID2_ERROR("Unexpected pNext");')
            p.leave_scope(f"  }}")
            p.print(f"baseStruct = baseStruct->pNext;")
            p.leave_scope(f"}}")
            p.print(f'printer->end_array();')
        else:
            p.print(
                f'if({vop}pNext) {{ GAPID2_ERROR("Unexpected pNext"); }}')
            p.print(f'printer->begin_array("pNext");')
            p.print(f'printer->end_array();')
    elif type(tp) == vulkan.union:
        prms = [f"{vop}{x.name}{idx}", "printer"]
        prms.extend([z.name() for z in tp.get_serialization_params()])
        p.print(f"// ignoring union for now")
        #p.print(f"_custom_print{tp.name}({nm}, state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.struct:
        prms = [f"{vop}{x.name}{idx}", "printer"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params()])
        p.print(f"print_{tp.name}({nm}, state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        p.print(f"printer->print({nm}, {vop}{x.name}{idx});")
    elif type(tp) == vulkan.platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        p.print(f"printer->print({nm}, {vop}{x.name}{idx});")
    elif type(tp) == vulkan.pointer_type:
        if (tp.pointee.name == "void"):
            p.print(f"// Ignoring nullptr for now")
            #prms = ["val", "printer"]
            #p.print(f"_{struct_name}_{x.name}_print({nm}, {', '.join(prms)});")
            return

        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                vulkan.error("Expected null-terminated char list")
            p.print(f"printer->print_string({nm}, {vop}{x.name}{idx});")
            return

        if x.noautovalidity:
            p.enter_scope(f"if (_{struct_name}_{x.name}_valid(val)) {{")
        elif x.optional:
            p.enter_scope(f"if ({vop}{x.name}{idx}) {{")

        mem_idx = f'{idx}[0]'
        if x.len:
            p.print(f"printer->begin_array({nm});")
            ll = f"{vop}{x.len.split(',')[0]}"
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                ll = f"_{struct_name}_{x.name}_length(val)"
            ii = p.get_depth()
            p.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {ll}; ++i_{ii}) {{")
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee

        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_print_member(xm, struct_name, memberid + 1,
                            mem_idx, f"{vop}", p)

        if x.len:
            p.leave_scope(f"}}")
            p.print(f"printer->end_array();")
        if x.noautovalidity:
            p.leave_enter_scope(f"}} else {{")
            p.print(f"printer->print({nm}, nullptr);")
            p.leave_scope(f"}}")
        elif x.optional:
            p.leave_enter_scope(f"}} else {{")
            p.print(f"printer->print({nm}, nullptr);")
            p.leave_scope(f"}}")

    elif type(tp) == vulkan.enum:
        p.print(
            f"printer->print({nm}, {vop}{x.name}{idx});")
    elif type(tp) == vulkan.handle:
        if tp.dispatch == vulkan.handle.DISPATCHABLE:
            p.print(
                f"printer->print({nm}, reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));")
        else:
            p.print(
                f"printer->print({nm}, reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));")
    elif type(tp) == vulkan.array_type:
        if tp.get_noncv_type().name == 'char':
            p.print(
                f"printer->print_char_array({nm}, {vop}{x.name}{idx}, {tp.size});")
            return
        p.print(f"printer->begin_array({nm});")
        ii = p.get_depth()
        p.enter_scope(
            f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{")
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_print_member(xm, struct_name, memberid + 1,
                            f"[i_{ii}]", vop, p)
        p.leave_scope(f"}}")
        p.print(f"printer->end_array();")


def output_print_union(x, p):
    pass


def output_print_struct(x, p):
    memberid = 0

    serialize_params = [f"const {x.name}& val", "printer* printer"]
    serialize_params.extend([y.function()
                            for y in x.get_serialization_params()])
    p.enter_scope(
        f"inline void print_{x.name}(const char* name, state_block* state_block_, {', '.join(serialize_params)}) {{")
    p.print(f"printer->begin_object(name);")

    for y in x.members:
        output_print_member(y, x.name, memberid, "", "val.", p)
        memberid += 1
    p.print(f"printer->end_object();")
    p.leave_scope("}\n")


def out_header(p):
    p.print(standard.HEADER)
    p.print('#include <functional>')
    p.line()
    p.print('#include "helpers.h"')
    p.print('#include "forwards.h"')
    p.print('#include "custom.h"')
    p.print('#include "printer.h"')
    p.print('namespace gapid2 {')


def out_footer(p):
    p.print('}')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "struct_print.h"), mode="w") as fdec:
        p = generator.generator(fdec)

        out_header(p)

        for x in definition.get_sorted_structs():
            if type(x) == vulkan.union:
                output_print_union(x, p)
            elif type(x) == vulkan.struct:
                output_print_struct(x, p)
        out_footer(p)


if __name__ == "__main__":
    main(args.get_args())
