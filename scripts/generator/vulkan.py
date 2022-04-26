import xml.etree.ElementTree as ET
import sys
from collections import deque
import hashlib
from functools import partial


def error(string):
    print(string)
    sys.exit(-1)


SERIALIZE = 0
DESERIALIZE = 1
CLONE = 2
FIX_HANDLES = 3


def verify_unique_commands(definition):
    shas = {}
    for cmd in definition.commands.values():
        sha = int.from_bytes(hashlib.sha256(
            cmd.name.encode('utf-8')).digest()[:8], 'little')
        if sha in shas:
            error(
                f'Two commands hash to the same value {cmd.name} and {shas[sha]}')
        shas[sha] = cmd.name


class base:
    def __init__(self, name):
        self.added_by_version = None
        self.added_by_extension = None
        self.alias = ""
        self.name = name

    def added_by(self):
        x = ""
        if (self.alias != ""):
            x += f" Alias: {self.alias}"
        if (self.added_by_version):
            x += f" Version: {self.added_by_version}"
        if (self.added_by_extension):
            x += f" Extension: {self.added_by_extension}"
        return x

    def get_noncv_type(self):
        return self

    def __str__(self):
        return self.name

    def short_str(self):
        return self.name

    def get_member(self, member_name):
        return f"{self.name} {member_name}"

    def get_non_const_member(self, member_name):
        return self.get_member(member_name)

    def get_serialization_params(self, _serialization_type=SERIALIZE):
        return []

    def has_handle(self):
        return False

    def has_pnext(self):
        return False


class alias(base):
    def __init__(self, name, tgt):
        base.__init__(self, name)
        self.tgt = tgt

    def __str__(self):
        return self.name

    def has_handle(self):
        return self.tgt.has_handle()

    def has_pnext(self):
        return self.tgt.has_pnext()


class platform_type(base):
    def __init__(self, name):
        base.__init__(self, name)

    def __str__(self):
        return self.name


class external_type(base):
    def __init__(self, name, t):
        base.__init__(self, name)

    def __str__(self):
        return self.name


class pointer_type(base):
    def __init__(self, pointee, const):
        if const:
            base.__init__(self, pointee.short_str() + "const*")
        else:
            base.__init__(self, pointee.short_str() + "*")
        self.pointee = pointee
        self.const = const

    def get_member(self, member_name):
        if self.const:
            return self.pointee.get_member(f"const* {member_name}")
        else:
            return self.pointee.get_member(f"* {member_name}")

    def get_non_const_member(self, member_name):
        if self.const:
            return self.pointee.get_non_const_member(f"* {member_name}")
        else:
            return self.pointee.get_non_const_member(f"* {member_name}")

    def get_noncv_type(self):
        return self.pointee.get_noncv_type()

    def has_handle(self):
        return self.pointee.has_handle()

    def has_pnext(self):
        return self.pointee.has_pnext()


class const_type(base):
    def __init__(self, child):
        base.__init__(self, "const " + child.short_str())
        self.child = child

    def get_member(self, member_name):
        return f"const {self.child.get_member(member_name)}"

    def get_noncv_type(self):
        return self.child.get_noncv_type()

    def has_handle(self):
        return self.child.has_handle()

    def has_pnext(self):
        return self.child.has_pnext()


class member:
    def __init__(self):
        pass


class array_type(base):
    def __init__(self, child, size):
        base.__init__(self, f"{child} [{size}]")
        self.child = child
        self.size = size

    def get_member(self, member_name):
        return f"{self.child.get_member(member_name)}[{self.size}]"

    def get_noncv_type(self):
        return self.child

    def has_handle(self):
        return self.child.has_handle()

    def has_pnext(self):
        return self.child.has_pnext()


