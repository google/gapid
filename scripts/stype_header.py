import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_enc(enc):
    enc.print(standard.HEADER)
    enc.line()
    enc.print('namespace gapid2 {')
    enc.print('template<typename T>')
    enc.enter_scope('struct get_stype {')
    enc.leave_scope('};')
    enc.line()


def output_struct_data(x, enc):
    if x.stype == None:
        return
    enc.print('template<>')
    enc.enter_scope(f'struct get_stype<{x.name}> {{')
    enc.print(f'  static const VkStructureType sType = {x.stype};')
    enc.leave_scope('};')
    enc.line()


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "stype_header.h"), mode="w") as fenc:
        enc = generator.generator(fenc)

        out_enc(enc)

        for x in definition.get_sorted_structs():
            if type(x) == vulkan.struct:
                output_struct_data(x, enc)

        enc.leave_scope('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
