import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os


def output_recorder(definition, g):
    g.print(standard.COPYRIGHT)

    for cmd in definition.commands.values():
        if cmd.args[0].name != "commandBuffer":
            continue
        prms = [x.short_str() for x in cmd.args]
        g.print(f'{cmd.short_str()};')


def output_cpp(definition, g):
    g.print(standard.CPPHEADER)
    g.print('#include "command_buffer_recorder.h"')
    g.print('#include "struct_serialization.h"')
    g.print('#include "forwards.h"')
    g.print('namespace gapid2 {')

    for cmd in definition.commands.values():
        if cmd.args[0].name != "commandBuffer":
            continue
        prms = [x.short_str() for x in cmd.args]
        g.line()
        g.enter_scope(
            f'{cmd.ret.short_str()} command_buffer_recorder::{cmd.name}({", ".join(prms)}) {{')
        if cmd.name == "vkBeginCommandBuffer":
            g.print(f'do_begin_command_buffer(commandBuffer);')
        arg_serialization.output_command(
            cmd, definition, g, False, True, False)
        g.leave_scope('}')
    g.print("}")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_buffer_recorder.inl"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_recorder(definition, g)

    with open(os.path.join(args.output_location, "command_buffer_recorder.cpp"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_cpp(definition, g)


if __name__ == "__main__":
    main(args.get_args())