def get_subtype(node, vk, api_definition, added_by_version, added_by_extension):
    type = node.find('type')
    tp = type.text
    tail = ""
    text = ""
    ret_type = None
    if type.tail:
        tail = str.lstrip(str.rstrip(type.tail))
    if node.text:
        text = str.lstrip(str.rstrip(node.text))

    if tail == "*" and text == "const":
        ret_type = api_definition.get_const_type(api_definition.get_pointer_type(
            api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)))
    elif tail == "*" and text == "const struct":
        ret_type = api_definition.get_const_type(api_definition.get_pointer_type(
            api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)))
    elif tail == "*" and text == "struct":
        ret_type = api_definition.get_pointer_type(
            api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension))
    elif tail == "*" and text == "":
        ret_type = api_definition.get_pointer_type(
            api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension))
    elif tail == "**" and text == "":
        ret_type = api_definition.get_pointer_type(api_definition.get_pointer_type(
            api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)))
    elif tail == "* const*" and text == "const":
        ret_type = api_definition.get_const_pointer_type(api_definition.get_const_type(
            api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension))))
    elif tail == "":
        ret_type = api_definition.get_or_add_type(
            vk, tp, added_by_version, added_by_extension)
    else:
        error("Dont know whats up with this tail")

    array_len = None
    name = node.find('name')
    if name != None and name.tail != None:
        if name.tail[0] == '[':
            en = node.find('enum')
            if en != None:
                if en.tail != "]":
                    error("Expected an enum name")
                array_len = en.text
            else:
                if name.tail[-1] != ']':
                    error("Expected an array size")
                array_len = name.tail[1: -1]
    if array_len != None:
        if not array_len.isnumeric():
            api_definition.add_named_constant(vk, array_len)
        return api_definition.get_array_type(ret_type, array_len)

    return ret_type


class basetype(base):
    def __init__(self, name, t, api_definition):
        base.__init__(self, name)
        self.base_type = None
        self.is_struct = False

        if t.find('type') != None:
            self.base_type = api_definition.types[t.find('type').text]
        if t.text and t.text.lstrip().rstrip() == "struct":
            self.is_struct = True

    def __str__(self):
        return f"{self.name}[{self.base_type}]'"


class handle(base):
    DISPATCHABLE = 0
    NON_DISPATCHABLE = 1

    def __init__(self, name, t):
        base.__init__(self, name)

    def __str__(self):
        if self.dispatch == self.DISPATCHABLE:
            return f"{self.name}:DISPATCH"
        else:
            return f"{self.name}:NON_DISPATCH"

    def has_handle(self):
        return True


class enum(base):
    ENUM = 0
    BITMASK = 1

    def __init__(self, name, t, vk, api_def):
        base.__init__(self, name)
        nm = name
        self.values = {}
        if 'type' in t:
            self.base_type = api_def.types[t.find('type').text]
        else:
            self.base_type = api_def.types['uint32_t']

        for x in vk.findall(f'./enums/[@name="{nm}"]/enum'):
            if 'bitpos' in x.attrib:
                self.values[x.attrib['name']] = (
                    1 << int(x.attrib['bitpos'], 0), self.BITMASK)
            elif 'value' in x.attrib:
                self.values[x.attrib['name']] = (
                    int(x.attrib['value'], 0), self.ENUM)

    def __str__(self):
        tp = "enum"
        x = [x for x in self.values.values() if x[1] == self.BITMASK]
        if len(x) != 0:
            tp = "bits"
        y = f"{self.name} {tp}[{self.base_type.name}]'\n"
        for x in self.values.items():
            y += f"        {x[0]}: {x[1][0]}\n"
        return y


class funcpointer(base):
    def __init__(self, name, t):
        base.__init__(self, name)
        self.typedef = "".join(t.itertext())


class command_param():
    def __init__(self):
        self.no_auto_validity = False

    def __str__(self):
        return f"{self.type.short_str()} {self.name}"

    def short_str(self):
        return f"{self.type.get_member(self.name)}"


def rf():
    return False


def rt():
    return True


def check_pnext(subparam):
    return subparam.type.has_pnext()


