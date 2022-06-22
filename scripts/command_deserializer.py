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
        'class command_deserializer : public transform_base {')
    g.print_scoping('public:')
    for cmd in definition.commands.values():
        prms = [x.short_str() for x in cmd.args]
        g.enter_scope(f'virtual void call_{cmd.name}(decoder* decoder_) {{')
        arg_serialization.output_command_deserializer(
            cmd, definition, g, True)
        g.leave_scope('}')
    g.enter_scope(
        "void DeserializeStream(decoder* decoder_, bool raw_stream = false) {")
    g.print("uint64_t current_command_index = 0; // For debugging")
    g.enter_scope("do {")
    g.print("if (!raw_stream) {")
    g.print("const uint64_t data_left = decoder_->data_left();")
    g.print("if (data_left < sizeof(uint64_t) * 2) { return; }")
    g.print(
        "if (data_left - sizeof(uint64_t) < decoder_->decode<uint64_t>()) { return; } ")
    g.print("} else {")
    g.print(
        "if (!decoder_->has_data_left()) { return; } ")
    g.print("}")
    g.print("uint64_t command_idx = decoder_->decode<uint64_t>();")
    g.print("uint64_t flags = decoder_->decode<uint64_t>();")
    g.print("notify_flag(flags);")
    g.print("switch(command_idx) {")

    for cmd in definition.commands.values():
        sha = int.from_bytes(hashlib.sha256(
            cmd.name.encode('utf-8')).digest()[:8], 'little')
        g.print(
            f"case {sha}u: call_{cmd.name}(decoder_); break;")
    g.print('default:')
    g.print('std::abort();')
    g.enter_scope('case 0:  { // mapped_memory_write')
    g.print(
        'VkDeviceMemory mem = reinterpret_cast<VkDeviceMemory>(decoder_->decode<uint64_t>()); ')
    g.print('VkDeviceSize offset = decoder_->decode<VkDeviceSize>();')
    g.print('VkDeviceSize size = decoder_->decode<VkDeviceSize>();')
    g.print('void* data_loc = get_memory_write_location(mem, offset, size);')
    g.enter_scope('if (data_loc) {')
    g.print(
        'decoder_->decode_primitive_array(reinterpret_cast<char*>(data_loc), size);')
    g.leave_enter_scope('} else {')
    g.print(
        'decoder_->drop_primitive_array<char>(size);')
    g.leave_scope('}')
    g.print('break;')
    g.leave_scope('}')
    g.enter_scope('case 1:  { // annotation')
    g.print('uint64_t annotation_size = decoder_->decode<uint64_t>();')
    g.print("std::vector<char> my_s;")
    g.print("my_s.resize(annotation_size);")
    g.print("decoder_->decode_primitive_array(my_s.data(), annotation_size);")
    g.print("annotation(my_s.data());")
    g.print("continue; // We dont want to increment the command idx here")
    g.leave_scope('}')
    g.print("}")
    g.print("current_command_index++;")
    g.leave_scope("} while(true);")
    g.leave_scope("}")
    g.print(
        'virtual void notify_flag(uint64_t flag) {}')
    g.enter_scope(
        'virtual void *get_memory_write_location(VkDeviceMemory, VkDeviceSize, VkDeviceSize) {')
    g.print('return nullptr;')
    g.leave_scope('}')
    g.enter_scope(
        'virtual void annotation(const char* annotation) {')
    g.leave_scope('}')
    g.leave_scope('};')
    g.print('} // namespace gapid2')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_deserializer.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_deserializer(definition, g)


if __name__ == "__main__":
    main(args.get_args())
