import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_header(device_functions):
    device_functions.print(standard.HEADER)
    device_functions.line()
    device_functions.print('namespace gapid2 {')
    device_functions.enter_scope('struct device_functions {')


def out_footer(device_functions):
    device_functions.leave_scope('};')
    device_functions.print('}')


def output_constructor(definition, device_functions):
    device_functions.enter_scope(
        'device_functions(VkDevice device, PFN_vkGetDeviceProcAddr get_device_proc_addr) {')
    for cmd in definition.commands.values():
        if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue':
            continue
        device_functions.print(
            f'{cmd.name}_ = reinterpret_cast<PFN_{cmd.name}>(get_device_proc_addr(device, "{cmd.name}"));')
    device_functions.leave_scope('};')


def output_ptrs(definition, device_functions):
    for cmd in definition.commands.values():
        if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue':
            continue
        device_functions.print(f'PFN_{cmd.name} {cmd.name}_;')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "device_functions.h"), mode="w") as f:
        device_functions = generator.generator(f)

        out_header(device_functions)
        output_constructor(definition, device_functions)
        output_ptrs(definition, device_functions)
        out_footer(device_functions)


if __name__ == "__main__":
    main(args.get_args())
