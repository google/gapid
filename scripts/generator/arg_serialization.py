import platform
import os
import hashlib
import copy
from .vulkan import pointer_type, const_type, union, struct, basetype, platform_type, error, enum, handle, array_type, DESERIALIZE


def output_arg_dec(cmd, x, vop, arg_idx, idx, g):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child

    if type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", "decoder_"]
        prms.extend([x.name()
                    for x in tp.get_serialization_params(DESERIALIZE)])

        g.print(
            f"_custom_deserialize_{tp.name}(state_block_, {', '.join(prms)});")
    if type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "decoder_"]
        prms.extend([x.name()
                    for x in tp.get_serialization_params(DESERIALIZE)])
        g.print(f"deserialize_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == basetype:
        if tp.base_type == "size_t":
            g.print(
                f"{vop}{x.name}{idx} = static_cast<size_t>(decoder_->decode<uint64_t>());")
        else:
            g.print(f"{vop}{x.name}{idx} = decoder_->decode<{tp.base_type}>();")
    elif type(tp) == platform_type:
        if str(tp) == "size_t":
            g.print(
                f"{vop}{x.name}{idx} = static_cast<size_t>(decoder_->decode<uint64_t>());")
        else:
            g.print(f"{vop}{x.name}{idx} = decoder_->decode<{str(tp)}>();")
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = [x.name for x in cmd.args]
            prms.append("decoder_")
            g.print(
                f"_custom_deserialize_{cmd.name}_{x.name}(state_block_, {', '.join(prms)});")
            return
        # Special case for strings
        if x.len and x.len.startswith('latexmath'):
            g.print(f"// Latexmath string")
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            # encode
            g.print(f"uint64_t len_ = decoder_->decode<uint64_t>();")
            g.enter_scope(f"if (len_) {{")
            g.print(
                f"{vop}{x.name}{idx} = static_cast<{x.type.get_noncv_type().get_member('')[0:-1]}*>(decoder_->get_memory(len_));")
            g.print(
                f"decoder_->decode_primitive_array<char>({vop}{x.name}{idx}, len_);")
            g.leave_enter_scope(f"}} else {{")
            g.print(f"{vop}{x.name}{idx} = nullptr;")
            g.leave_scope(f"}}")
            return

        ct = "1"
        if x.len:
            ct = f"{vop}{x.len.split(',')[0]}"
            p = [x for x in cmd.args if x.name == ct]
            if len(p):
                if type(p[0].type) == pointer_type:
                    ct = f"*{ct}"

        if x.optional:
            g.enter_scope(f"if (decoder_->decode<char>()) {{")

        if x.optional or x.len:
            g.print(
                f"{vop}{x.name}{idx} = decoder_->get_typed_memory<{tp.pointee.get_non_const_member('')[0:-1]}>({ct});")

        mem_idx = f'{idx}[0]'
        if x.len:
            ii = g.get_depth()
            argct = f"{vop}{x.len.split(',')[0]}"
            p = [x for x in cmd.args if x.name == argct]
            if len(p):
                if type(p[0].type) == pointer_type:
                    argct = f"*{argct}"
            g.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {argct}; ++i_{ii}) {{")
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee

        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_arg_dec(cmd, xm, vop, arg_idx, mem_idx, g)

        if x.len:
            g.leave_scope(f"}}")
        if x.optional:
            g.leave_enter_scope(f"}} else {{")
            g.print(f"{vop}{x.name}{idx} = nullptr;")
            g.leave_scope(f"}}")

    elif type(tp) == enum:
        g.print(
            f"{vop}{x.name}{idx} = static_cast<{tp.name}>(decoder_->decode<{tp.base_type.name}>());")
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            g.print(
                f"{vop}{x.name}{idx} = reinterpret_cast<{tp.name}>(static_cast<uintptr_t>(decoder_->decode<uint64_t>()));")
        else:
            g.print(
                f"{vop}{x.name}{idx} = reinterpret_cast<{tp.name}>(decoder_->decode<uint64_t>());")
    elif type(tp) == array_type:
        g.enter_scope(f"for (size_t i = 0; i < {tp.size}; ++i) {{")
        mem_idx = f'{idx}[i]'
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_arg_dec(cmd, xm, vop, arg_idx, mem_idx, g)
        g.leave_scope(f"}}")


def output_command_deserializer(cmd, definition, g, serialize_return=True):
    arg_idx = 0
    g.print(f"// -------- Args ------")
    for arg in cmd.args:
        t = arg.type
        while(type(t) == const_type):
            t = t.child
        if type(t) == array_type:
            g.print(f"{t.child.get_noncv_type().name} {arg.name}[{t.size}];")
            continue
        if type(t) != pointer_type:
            g.print(f"{t.name} {arg.name};")
            continue

        if t.pointee.name == "void":
            g.print(f"void* {arg.name};")
            continue
        if type(t.pointee) == pointer_type and t.pointee.pointee.name == "void":
            g.print(f"void** {arg.name};")
            continue
        ptr = arg.type
        name = arg.name
        if arg.inout():
            name = f'tmp_{arg.name}'
        if not arg.optional and not arg.len:
            g.print(f"{t.pointee.get_noncv_type().name} {name}[1];")
        elif arg.optional:
            # Is a pointer type
            g.print(f"{arg.type.get_noncv_type().name}* {name}; // optional")
        elif arg.len:
            g.print(
                f"{arg.type.get_noncv_type().name}* {name}; // length {arg.len}")
    g.print(f"// -------- Serialized Params ------")
    for arg in cmd.args:
        if arg.name == 'pAllocator':
            g.print(f"pAllocator = nullptr; // pAllocator ignored on replay")
            continue
        t = ""
        if type(arg.type) == pointer_type and not arg.type.const:
            if not arg.inout():
                continue
            t = "tmp_"

        output_arg_dec(cmd, arg, t, arg_idx, "",  g)
        arg_idx += 1

    g.print(f"// -------- Out Params ------")
    for arg in cmd.args:
        if not(type(arg.type) == pointer_type and not arg.type.const):
            continue

        if arg.inout():
            if not arg.optional and not arg.len:
                g.print(f"{arg.type.pointee.name} {arg.name}[1]; // inout")
            elif arg.optional:
                # Is a pointer type
                g.print(f"{arg.type.name} {arg.name}; // optional inout")
            elif arg.len:
                g.print(f"{arg.type.name} {arg.name}; // length {arg.len} inout")
            output_arg_dec(cmd, arg, "", arg_idx, "", g)
        else:
            output_arg_dec(cmd, arg, "", arg_idx, "", g)
        arg_idx += 1

    for arg in cmd.args:
        if not(type(arg.type) == pointer_type and not arg.type.const):
            continue

        if arg.inout():
            ct = "1"
            if arg.len:
                ct = f"{x.len.split(',')[0]}"
                p = [x for x in cmd.args if x.name == ct]
                if len(p):
                    if type(p[0].type) == pointer_type:
                        ct = f"*{ct}"
            g.print(
                f"memcpy({arg.name}, tmp_{arg.name}, sizeof({arg.name}[0]) * {ct}); // setting inout properly")
    if serialize_return:
        if cmd.ret.name == "VkResult":
            g.print(f"VkResult current_return_ = decoder_->decode<VkResult>();")
            g.print(f"(void)current_return_;")
    g.print(f"// -------- Call ------")
    args = ", ".join(x.name for x in cmd.args)
    g.print(f"transform_base::{cmd.name}({args});")


def output_arg_enc(cmd, x, vop, arg_idx, idx, g):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child

    if type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend([x.name() for x in tp.get_serialization_params()])

        g.print(
            f"_custom_serialize_{tp.name}(state_block_, {', '.join(prms)});")
    if type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend([x.name() for x in tp.get_serialization_params()])

        g.print(f"serialize_{tp.name}(state_block_, {', '.join(prms)});")
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        g.print(f"enc->template encode<{enc_type}>({vop}{x.name}{idx});")
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        g.print(f"enc->template encode<{enc_type}>({vop}{x.name}{idx});")
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = [x.name for x in cmd.args]
            prms.append("enc")
            g.print(
                f"_custom_serialize_{cmd.name}_{x.name}(state_block_, {', '.join(prms)});")
            return
        # Special case for strings
        if x.len and x.len.startswith('latexmath'):
            g.print(f"// Latexmath string")
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            # encode
            g.print(f"if ({vop}{x.name}{idx}) {{")
            g.print(
                f"uint64_t len = strlen({vop}{x.name}{idx});")
            g.print(f"enc->template encode<uint64_t>(len + 1);")
            g.print(
                f"enc->template encode_primitive_array<char>({vop}{x.name}{idx}, len + 1);")
            g.print(f"}} else {{")
            g.print(f"enc->template encode<uint64_t>(0);")
            g.print(f"}}")
            return

        if x.optional:
            g.enter_scope(f"if ({vop}{x.name}{idx}) {{")
            g.print(f"  enc->template encode<char>(1);")

        mem_idx = f'{idx}[0]'
        if x.len:
            ii = g.get_depth()
            argct = f"{vop}{x.len.split(',')[0]}"
            p = [x for x in cmd.args if x.name == argct]
            if len(p):
                if type(p[0].type) == pointer_type:
                    argct = f"*{argct}"
            g.enter_scope(
                f"for (size_t i_{ii} = 0; i_{ii} < {argct}; ++i_{ii}) {{")
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee

        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_arg_enc(cmd, xm, vop, arg_idx, mem_idx, g)

        if x.len:
            g.leave_scope(f"}}")
        if x.optional:
            g.leave_enter_scope(f"}} else {{")
            g.print(f"enc->template encode<char>(0);")
            g.leave_scope(f"}}")

    elif type(tp) == enum:
        g.print(
            f"enc->template encode<{tp.base_type.name}>({vop}{x.name}{idx});")
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            g.print(
                f"enc->template encode<uint64_t>(reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));")
        else:
            g.print(
                f"enc->template encode<uint64_t>(reinterpret_cast<uint64_t>({vop}{x.name}{idx}));")
    elif type(tp) == array_type:
        g.enter_scope(f"for (size_t i = 0; i < {tp.size}; ++i) {{")
        mem_idx = f'{idx}[i]'
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_arg_enc(cmd, xm, vop, arg_idx, mem_idx, g)
        g.leave_scope(f"}}")


def output_command(cmd, definition, g, only_return=False, optional_serialize=False, serialize_return=True):
    # Special case. Anything that can unblock the CPU we have to actually cause
    # block around call. Otherwise the returns might encode out of order
    # which means we deadlock on replay.
    if (cmd.name == "vkWaitSemaphoresKHR" or cmd.name == "vkWaitSemaphores"):
        g.print(
            f"auto enc = get_encoder(reinterpret_cast<uintptr_t>({cmd.args[0].name}));")
    else:
        g.print(
            f"auto enc = get_locked_encoder(reinterpret_cast<uintptr_t>({cmd.args[0].name}));")
    sha = int.from_bytes(hashlib.sha256(
        cmd.name.encode('utf-8')).digest()[:8], 'little')

    if not only_return:
        if optional_serialize:
            g.enter_scope("if (enc) {")
        g.print(f"enc->template encode<uint64_t>({sha}u);")
        arg_idx = 0
        for arg in cmd.args:
            if arg.name == 'pAllocator':
                g.print(
                    f"// Skipping: {arg.name} for as it cannot be replayed")
                continue
            if type(arg.type) == pointer_type and not arg.type.const:
                if arg.inout():
                    g.print(f"// Inout: {arg.name}")
                else:
                    g.print(
                        f"// Skipping: {arg.name} for now as it is an output param")
                    continue
            output_arg_enc(cmd, arg, "", arg_idx, "", g)
            arg_idx += 1
        if optional_serialize:
            g.leave_scope("}")
    args = ", ".join(x.name for x in cmd.args)
    if (cmd.ret.name == 'void'):
        g.print(f"transform_base::{cmd.name}({args});")
    else:
        g.print(f"const auto ret = transform_base::{cmd.name}({args});")

    arg_idx = 0
    any = False
    for arg in cmd.args:
        if type(arg.type) == pointer_type and not arg.type.const:
            if not any:
                if optional_serialize:
                    g.enter_scope("if (enc) {")
                any = True
            if arg.inout():
                g.print(f"// Inout value: {arg.name}")
            else:
                g.print(f"// Return value: {arg.name}")

        else:
            continue
        output_arg_enc(cmd, arg, "", arg_idx, "", g)
        arg_idx += 1
    if cmd.ret.name != "void":
        if serialize_return:
            if not any:
                if optional_serialize:
                    g.enter_scope("if (enc) {")
                any = True
            if cmd.ret.name == "VkResult":
                g.print(f"enc->template encode<uint32_t>(ret);")
            if any:
                if optional_serialize:
                    g.leave_scope("}")
                any = False
        g.print(f"return ret;")
    if any:
        if optional_serialize:
            g.leave_scope("}")
        any = False
