import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os


def output_header(definition, g):
    g.print(standard.HEADER)
    g.print('#include "transform_base.h"')
    g.print('namespace gapid2 {')
    g.enter_scope('struct handle_replacer : public transform_base {')

    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.print(f'{cmd.short_str()};')
    g.leave_scope('};')
    g.print('}')


def output_cpp(definition, g):
    g.print(standard.HEADER)
    g.print('#include "handle_replacer.h"')
    g.print('namespace gapid2 {')
    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.line()
        g.enter_scope(
            f'{cmd.ret.short_str()} handle_replacer::{cmd.name}({", ".join(prms)}) {{')
        if cmd.ret.name != "void":
            g.print(
                f'auto ret = transform_base::{cmd.name}({", ".join([x.name for x in cmd.args])}); ')
        else:
            g.print(
                f'transform_base::{cmd.name}({", ".join([x.name for x in cmd.args])}); ')

        if cmd.ret.name != "void":
            g.print("return ret;")
        g.leave_scope('}')
    g.print('}')
    pass


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "handle_replacer.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_header(definition, g)

    with open(os.path.join(args.output_location, "handle_replacer.cpp"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_cpp(definition, g)


if __name__ == "__main__":
    main(args.get_args())
