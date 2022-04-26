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
    with open(os.path.join(args.output_location, "transform.h"), mode="w") as transform:
        g = generator.generator(transform)
        g.print(standard.HEADER)
        g.print('#include "transform_base.h"')
        g.print('#include <type_traits>')
        g.print('namespace gapid2 {')
        g.line()
        g.print('template <typename T>')
        g.print('concept TransformBase = std::is_base_of_v<transform_base, T>;')
        g.line()
        g.print('class state_block;')
        g.print('template<TransformBase T>')
        g.enter_scope('class transform : public T {')
        g.print_scoping('public:')
        g.enter_scope('transform(transform_base* next) {')
        g.enter_scope('if (!next) {')
        g.print('return;')
        g.leave_scope('}')
        g.print('*(static_cast<transform_base*>(this)) = *next;')
        g.enter_scope(
            f'if constexpr (std::is_base_of_v<state_block, T>) {{')
        g.print('next->state_block_ = this;')
        g.leave_scope(f'}}')
        for cmd in definition.commands.values():
            g.enter_scope(
                f'if constexpr (&T::{cmd.name} != &transform_base::{cmd.name}) {{')
            g.print(f'next->{cmd.name}_next = this;')
            g.leave_scope(f'}}')
        g.leave_scope('}')
        g.leave_scope('};')
        g.print('}  // namespace gapid2')


if __name__ == "__main__":
    main(args.get_args())