class command(base):
    def __init__(self, name, vk, api_definition):
        base.__init__(self, name)
        self.args = []
        self.ret = None
        cmd = vk.findall(f'./commands/command/proto/[name="{name}"]/..')
        if len(cmd) != 1:
            cmd = vk.findall(f'./commands/command/[@name="{name}"]')
            if len(cmd) != 1:
                error(f"Found more than one command with the name {name}")
        cmd = cmd[0]
        if 'alias' in cmd.attrib:
            name = cmd.attrib['alias']
            cmd = vk.findall(f'./commands/command/proto/[name="{name}"]/..')
            if len(cmd) != 1:
                error(f"Found more than one command with the name {name}")
            cmd = cmd[0]
        proto = cmd.find('proto')
        params = cmd.findall('param')
        return_type = proto.find('type')
        if return_type == None or not return_type.text in api_definition.types:
            error(f'Could not find return type for {name}')
        self.ret = api_definition.types[return_type.text]
        for p in params:
            subparam = command_param()
            subparam.name = p.find('name').text
            subparam.type = get_subtype(
                p, vk, api_definition, self.added_by_version, self.added_by_extension)
            subparam.inout = rf
            if type(subparam.type) == pointer_type and not subparam.type.const:
                subparam.inout = partial(check_pnext, subparam)

            if 'optional' in p.attrib:
                subparam.optional = p.attrib['optional'] == 'true'
            else:
                subparam.optional = False

            subparam.no_auto_validity = 'noautovalidity' in p.attrib

            if 'len' in p.attrib:
                subparam.len = p.attrib['len']
                for x in self.args:
                    if x.name == subparam.len:
                        x.inout = rt
            else:
                subparam.len = None
            self.args.append(subparam)

    def __str__(self):
        prms = [str(x) for x in self.args]
        for i in range(0, len(self.args)):
            if self.args[i].optional:
                prms[i] += "{optional}"
            if self.args[i].len:
                prms[i] += f"{{len={self.args[i].len}}}"
        return f"{self.ret.short_str()} {self.name}({', '.join(prms)})"

    def short_str(self):
        prms = [x.short_str() for x in self.args]
        return f"{self.ret.short_str()} {self.name}({', '.join(prms)})"


class struct_param():
    def __init__(self):
        self.extended_by = []

    def __str__(self):
        return f"        {self.name} == {self.type.short_str()}"

    def has_handle(self):
        if self.type.has_handle():
            return True
        for x in self.extended_by:
            if x[1].has_handle():
                return True
        return False


class serialization_arg():
    def __init__(self, name, type):
        self.name = name
        self.type = type

    def __str__(self):
        return f'{self.type} {self.name}'


class serialization_param():
    def __init__(self, type, argname, ret):
        self.type = type
        self.args = []
        self.argname = argname
        self.ret = ret

    def signature(self):
        return f'{self.ret} _{self.argname}_{self.type}({", ".join([x.__str__() for x in self.args])})'

    def function(self):
        return f'std::function<{self.ret}({", ".join([x.__str__() for x in self.args])})> _{self.argname}_{self.type}'

    def name(self):
        return f'_{self.argname}_{self.type}'


class deserialization_param():
    def __init__(self, type, argname, ret):
        self.type = type
        self.args = []
        self.argname = argname
        self.ret = ret

    def signature(self):
        return f'{self.ret} _{self.argname}_{self.type}({", ".join([x.__str__() for x in self.args])})'

    def function(self):
        return f'std::function<{self.ret}({", ".join([x.__str__() for x in self.args])})> _{self.argname}_{self.type}'

    def name(self):
        return f'_{self.argname}_{self.type}'


