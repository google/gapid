import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os


def needs_command_buffer_invalidation(definition, cmd):
    if cmd.args[0].name != "commandBuffer":
        return
    for args in cmd.args[1:]:
        if args.type.has_handle():
            return True
    return False


def output_recorder(definition, g):
    g.print(standard.HEADER)
    g.print('#include "transform_base.h"')
    g.print("namespace gapid2 {")
    g.enter_scope("class command_buffer_invalidator : public transform_base {")
    g.print_scoping("public:")
    g.print("using super = transform_base;")

    for cmd in definition.commands.values():
        if not needs_command_buffer_invalidation(definition, cmd):
            continue
        prms = [x.short_str() for x in cmd.args]
        g.print(f'{cmd.short_str()} override;')
    g.line()
    g.leave_scope("};")
    g.print("}")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_buffer_invalidator.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_recorder(definition, g)


if __name__ == "__main__":
    main(args.get_args())
