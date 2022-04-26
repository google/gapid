import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_header(instance_functions):
    instance_functions.print(standard.HEADER)
    instance_functions.line()
    instance_functions.print('namespace gapid2 {')
    instance_functions.enter_scope('struct instance_functions {')


def out_footer(instance_functions):
    instance_functions.leave_scope('};')
    instance_functions.print('}')


def output_constructor(definition, instance_functions):
    instance_functions.enter_scope(
        'instance_functions(VkInstance instance, PFN_vkGetInstanceProcAddr get_instance_proc_addr) {')
    for cmd in definition.commands.values():
        if cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
            continue
        instance_functions.print(
            f'{cmd.name}_ = reinterpret_cast<PFN_{cmd.name}>(get_instance_proc_addr(instance, "{cmd.name}"));')
    instance_functions.leave_scope('};')


def output_ptrs(definition, instance_functions):
    for cmd in definition.commands.values():
        if cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
            continue
        instance_functions.print(f'PFN_{cmd.name} {cmd.name}_;')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "instance_functions.h"), mode="w") as f:
        instance_functions = generator.generator(f)

        out_header(instance_functions)
        output_constructor(definition, instance_functions)
        output_ptrs(definition, instance_functions)
        out_footer(instance_functions)


if __name__ == "__main__":
    main(args.get_args())