class struct(base):
    def __init__(self, name, t):
        base.__init__(self, name)
        self.t = t
        self.members = []

    def setup(self, vk, api_definitions):
        for m in self.t.findall('member'):
            subparam = struct_param()
            subparam.name = m.find('name').text
            subparam.type = get_subtype(
                m, vk, api_definitions, self.added_by_version, self.added_by_extension)
            if 'optional' in m.attrib:
                subparam.optional = m.attrib['optional'] == 'true'
            else:
                subparam.optional = False
            if 'len' in m.attrib:
                subparam.len = m.attrib['len']
            else:
                subparam.len = None
            if 'noautovalidity' in m.attrib:
                subparam.noautovalidity = True if m.attrib['noautovalidity'] == 'true' else False
            else:
                subparam.noautovalidity = False
            self.members.append(subparam)

    def finalize(self, vk, api_definitions):
        if 'structextends' in self.t.attrib:
            extends = self.t.attrib['structextends'].split(',')
            member = self.t.find('member[1]')
            enum = member.attrib['values'].split(',')
            for e in extends:
                if e in api_definitions.types:
                    tp = api_definitions.types[e]
                    for en in enum:
                        tp.members[1].extended_by.append((en, self))

    def __str__(self):
        str = self.name + "\n"
        for i in self.members:
            str += f"        {i.type.short_str()} {i.name}"
            if (i.optional):
                str += " {optional} "
            if (i.len):
                str += f" {{len={i.len}}}"
            str += "\n"
        return str

    def has_handle(self):
        for x in self.members:
            if x.type.has_handle():
                return True
            xt = x.type
            while type(xt) == const_type:
                xt = xt.child
            if xt == self:
                return False
            if type(xt) == pointer_type and xt.pointee.name == 'void':
                for eb in x.extended_by:
                    if eb[1].has_handle():
                        return True
            # prevent infinite loops for VkBaseInStructure
            if type(xt) == pointer_type and xt.pointee == self:
                return False
            if(xt.has_handle()):
                return True
        return False

    def has_pnext(self):
        for x in self.members:
            if x.name == 'pNext' and len(x.extended_by) != 0:
                return True
        return False

    def get_serialization_params(self,  _serialization_type=SERIALIZE):
        p = []
        for x in self.members:
            if x.type.get_noncv_type() == self:
                continue
            xt = x.type
            while type(xt) == const_type:
                xt = xt.child
            if type(xt) == pointer_type and xt.pointee.name == 'void':
                if x.name == 'pNext':
                    for eb in x.extended_by:
                        new_params = eb[1].get_serialization_params(
                            _serialization_type)
                        for x in new_params:
                            x.argname = f'{self.name}_{x.argname}'
                            oldargs = x.args
                            x.args = [serialization_arg(
                                'self', f'const {self.name}&')]
                            for a in oldargs:
                                a.name = "_" + a.name
                                x.args.append(a)
                        p.extend(new_params)
                    continue
                if _serialization_type == CLONE:
                    a = serialization_param(
                        "clone", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg(
                        'src', f'const {self.name}&'))
                    a.args.append(serialization_arg('dst', f'{self.name}&'))
                    a.args.append(serialization_arg(
                        'mem', f'temporary_allocator*'))
                elif _serialization_type == FIX_HANDLES:
                    a = serialization_param(
                        "fix_handles", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg('val', f'{self.name}&'))
                    a.args.append(serialization_arg(
                        'mem', f'temporary_allocator*'))
                elif _serialization_type == DESERIALIZE:
                    a = serialization_param(
                        "deserialize", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg('self', f'{self.name}&'))
                    a.args.append(serialization_arg('dec', f'decoder*'))
                else:
                    a = serialization_param(
                        "serialize", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg(
                        'self', f'const {self.name}&'))
                    a.args.append(serialization_arg('enc', f'encoder*'))
                p.append(a)
            if x.noautovalidity and not _serialization_type == DESERIALIZE:
                a = serialization_param(
                    "valid", f'{self.name}_{x.name}', 'bool')
                a.args.append(serialization_arg('self', f'const {self.name}&'))
                p.append(a)
            if x.len and x.len.startswith('latexmath') and not _serialization_type == DESERIALIZE:
                a = serialization_param(
                    'length', f'{self.name}_{x.name}', 'uint64_t')
                a.args.append(serialization_arg('self', f'const {self.name}&'))
                p.append(a)
            child_params = x.type.get_noncv_type().get_serialization_params(_serialization_type)
            for x in child_params:
                x.argname = f'{self.name}_{x.argname}'
                oldargs = x.args
                x.args = [serialization_arg('self', f'const {self.name}&')]
                for a in oldargs:
                    a.name = "_" + a.name
                    x.args.append(a)
            p.extend(child_params)
        used = set()
        return [x for x in p if x.signature() not in used and (used.add(x.signature()) or True)]


class constant:
    def __init__(self):
        self.type = None
        self.name = None
        self.value = None


class union(struct):
    def __init__(self, name, t):
        struct.__init__(self, name, t)


