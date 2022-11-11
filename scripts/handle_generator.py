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


def has_return_pointer(cmd):
    for x in cmd.args:
        if is_return_pointer(x):
            return True
    return False


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
        g.enter_scope(
            f"for (size_t {idx} = 0; {arg.name} && ({idx} < {ct}); ++{idx}) {{")

    if type(bt) != vulkan.handle:
        g.print(
            f'// Special case for {arg.name}[{idx}] because it just has to be unique')
        g.print(
            f'custom_generate_{arg.name}(state_block_, &{arg.name}[{idx}]);')
    else:
        g.enter_scope(f'if ({arg.name}[{idx}] == VK_NULL_HANDLE) {{')
        g.print(
            f'{arg.name}[{idx}] = state_block_->get_unused_{bt.short_str()}();')
        g.leave_scope("}")

    if arg.len:
        g.leave_scope("}")
    return True


def fix(cmd, defintion, g):

    arg_idx = 0
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
    if cmd.ret.name != "void":
        g.print(f'return ret;')


def output_header(definition, g):
    g.print(standard.HEADER)
    g.print('#include "transform_base.h"')
    g.print("namespace gapid2 {")
    g.enter_scope("class handle_generator : public transform_base {")
    g.print_scoping("public:")

    for cmd in definition.commands.values():
        if has_return_pointer(cmd):
            prms = [x.short_str() for x in cmd.args]
            g.print(f'{cmd.short_str()};')
    g.line()
    g.leave_scope("};")
    g.print("}")


def output_cpp(definition, g):
    g.print(standard.HEADER)
    g.print('#include "handle_generator.h"')
    g.print('#include "custom.h"')
    g.print('#include "state_block.h"')
    g.print('namespace gapid2 {')

    for cmd in definition.commands.values():
        if has_return_pointer(cmd):
            prms = [x.short_str() for x in cmd.args]
            g.line()
            g.enter_scope(
                f'{cmd.ret.short_str()} handle_generator::{cmd.name}({", ".join(prms)}) {{')
            fix(cmd, definition, g)
            g.leave_scope('}')
    g.print("}")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "handle_generator.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_header(definition, g)

    with open(os.path.join(args.output_location, "handle_generator.cpp"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_cpp(definition, g)


if __name__ == "__main__":
    main(args.get_args())
