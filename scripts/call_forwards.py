import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os


def output_command_header(cmd, definition, header):
    header.print(
        f'VKAPI_ATTR {cmd.ret.name} VKAPI_CALL {cmd.name}({", ".join([x.short_str() for x in cmd.args])});')
    pass


def output_command_forward(cmd, definition, source):
    source.enter_scope(f"{cmd.short_str()} {{")
    args = ", ".join(x.name for x in cmd.args)
    source.print(
        f"return get_layer_base()->get_top_level_functions()->{cmd.name}({args});")
    source.leave_scope(f"}}")


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "call_forwards.h"), mode="w") as fh, open(os.path.join(args.output_location, "call_forwards.cpp"), mode="w") as fcpp:
        header = generator.generator(fh)
        header.print(standard.HEADER)
        header.line()

        source = generator.generator(fcpp)
        source.print(standard.HEADER)
        source.line()

        source.print('#include "call_forwards.h"')

        source.print('#include "layer_base.h"')
        source.print('namespace gapid2 {')
        source.print('layer_base* get_layer_base();')

        header.print('namespace gapid2 {')

        for cmd in definition.commands.values():
            output_command_header(cmd, definition, header)
            output_command_forward(cmd, definition, source)
        source.enter_scope(
            '''PFN_vkVoidFunction get_instance_function(const char * name) {''')
        for cmd in definition.commands.values():
            source.enter_scope(f"if (!strcmp(name, \"{cmd.name}\")) {{")
            source.print(
                f"return (PFN_vkVoidFunction) {cmd.name};")
            source.leave_scope(f"}}")
        source.print('return nullptr;')
        source.leave_scope("}")
        source.enter_scope(
            'PFN_vkVoidFunction get_device_function(const char* name) {')
        for cmd in definition.commands.values():
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue':
                continue
            source.enter_scope(
                f"if (!strcmp(name, \"{cmd.name}\")) {{")
            source.print(
                f"return (PFN_vkVoidFunction) {cmd.name};")
            source.leave_scope(f"}}")
        source.print('return nullptr;')
        source.leave_scope('}')
        source.print('} // namespace gapid2')
        header.print(
            'PFN_vkVoidFunction get_instance_function(const char * name);')
        header.print(
            'PFN_vkVoidFunction get_device_function(const char * name);')
        header.print('} // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
