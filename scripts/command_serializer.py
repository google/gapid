import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os


def output_recorder(definition, g):
    g.print(standard.HEADER)
    g.print('#include "encoder.h"')
    g.print('#include "transform_base.h"')
    g.print("namespace gapid2 {")
    g.enter_scope("class command_serializer : public transform_base {")
    g.print_scoping("public:")

    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.print(f'{cmd.short_str()};')
    g.line()
    g.print("virtual encoder_handle get_locked_encoder(uintptr_t key) = 0;")
    g.print("virtual encoder_handle get_encoder(uintptr_t key) = 0;")
    g.leave_scope("};")
    g.print("}")


def output_cpp(definition, g):
    g.print(standard.HEADER)
    g.print('#include "command_serializer.h"')
    g.print('#include "struct_serialization.h"')
    g.print('#include "forwards.h"')
    g.print('namespace gapid2 {')

    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.line()
        g.enter_scope(
            f'{cmd.ret.short_str()} command_serializer::{cmd.name}({", ".join(prms)}) {{')
        arg_serialization.output_command(
            cmd, definition, g, False, True, True)
        g.leave_scope('}')
    g.print("}")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_serializer.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_recorder(definition, g)

    with open(os.path.join(args.output_location, "command_serializer.cpp"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_cpp(definition, g)


if __name__ == "__main__":
    main(args.get_args())
