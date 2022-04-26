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
    with open(os.path.join(args.output_location, "handle_defines.inl"), mode="w") as handle_defines:
        g = generator.generator(handle_defines)
        g.print(standard.COPYRIGHT)
        g.print("#ifndef PROCESS_HANDLE")
        g.print('#error "Please define PROCESS_HANDLE"')
        g.print('#endif')
        g.line()

        for x in definition.types.values():
            if type(x) == vulkan.handle:
                g.print(f'PROCESS_HANDLE({x.name})')


if __name__ == "__main__":
    main(args.get_args())
