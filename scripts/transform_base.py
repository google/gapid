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
    with open(os.path.join(args.output_location, "transform_base.h"), mode="w") as transform_base:
        g = generator.generator(transform_base)
        g.print(standard.HEADER)
        g.print("#include \"data_provider.h\"")
        g.print('namespace gapid2 {')
        g.print(f'class state_block;')
        g.enter_scope('class transform_base {')
        g.print_scoping('public:')
        for cmd in definition.commands.values():
            g.enter_scope(f'virtual {cmd.short_str()} {{')
            args = [x.name for x in cmd.args]
            g.print(f'return {cmd.name}_next->{cmd.name}({", ".join(args)});')
            g.leave_scope(f'}}')
        g.line()
        for cmd in definition.commands.values():
            g.print(f'transform_base* {cmd.name}_next = nullptr;')
        g.line()
        g.print(f'state_block* state_block_ = nullptr;')
        g.enter_scope("uint64_t get_current_command_index() const {")
        g.print(
            "return data_provider_? data_provider_->get_current_command_index(): 0;")
        g.leave_scope("}")
        g.print(f'const transform_data_provider* data_provider_ = nullptr;')
        g.leave_scope('};')
        g.print('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
