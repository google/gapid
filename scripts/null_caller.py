import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_header(null_caller):
    null_caller.print(standard.HEADER)
    null_caller.enter_scope(
        '''#include <memory>
# include <unordered_map>
# include <shared_mutex>

# include "transform_base.h"
# include "instance_functions.h"
# include "device_functions.h"

namespace gapid2 {
class null_caller : public transform_base {''')
    null_caller.print(
        '''public:
''')


def out_footer(null_caller, definition):
    null_caller.leave_scope(
        '''};
} // namespace gapid2
''')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "null_caller.h"), mode="w") as f:
        null_caller = generator.generator(f)

        out_header(null_caller)

        for cmd in definition.commands.values():
            null_caller.enter_scope(f"{cmd.short_str()} override {{")

            if (cmd.ret.name != 'void'):
                null_caller.print(f"return static_cast<{cmd.ret.name}>(0);")
            null_caller.leave_scope(f"}}")

        out_footer(null_caller, definition)


if __name__ == "__main__":
    main(args.get_args())