def sort_extensions(vk, extensions):
    enabled_extensions = []
    for x in extensions:
        if x in enabled_extensions:
            continue
        ext = vk.find(f'./extensions/extension/[@name="{x}"]')
        if ext == None:
            error(f"Could not find extension {x}")

        if 'requires' in ext.attrib:
            required = ext.attrib['requires'].split(',')
            for y in required:
                if not y in extensions:
                    error(f'Error extension "{x}" requires "{y}"')
                if not y in enabled_extensions:
                    enabled_extensions.append(y)
        enabled_extensions.append(x)
    return enabled_extensions


class api_definition:
    def get_sorted_structs(self):
        return sorted_structs(self)

    def get_or_add_type(self, vk, tn, version, extension):
        if tn in self.types:
            return self.types[tn]
        self.add_type(vk, tn, version, extension)
        return self.types[tn]

    def add_type(self, vk, tn, version, extension):
        t = vk.findall(f'./types/type/[@name="{tn}"]')
        if len(t) == 0:
            t = vk.findall(f'./types/type/[name="{tn}"]')
        if len(t) == 0:
            error(f"Error could not find type {tn}")
        if len(t) > 1:
            error(f"Error could not find type {tn}")
        t = t[0]
        aliased = ""
        if 'alias' in t.attrib:
            aliased = t.attrib['alias']
            t = vk.findall(f'./types/type/[@name="{aliased}"]')
            if len(t) == 0:
                t = vk.findall(f'./types/type/[name="{aliased}"]')
            if len(t) == 0:
                error(f"Error could not find type {tn}")
            if len(t) > 1:
                error(f"Error could not find type {tn}")
            t = t[0]
        tp = None
        if aliased != "":
            if not aliased in self.types:
                self.add_type(vk, aliased, version, extension)
            tp = alias(tn, self.types[aliased])
        elif 'category' in t.attrib:
            category = t.attrib['category']
            if category == 'basetype':
                tp = basetype(tn, t, self)
            elif category == 'bitmask':
                tp = enum(tn, t, vk, self)
            elif category == 'handle':
                tp = handle(tn, t)
                if t.find('type').text == "VK_DEFINE_HANDLE":
                    tp.dispatch = tp.DISPATCHABLE
                elif t.find('type').text == "VK_DEFINE_NON_DISPATCHABLE_HANDLE":
                    tp.dispatch = tp.NON_DISPATCHABLE
            elif category == 'enum':
                tp = enum(tn, t, vk, self)
            elif category == 'funcpointer':
                tp = funcpointer(tn, t)
            elif category == 'struct':
                tp = struct(tn, t)
            elif category == 'include':
                return None
            elif category == 'define':
                return None
            elif category == 'union':
                tp = union(tn, t)
            else:
                error(f"No idea what the category is {category}")
        elif 'name' in t.attrib:
            tp = external_type(t.attrib['name'], t)
        else:
            error("What!!?!")
        if tp:
            if not tn in self.types:
                tp.added_by_version = version
                tp.added_by_extension = extension
                self.types[tn] = tp
            else:
                #print(f"Adding a type that already exists {tn}")
                pass

    def __init__(self, vk, max_version, extensions):
        extensions = sort_extensions(vk, extensions)
        self.types = {}
        self.commands = {}
        self.types = {x.attrib['name']: platform_type(
            x.attrib['name']) for x in vk.findall(f'./types/type/[@requires="vk_platform"]')}
        self.constants = {}
        self.add_versions(vk, max_version)
        for ext in extensions:
            self.add_extension(vk, ext, extensions)
        for x in self.types.values():
            if type(x) == struct:
                x.finalize(vk, self)
        verify_unique_commands(self)

    def add_versions(self, vk, max_version):
        for version in vk.findall('./feature'):
            if float(version.attrib['number']) <= max_version:
                self.add_version(vk, version.attrib['number'])

    def add_named_constant(self, vk, name):
        if name in self.constants:
            return
        const = vk.find(f'./enums/enum/[@name="{name}"]')
        if const == None:
            error(f"Could not find constant {name}")
        if 'alias' in const.attrib:
            old_const = self.constants[const.attrib['alias']]
            new_const = constant()
            new_const.type = old_const.type
            new_const.name = name
            new_const.value = const.attrib['alias']
            self.constants[name] = new_const
            return
        new_const = constant()
        new_const.type = self.types[const.attrib['type']]
        if new_const.type == None:
            error(f"Could not find type for constant {name} {new_const.type}")
        new_const.value = const.attrib['value']
        new_const.name = name
        self.constants[name] = new_const

    def add_constant(self, vk, ee):
        name = ee.attrib['name']
        if name in self.constants:
            #print(f"Adding constant {name} again")
            return
        nm = name
        if 'alias' in ee.attrib:
            return
        if 'value' in ee.attrib:
            new_const = constant()
            new_const.type = self.types['uint32_t']
            new_const.value = ee.attrib['value']
            new_const.name = name
            self.constants[name] = new_const
            return
        const = vk.find(f'./enums/enum/[@name="{nm}"]')
        if const == None:
            error(f"Could not find constant {name}")
        if 'alias' in const.attrib:
            old_const = self.constants[const.attrib['alias']]
            new_const = constant()
            new_const.type = old_const.type
            new_const.name = name
            new_const.value = const.attrib['alias']
            self.constants[name] = new_const
            return
        new_const = constant()
        new_const.type = self.types[const.attrib['type']]
        if new_const.type == None:
            error(f"Could not find type for constant {name} {new_const.type}")
        new_const.value = const.attrib['value']
        new_const.name = name
        self.constants[name] = new_const

    def extend_enum(self, ee, ext_num):
        base_enum = ee.attrib['extends']
        if not base_enum in self.types:
            error("error trying to extend an enum that has not been included")
        be = self.types[base_enum]

        if type(be) == alias:
            be = alias.tgt
        if type(be) != enum:
            error("error trying to extend an enum that is not an enum")
        if 'bitpos' in ee.attrib:
            be.values[ee.attrib['name']] = (
                1 << int(ee.attrib['bitpos'], 0), be.BITMASK)
        elif 'value' in ee.attrib:
            be.values[ee.attrib['name']] = (
                int(ee.attrib['value'], 0),  be.ENUM)
        elif 'offset' in ee.attrib:
            if 'number' in ee.attrib:
                val = int(ee.attrib['offset'], 0) + 1000000000 + \
                    (int(ee.attrib['extnumber'], 0) - 1) * 1000
            else:
                val = int(ee.attrib['offset'], 0) + \
                    1000000000 + (ext_num - 1) * 1000
            if 'dir' in ee.attrib and ee.attrib['dir'] == "-":
                val = -val
            be.values[ee.attrib['name']] = (val, be.ENUM)
        elif 'alias' in ee.attrib:
            be.values[ee.attrib['name']] = be.values[ee.attrib['alias']]
        else:
            error("Error dont know how to extend this enum")

    def add_command(self, vk, ee, version, extension):
        newcmd = command(ee.attrib['name'], vk, self)
        newcmd.added_by_version = version
        newcmd.added_by_extension = extension
        self.commands[ee.attrib['name']] = newcmd

    def add_version(self, vk, version):
        types = []
        types = [tp.attrib["name"] for tp in vk.findall(
            f'./feature/[@number="{version}"]/require/type')]

        for tn in types:
            self.add_type(vk, tn, version, None)
        structs = []
        for x in self.types.values():
            if (type(x) == struct or type(x) == union) and x.added_by_version == version:
                structs.append(x)
        i = 0
        for x in structs:
            x.setup(vk, self)
            i = i + 1
        for ee in vk.findall(f'./feature/[@number="{version}"]/require/enum'):
            if ('extends' in ee.attrib):
                extnumber = -1
                if 'extnumber' in ee.attrib:
                    extnumber = int(ee.attrib['extnumber'], 0)
                self.extend_enum(ee, extnumber)
            else:
                self.add_constant(vk, ee)
        for ee in vk.findall(f'./feature/[@number="{version}"]/require/command'):
            self.add_command(vk, ee, version, None)

    def add_extension(self, vk, name, enabled):
        types = []
        for req in vk.findall(f'./extensions/extension/[@name="{name}"]/require'):
            if 'extension' in req.attrib:
                if not req.attrib['extension'] in enabled:
                    continue
            ext = req.findall(f'type')
            for t in ext:
                types.append(t.attrib['name'])
        extnum = int(
            vk.find(f'./extensions/extension/[@name="{name}"]').attrib['number'], 0)
        for tn in types:
            self.add_type(vk, tn, None, name)
        structs = []
        for x in self.types.values():
            if (type(x) == struct or type(x) == union) and x.added_by_extension == name:
                structs.append(x)
        i = 0
        for x in structs:
            x.setup(vk, self)
            i = i + 1

        for req in vk.findall(f'./extensions/extension/[@name="{name}"]/require'):
            if 'extension' in req.attrib:
                if not (req.attrib['extension'] in enabled):
                    continue
            ext = req.findall(f'enum')
            for ee in ext:
                if ('extends' in ee.attrib):
                    self.extend_enum(ee, extnum)
                else:
                    self.add_constant(vk, ee)
        for req in vk.findall(f'./extensions/extension/[@name="{name}"]/require'):
            if 'extension' in req.attrib:
                if not (req.attrib['extension'] in enabled):
                    continue
            ext = req.findall(f'command')
            for ee in ext:
                self.add_command(vk, ee, None, name)

    def get_pointer_type(self, type):
        if type.short_str() + "*" in self.types:
            return self.types[type.short_str() + "*"]
        else:
            ptrtype = pointer_type(type, False)
            ptrtype.added_by_version = type.added_by_version
            self.types[type.short_str() + "*"] = ptrtype
            return ptrtype

    def get_const_pointer_type(self, type):
        if type.short_str() + " const*" in self.types:
            return self.types[type.short_str() + " const*"]
        else:
            ptrtype = pointer_type(type, True)
            ptrtype.added_by_version = type.added_by_version
            self.types[type.short_str() + " const*"] = ptrtype
            return ptrtype

    def get_const_type(self, type):
        if "C_" + type.short_str() in self.types:
            return self.types["C_" + type.short_str()]
        else:
            consttype = const_type(type)
            consttype.added_by_version = type.added_by_version
            self.types["C_" + type.short_str()] = consttype
            return consttype

    def get_array_type(self, type, length):
        if type.short_str() + f"ArrayOf{length}" in self.types:
            return self.types[type.short_str() + f"ArrayOf{length}"]
        else:
            arraytype = array_type(type, length)
            arraytype.added_by_version = type.added_by_version
            self.types[type.short_str() + f"ArrayOf{length}"] = arraytype
            return arraytype

    def print(self):
        print("Api Definition:")
        for t in self.types.values():
            if type(t) != external_type:
                print(f'    {t} was added by {t.added_by()}')
        for c in self.commands.values():
            print(f'    {c} was added by {c.added_by()}')


