import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import generator.arg_serialization as arg_serialization
import platform
import os
import hashlib


def print_arg_ptr(cmd, param, tp, count, g):
    nm = f"{param.name}"
    tp = tp.get_noncv_type()
    idx = "[0]"

    p = [x for x in cmd.args if x.name == count]
    if len(p):
        if type(p[0].type) == vulkan.pointer_type:
            count = f"*{count}"

    if count != "1":
        g.print(f"printer_->begin_array(\"{nm}\");")
        g.enter_scope(f"for (size_t i = 0; i < {count}; ++i) {{")
        idx = "[i]"
        nm = ""
    if tp.name == "void":
        g.print(f"// Ignoring void for now")
    elif type(tp) == vulkan.handle:
        if tp.dispatch == vulkan.handle.DISPATCHABLE:
            g.print(
                f"printer_->print(\"{nm}\", reinterpret_cast<uintptr_t>({param.name}{idx}));")
        else:
            g.print(
                f"printer_->print(\"{nm}\", reinterpret_cast<uintptr_t>({param.name}{idx}));")
    elif type(tp) == vulkan.struct:
        prms = [f"\"{nm}\"", f"state_block_", f"{param.name}{idx}", "printer_"]
        prms.extend([x.name() for x in tp.get_serialization_params()])
        g.print(f"print_{tp.name}({', '.join(prms)});")
    elif type(tp) == vulkan.basetype:
        g.print(
            f"printer_->print(\"{nm}\", {param.name}{idx});")
    elif type(tp) == vulkan.platform_type:
        enc_type = str(tp)
        g.print(
            f"printer_->print(\"{nm}\", {param.name}{idx});")
    elif type(tp) == vulkan.union:
        g.print(f"// Ignoring union for now")
    elif type(tp) == vulkan.enum:
        g.print(
            f"printer_->print(\"{nm}\", {param.name}{idx});")
    elif type(tp) == vulkan.external_type:
        g.print(f'// Not printing type because its external')
    else:
        vulkan.error(
            f'Error printing {param.name} type: {tp.name} :: type_type: {type(tp)}')
    if count != "1":
        g.leave_scope(f"}}")
        g.print(f"printer_->end_array();")


def output_arg_print(cmd, param, g):
    if param.name == "pAllocator":
        return
    tp = param.type
    nm = f"{param.name}"
    while type(tp) == vulkan.const_type:
        tp = tp.child
    if type(tp) == vulkan.enum:
        g.print(
            f"printer_->print(\"{nm}\", {nm});")
    elif type(tp) == vulkan.handle:
        if tp.dispatch == vulkan.handle.DISPATCHABLE:
            g.print(
                f"printer_->print(\"{nm}\", reinterpret_cast<uintptr_t>({nm}));")
        else:
            g.print(
                f"printer_->print(\"{nm}\", reinterpret_cast<uintptr_t>({nm}));")
    elif type(tp) == vulkan.basetype:
        g.print(f"printer_->print(\"{nm}\", {nm});")
    elif type(tp) == vulkan.platform_type:
        enc_type = str(tp)
        g.print(f"printer_->print(\"{nm}\", {nm});")
    elif type(tp) == vulkan.pointer_type:
        if param.len and param.len == 'null-terminated':
            if tp.pointee.get_noncv_type().name != 'char':
                vulkan.error("Non-char null terminated type")
            g.print(f"printer_->print_string(\"{nm}\", {nm});")
            return
        if param.optional:
            g.enter_scope(f"if ({nm}) {{")
        ct = "1"
        if param.len:
            ct = param.len
        print_arg_ptr(cmd, param, tp.pointee, ct, g)
        if param.optional:
            g.leave_enter_scope(f"}} else {{ ")
            g.print(f"printer_->print_null(\"{nm}\");")
            g.leave_scope(f"}}")

    elif type(tp) == vulkan.array_type:
        print_arg_ptr(cmd, param, tp.get_noncv_type(),
                      f'{tp.size}', g)
    elif type(tp) == vulkan.external_type:
        g.print(f"// Not printing external type")
    else:
        vulkan.error(f'Error printing {param.name} type: {tp.name}')


def output_printer(definition, g):
    g.print(standard.HEADER)
    g.print('#include "transform_base.h"')
    g.print('#include "struct_print.h"')
    g.print('#include "printer.h"')
    g.print('#include "custom.h"')
    g.print('#include <functional>')
    g.print('namespace gapid2 {')
    g.enter_scope(
        'class command_printer : public transform_base {')
    g.print_scoping('public:')
    for cmd in definition.commands.values():
        g.enter_scope(f"{cmd.short_str()} override {{")
        g.print(f"printer_->begin_object(\"\");")
        g.print(f"printer_->print_string(\"name\", \"{cmd.name}\");")
        g.enter_scope(
            'if (get_flags) {')
        g.print(f"printer_->print(\"tracer_flags\", get_flags());")
        g.leave_scope('}')
        for x in cmd.args:
            output_arg_print(cmd, x, g)
        args = ", ".join(x.name for x in cmd.args)
        if (cmd.ret.name != 'void'):
            g.print(
                f"auto ret = transform_base::{cmd.name}({args});")
        else:
            g.print(f"transform_base::{cmd.name}({args});")
        g.print(f"printer_->end_object();")
        if (cmd.ret.name != 'void'):
            g.print(f"return ret;")
        g.leave_scope(f"}}")
    g.print('std::function<uint64_t()> get_flags;')
    g.print('printer* printer_;')
    g.leave_scope('};')
    g.print('} // namespace gapid2')


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))
    with open(os.path.join(args.output_location, "command_printer.h"), mode="w") as cbr:
        g = generator.generator(cbr)
        output_printer(definition, g)


if __name__ == "__main__":
    main(args.get_args())
