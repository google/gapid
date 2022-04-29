import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os


def is_return_pointer(arg):
    tp = arg.type
    return type(tp) == vulkan.pointer_type and not tp.const and tp.has_handle()


def register_return_pointer(cmd, arg, arg_idx, defintion, g):
    if not is_return_pointer(arg):
        return False
    bt = arg.type.get_noncv_type()

    idx = "0"
    if arg.len:
        ct = f"{arg.len.split(',')[0]}"
        p = [x for x in cmd.args if x.name == ct]
        if len(p):
            if type(p[0].type) == vulkan.pointer_type:
                ct = f"*{ct}"

        idx = f'i_{arg_idx}'
        g.print("// If the arg is nullptr then we dont register anything")
        g.enter_scope(
            f"for (size_t {idx} = 0; {arg.name} && ({idx} < {ct}); ++{idx}) {{")

    if type(bt) != vulkan.handle:
        g.print(
            f'// Special case for {arg.name}[{idx}] because it just has to be unique')
        g.print(f'custom_register_{arg.name}(&{arg.name}[{idx}], fix_);')
    else:
        g.print(f'fix_.register_handle(&{arg.name}[{idx}]);')

    if arg.len:
        g.leave_scope("}")
    return True


def process_return_pointer(cmd, arg, arg_idx, definition, g):
    if not is_return_pointer(arg):
        return False
    bt = arg.type.get_noncv_type()

    idx = "0"
    if arg.len:
        ct = f"{arg.len.split(',')[0]}"
        p = [x for x in cmd.args if x.name == ct]
        if len(p):
            if type(p[0].type) == vulkan.pointer_type:
                ct = f"*{ct}"

        idx = f'i_{arg_idx}'
        g.print("// If the arg is nullptr then we dont register anything")
        g.enter_scope(
            f"for (size_t {idx} = 0; {arg.name} && ({idx} < {ct}); ++{idx}) {{")

    if type(bt) != vulkan.handle:
        g.print(
            f'// Special case for {arg.name} because it just has to be unique')
        g.print(f'custom_process_{arg.name}(&{arg.name}[{idx}], fix_);')
    else:
        g.print(f'fix_.process_handle(&{arg.name}[{idx}]);')

    if arg.len:
        g.leave_scope("}")
    return True


def output_arg(cmd, arg, arg_idx, defintion, g):
    if is_return_pointer(arg):
        g.print(f"// {arg.name} Handled below as it is a return pointer")
        return

    tp = arg.type
    const = type(tp) == vulkan.const_type or (
        type(tp) == vulkan.pointer_type and tp.const)
    while(type(tp) == vulkan.const_type):
        tp = tp.child
    if not tp.has_handle():
        return
    if type(tp) == vulkan.pointer_type and not const:
        # Non constant pointer. This is a return value
        return

    if type(tp) == vulkan.handle:
        g.print(f"fix_.fix_handle(&{arg.name});")
        # Fixup individual handle
        return
    if type(tp) != vulkan.pointer_type:
        vulkan.error(
            f"Not a pointer, not a handle, no idea what this is ${tp}")

    idx = "0"
    if arg.len:
        ct = f"{arg.len.split(',')[0]}"
        p = [x for x in cmd.args if x.name == ct]
        if len(p):
            if type(p[0].type) == vulkan.pointer_type:
                ct = f"*{ct}"

        idx = f'i_{arg_idx}'
        g.enter_scope(
            f"for (size_t {idx} = 0; {idx} < {ct}; ++{idx}) {{")

    bt = tp.get_noncv_type()
    if (type(bt) == vulkan.handle):
        g.print(f"fix_.fix_handle(&{arg.name}[{idx}]);")
    else:
        prms = ['state_block_', f'{arg.name}[{idx}]', '&fix_']
        prms.extend([x.name()
                    for x in bt.get_serialization_params(vulkan.FIX_HANDLES)])
        g.print(
            f"fix_handles_{bt.name}({', '.join(prms)});")
    if arg.len:
        g.leave_scope("}")


def fix(cmd, defintion, g):

    arg_idx = 0
    for arg in cmd.args:
        output_arg(cmd, arg, arg_idx, defintion, g)
        arg_idx += 1
    has_return = False
    for arg in cmd.args:
        has_return |= register_return_pointer(cmd, arg, arg_idx, defintion, g)
        arg_idx += 1

    if cmd.ret.name != "void":
        g.print(
            f"auto ret = transform_base::{cmd.name}({', '.join([x.name for x in cmd.args])});")
    else:
        g.print(
            f"transform_base::{cmd.name}({', '.join([x.name for x in cmd.args])});")
    for arg in cmd.args:
        process_return_pointer(cmd, arg, arg_idx, defintion, g)
        arg_idx += 1

    if has_return:
        g.print("// Sanity check")
        g.print("fix_.ensure_clean();")
    if cmd.ret.name != "void":
        g.print(f'return ret;')


def output_header(definition, g):
    g.print(standard.HEADER)
    g.print('#include "encoder.h"')
    g.print('#include "transform_base.h"')
    g.print('#include "handle_fixer.h"')
    g.print("namespace gapid2 {")
    g.enter_scope("class command_inline_fixer : public transform_base {")
    g.print_scoping("public:")

    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.print(f'{cmd.short_str()};')
    g.line()
    g.print("handle_fixer fix_;")
    g.leave_scope("};")
    g.print("}")


def output_cpp(definition, g):
    g.print(standard.HEADER)
    g.print('#include "command_inline_fixer.h"')
    g.print('#include "fix_handles.h"')
    g.print('#include "forwards.h"')
    g.print('namespace gapid2 {')

    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.line()
        g.enter_scope(
            f'{cmd.ret.short_str()} command_inline_fixer::{cmd.name}({", ".join(prms)}) {{')
        fix(cmd, definition, g)
        g.leave_scope('}')
    g.print("}")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_inline_fixer.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_header(definition, g)

    with open(os.path.join(args.output_location, "command_inline_fixer.cpp"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_cpp(definition, g)


if __name__ == "__main__":
    main(args.get_args())
