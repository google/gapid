import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os
import hashlib


def output_deserializer(definition, g):
    g.print(standard.HEADER)
    g.print('#include "transform_base.h"')
    g.print('#include "struct_deserialization.h"')
    g.print('#include "decoder.h"')
    g.print('#include "custom.h"')
    g.print('namespace gapid2 {')
    g.enter_scope(
        'class command_buffer_deserializer : public t ransform_base {')
    g.print_scoping('public:')
    for cmd in definition.commands.values():
        if cmd.args[0].name != "commandBuffer":
            continue
        prms = [x.short_str() for x in cmd.args]
        g.enter_scope(f'void call_{cmd.name}(decoder* decoder_) {{')
        arg_serialization.output_command_deserializer(
            cmd, definition, g, False, True)
        g.leave_scope('}')
    g.print("  std::function<void(uint64_t)> notify_pre_command_fn;")
    g.enter_scope(
        "virtual void notify_pre_command(uint64_t command_number) {")
    g.enter_scope("if (notify_pre_command_fn) {")
    g.print("notify_pre_command_fn(command_number);")
    g.leave_scope("}")
    g.leave_scope("}")
    g.enter_scope(
        "virtual void DeserializeStream(decoder* decoder_, bool raw_stream = false) {")
    g.print("uint64_t command_number = 0;")
    g.enter_scope("do {")
    g.print("command_number++;")
    g.enter_scope("if (!raw_stream) {")
    g.print("const uint64_t data_left = decoder_->data_left();")
    g.print("if (data_left < sizeof(uint64_t)) { return; }")
    g.print("auto needed = decoder_->decode<uint64_t>();")
    g.print(
        "if (data_left - sizeof(uint64_t) < needed) { return; } ")
    g.leave_enter_scope("} else {")
    g.print(
        "if (!decoder_->has_data_left()) { return; } ")
    g.leave_scope("}")
    g.print("uint64_t command_idx = decoder_->decode<uint64_t>();")
    g.print("uint64_t flags = decoder_->decode<uint64_t>();")
    g.print("(void)flags;")
    g.print("notify_pre_command(command_number-1);")
    g.enter_scope("switch(command_idx) {")

    for cmd in definition.commands.values():
        if cmd.args[0].name != 'commandBuffer':
            continue
        sha = int.from_bytes(hashlib.sha256(
            cmd.name.encode('utf-8')).digest()[:8], 'little')
        g.print(
            f"case {sha}u: call_{cmd.name}(decoder_); continue;")
    g.enter_scope('default:')
    g.post_leave_scope('std::abort();')
    g.leave_scope("}")
    g.leave_scope("} while(true);")
    g.leave_scope("}")
    g.print("transform_base* next = nullptr;")
    g.leave_scope('};')
    g.print('} // namespace gapid2')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_buffer_deserializer.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_deserializer(definition, g)


if __name__ == "__main__":
    main(args.get_args())
