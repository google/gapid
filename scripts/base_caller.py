import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os
import copy


def out_header(base_caller):
    base_caller.print(standard.HEADER)
    base_caller.enter_scope(
        '''#include <memory>
# include <unordered_map>
# include <shared_mutex>

# include "transform_base.h"
# include "instance_functions.h"
# include "device_functions.h"

namespace gapid2 {
class base_caller : public transform_base {''')
    base_caller.print(
        '''public:
''')


def out_footer(base_caller, definition):
    for cmd in definition.commands.values():
        if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue' and cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
            base_caller.print(f"PFN_{cmd.name} {cmd.name}_ = nullptr;")
    base_caller.print(
        f"PFN_vkGetInstanceProcAddr vkGetInstanceProcAddr_ = nullptr;")
    base_caller.print(
        f"PFN_vkGetDeviceProcAddr vkGetDeviceProcAddr_ = nullptr;")
        
    for t in definition.types.values():
        if type(t) == vulkan.handle and t.dispatch == vulkan.handle.DISPATCHABLE:
            extra_args = "VkDevice device"
            if t.name == "VkInstance":
                extra_args = "const VkInstanceCreateInfo* create_info"
            elif t.name == "VkPhysicalDevice":
                extra_args = "VkInstance instance"
            elif t.name == "VkDevice":
                extra_args = "VkPhysicalDevice physical_device"
            base_caller.print(
                f'void on_{t.name.lower()[2:]}_created({extra_args}, {t.name}* val, uint32_t count);')
            base_caller.print(
                f'void on_{t.name.lower()[2:]}_destroyed(const {t.name}* val, uint32_t count);')
    for t in definition.types.values():
        if type(t) == vulkan.handle and t.dispatch == vulkan.handle.DISPATCHABLE:
            base_caller.print(
                f"std::shared_mutex {t.name.lower()[2:]}_lock_;")
            if t.name == "VkInstance" or t.name == "VkDevice":
                base_caller.print(
                    f"std::unordered_map<{t.name}, std::unique_ptr<{t.name.lower()[2:]}_functions>> {t.name.lower()[2:]}_functions_;")
            else:
                if t.name == "VkPhysicalDevice":
                    base_caller.print(
                        f"std::unordered_map<{t.name}, instance_functions*> {t.name.lower()[2:]}_functions_;")
                else:
                    base_caller.print(
                        f"std::unordered_map<{t.name}, device_functions*> {t.name.lower()[2:]}_functions_;")
    base_caller.leave_scope(
        '''};
} // namespace gapid2
''')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "base_caller.h"), mode="w") as f:
        base_caller = generator.generator(f)

        out_header(base_caller)

        for cmd in definition.commands.values():
            base_caller.enter_scope(f"{cmd.short_str()} override {{")

            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue' and cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                base_caller.print(f"const auto fn = {cmd.name}_;")
            else:
                base_caller.print(
                    f"{cmd.args[0].type.name.lower()[2:]}_lock_.lock_shared();")
                base_caller.print(
                    f"const auto fn = {cmd.args[0].type.name.lower()[2:]}_functions_[{cmd.args[0].name}]->{cmd.name}_;")
                base_caller.print(
                    f"{cmd.args[0].type.name.lower()[2:]}_lock_.unlock_shared();")

            args = ", ".join(x.name for x in cmd.args)
            if (cmd.ret.name == 'void'):
                base_caller.print(f"fn({args});")
            else:
                base_caller.print(f"auto ret = fn({args});")

            creation = type(cmd.args[-1].type) == vulkan.pointer_type
            creation = creation and not cmd.args[-1].type.const
            creation = creation and type(
                cmd.args[-1].type.get_noncv_type()) == vulkan.handle
            creation = creation and cmd.args[-1].type.get_noncv_type(
            ).dispatch == vulkan.handle.DISPATCHABLE
            if creation:
                len = "1"
                if cmd.args[-1].len:
                    len = cmd.args[-1].len
                    # if len is a pointer type, then the actual length is the deref of that pointer
                    if len in [x.name for x in cmd.args] and type([x for x in cmd.args if x.name == len][0].type) == vulkan.pointer_type:
                        len = "*" + len
                base_caller.print(
                    f"on_{cmd.args[-1].type.get_noncv_type().name[2:].lower()}_created({cmd.args[0].name}, {cmd.args[-1].name}, {len});")

            deletion = cmd.name.startswith(
                "vkDestroy") or cmd.name.startswith("vkFree")
            if deletion:
                dest_arg = cmd.args[1]
                if cmd.name == "vkDestroyInstance" or cmd.name == "vkDestroyDevice":
                    dest_arg = cmd.args[0]
                if cmd.name == "vkFreeDescriptorSets" or cmd.name == "vkFreeCommandBuffers":
                    dest_arg = cmd.args[-1]
                len = "1"
                if dest_arg.len:
                    len = dest_arg.len
                    # if len is a pointer type, then the actual length is the deref of that pointer
                    if len in [x.name for x in cmd.args] and type([x for x in cmd.args if x.name == len][0].type) == vulkan.pointer_type:
                        len = "*" + len
                tp = dest_arg.type

                while type(tp) == vulkan.const_type:
                    tp = tp.child
                arg = dest_arg.name
                if type(tp) != vulkan.pointer_type:
                    arg = "&" + arg
                if tp.get_noncv_type().dispatch == vulkan.handle.DISPATCHABLE:
                    base_caller.print(
                        f"on_{dest_arg.type.get_noncv_type().name[2:].lower()}_destroyed({arg}, {len});")

            if (cmd.ret.name != 'void'):
                base_caller.print(f"return ret;")

            base_caller.leave_scope(f"}}")

        out_footer(base_caller, definition)


if __name__ == "__main__":
    main(args.get_args())
