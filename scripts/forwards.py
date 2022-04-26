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

    with open(os.path.join(args.output_location, "forwards.h"), mode="w") as forwards:
        all_enc_args = []
        g = generator.generator(forwards)
        g.print(standard.HEADER)
        g.print('#include "bind_first.h"')
        g.print('#include "temporary_allocator.h"')
        g.print('namespace gapid2 {')
        g.print('struct encoder;')
        g.print('struct decoder;')
        output = []
        for cmd in definition.commands.values():
            for arg in cmd.args:
                if arg.name == 'pAllocator':
                    continue
                # if type(arg.type) == pointer_type and not arg.type.const:
                #   continue
                tp = arg.type.get_noncv_type()
                all_enc_args.extend([y.signature()
                                    for y in tp.get_serialization_params()])
                all_enc_args.extend(
                    [y.signature() for y in tp.get_serialization_params(vulkan.DESERIALIZE)])
                all_enc_args.extend([y.signature()
                                    for y in tp.get_serialization_params(vulkan.CLONE)])
        g.line()
        for x in all_enc_args:
            if not x in output:
                g.print(f'{x};')
                output.append(x)
        g.print('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
