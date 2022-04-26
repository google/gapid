import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "indirect_functions.h"), mode="w") as transform:
        g = generator.generator(transform)
        g.print(standard.HEADER)
        g.print("namespace gapid2 {")
        g.enter_scope('struct indirect_functions {')
        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            g.print(
                f'{cmd.ret.short_str()}(*fn_{cmd.name})(void*, {", ".join(prms)});')
            g.print(f'void* {cmd.name}_user_data;')
        g.leave_scope('};')
        g.print('}')


if __name__ == "__main__":
    main(args.get_args())