def load_vulkan(args):
    vk = ET.parse(args.vulkan_xml)
    return vk.getroot()


def sorted_structs(definition):
    structs = [x for x in definition.types.values() if type(x) ==
               struct or type(x) == union]
    output_structs = []
    for x in structs:
        if x in output_structs:
            continue
        to_check_members = deque(x.members)
        while len(to_check_members) != 0:
            m = to_check_members[0]
            mt = m.type.get_noncv_type()
            appended = False
            for e in m.extended_by:
                if not e[1] in output_structs and not e[1] == mt:
                    sp = struct_param()
                    sp.name = ""
                    sp.type = e[1]
                    to_check_members.appendleft(sp)
                    appended = True
            if appended:
                continue
            if mt in output_structs:
                to_check_members.popleft()
                continue
            if not type(mt) == struct and not type(mt) == union:
                to_check_members.popleft()
                continue
            appended = False
            for z in mt.members:
                bt = z.type.get_noncv_type()
                if (type(bt) == struct or type(bt) == union) and not bt in output_structs and not bt == x:
                    to_check_members.appendleft(z)
                    appended = True
                for e in z.extended_by:
                    if not e[1] in output_structs and not e[1] == mt:
                        sp = struct_param()
                        sp.name = ""
                        sp.type = e[1]
                        to_check_members.appendleft(sp)
                        appended = True

            if appended:
                continue
            output_structs.append(mt)
            to_check_members.popleft()
        if not x in output_structs:
            output_structs.append(x)
    return output_structs
