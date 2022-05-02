from re import L
import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_fix_handles(enc):
    enc.print(standard.HEADER)
    enc.print('#include <functional>')
    enc.line()
    enc.print('#include "bind_first.h"')
    enc.print('#include "custom.h"')
    enc.print('#include "handle_fixer.h"')
    enc.print('namespace gapid2 {')


def output_union_fix_handles(x, fix_handles):
    pass


def output_member_fix_handles(x, struct_name, memberid, idx, vop, fix_handles):
    tp = x.type
    while(type(tp) == vulkan.const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(list(filter(lambda x: x[1].has_handle(), x.extended_by))):
            fix_handles.print(
                f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);")
            fix_handles.enter_scope(f"while (baseStruct) {{")
            fix_handles.enter_scope(f"switch (baseStruct->sType) {{")
            for y in x.extended_by:
                if not y[1].has_handle():
                    continue
                fix_handles.enter_scope(f"case {y[0]}:")
                fix_handles.enter_scope(
                    f"if (baseStruct->pNext != nullptr) {{")
                fix_handles.print(
                    f"{y[1].name} _tmp = *reinterpret_cast<const {y[1].name}*>(baseStruct);")
                fix_handles.print(f"_tmp.pNext = nullptr;")
                prms = [f"_tmp", "fix"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params(vulkan.FIX_HANDLES)])
                fix_handles.print(
                    f"fix_handles_{y[1].name}(state_block_, {', '.join(prms)});")
                fix_handles.leave_enter_scope(f"}} else {{")
                prms = [
                    f"*reinterpret_cast<const {y[1].name}*>(baseStruct)", "fix"]
                prms.extend(
                    [f'bind_first(_{struct_name}{z.name()}, val)' for z in y[1].get_serialization_params(vulkan.FIX_HANDLES)])
                fix_handles.print(
                    f"fix_handles_{y[1].name}(state_block_, {', '.join(prms)});")
                fix_handles.leave_scope(f"}}")
                fix_handles.post_leave_scope(f"break;")
            fix_handles.enter_scope(f"default:")
            fix_handles.post_leave_scope(f'break;')
            fix_handles.leave_scope(f"}}")
            fix_handles.print(f"baseStruct = baseStruct->pNext;")
            fix_handles.leave_scope(f"}}")
    elif not tp.has_handle():
        return
    elif type(tp) == vulkan.union:
        prms = [f"{vop}{x.name}{idx}", "fix"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params(vulkan.FIX_HANDLES)])
        fix_handles.print(
            f"_custom_fix_handles_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.struct:
        prms = [f"{vop}{x.name}{idx}", "fix"]
        prms.extend(
            [f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params(vulkan.FIX_HANDLES)])
        fix_handles.print(
            f"fix_handles_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == vulkan.pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["val", "fix"]
            fix_handles.print(
                f"_{struct_name}_{x.name}_fix_handles({', '.join(prms)});")
            return

        if x.len and x.len == 'null-terminated':
            return

        if x.noautovalidity:
            fix_handles.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(val)) {{")
        elif x.optional:
            fix_handles.enter_scope(f"if ({vop}{x.name}{idx}) {{")

        mem_idx = f'{idx}[0]'
        if x.len:
            ll = f"{vop}{x.len.split(',')[0]}"
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                ll = f"_{struct_name}_{x.name}_length(val)"
            ii = fix_handles.get_depth()
            fix_handles.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {ll}; ++i_{ii}) {{")
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee

        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_member_fix_handles(xm, struct_name, memberid + 1,
                                  mem_idx, f"{vop}", fix_handles)

        if x.len:
            fix_handles.leave_scope(f"}}")
        if x.noautovalidity:
            fix_handles.leave_scope(f"}}")
        elif x.optional:
            fix_handles.leave_scope(f"}}")
    elif type(tp) == vulkan.handle:
        if x.noautovalidity:
            fix_handles.enter_scope(
                f"if (_{struct_name}_{x.name}_valid(val)) {{")
        fix_handles.print(f"fix->fix_handle(&{vop}{x.name}{idx});")
        if x.noautovalidity:
            fix_handles.leave_scope("}")
    elif type(tp) == vulkan.array_type:
        ii = fix_handles.get_depth()
        fix_handles.enter_scope(
            f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{")
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_member_fix_handles(xm, struct_name, memberid + 1,
                                  f"[i_{ii}]", vop, fix_handles)
        fix_handles.leave_scope(f"}}")


def output_struct_fix_handles(x, fix_handles):

    memberid = 0

    serialize_params = [f"const {x.name}& val", "handle_fixer* fix"]
    serialize_params.extend([y.function()
                            for y in x.get_serialization_params(vulkan.FIX_HANDLES)])
    fix_handles.enter_scope(
        f"inline void fix_handles_{x.name}(state_block* state_block_, {', '.join(serialize_params)}) {{")

    for y in x.members:
        output_member_fix_handles(y, x.name, memberid, "", "val.", fix_handles)
        memberid += 1
    fix_handles.leave_scope("}\n")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "fix_handles.h"), mode="w") as fenc:
        enc = generator.generator(fenc)

        out_fix_handles(enc)

        for x in definition.get_sorted_structs():
            if not x.has_handle():
                continue

            if type(x) == vulkan.union:
                output_union_fix_handles(x, enc)
            elif type(x) == vulkan.struct:
                output_struct_fix_handles(x, enc)
        enc.leave_scope('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
