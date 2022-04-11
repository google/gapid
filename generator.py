import os
import xml.etree.ElementTree as ET
import argparse
import sys
from collections import deque
import copy
import hashlib
from functools import partial
def error(string):
    print(string)
    sys.exit(-1)

SERIALIZE = 0
DESERIALIZE = 1
CLONE = 2
FIX_HANDLES = 3

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
        ret_type = api_definition.get_const_type(api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)))
    elif tail == "*" and text == "const struct":
        ret_type = api_definition.get_const_type(api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)))
    elif tail == "*" and text == "struct":
        ret_type = api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension))
    elif tail == "*" and text == "":
        ret_type = api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension))
    elif tail == "**" and text == "":
        ret_type = api_definition.get_pointer_type(api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)))
    elif tail == "* const*" and text == "const":
        ret_type = api_definition.get_const_pointer_type(api_definition.get_const_type(api_definition.get_pointer_type(api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension))))
    elif tail == "":
        ret_type = api_definition.get_or_add_type(vk, tp, added_by_version, added_by_extension)
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
                self.values[x.attrib['name']] = (1 << int(x.attrib['bitpos'], 0), self.BITMASK)
            elif 'value' in x.attrib:
                self.values[x.attrib['name']] = (int(x.attrib['value'], 0), self.ENUM)

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
            subparam.type = get_subtype(p, vk, api_definition, self.added_by_version, self.added_by_extension)
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
        return f'std::function<{self.ret} ({", ".join([x.__str__() for x in self.args])})> _{self.argname}_{self.type}'

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
        return f'std::function<{self.ret} ({", ".join([x.__str__() for x in self.args])})> _{self.argname}_{self.type}'

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
            subparam.type = get_subtype(m, vk, api_definitions, self.added_by_version, self.added_by_extension)
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
                        new_params = eb[1].get_serialization_params(_serialization_type)
                        for x in new_params:
                            x.argname = f'{self.name}_{x.argname}'
                            oldargs = x.args
                            x.args = [serialization_arg('self', f'const {self.name}&')]
                            for a in oldargs:
                                a.name = "_" + a.name
                                x.args.append(a)
                        p.extend(new_params)
                    continue
                if _serialization_type == CLONE:
                    a = serialization_param("clone", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg('src', f'const {self.name}&'))
                    a.args.append(serialization_arg('dst', f'{self.name}&'))
                    a.args.append(serialization_arg('mem', f'temporary_allocator*'))
                elif _serialization_type == FIX_HANDLES:
                    a = serialization_param("fix_handles", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg('val', f'{self.name}&'))
                    a.args.append(serialization_arg('mem', f'temporary_allocator*'))
                elif _serialization_type == DESERIALIZE:
                    a = serialization_param("deserialize", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg('self', f'{self.name}&'))
                    a.args.append(serialization_arg('dec', f'decoder*'))
                else:
                    a = serialization_param("serialize", f'{self.name}_{x.name}', 'void')
                    a.args.append(serialization_arg('self', f'const {self.name}&'))
                    a.args.append(serialization_arg('enc', f'encoder*'))
                p.append(a)
            if x.noautovalidity and type(xt) == pointer_type and not _serialization_type == DESERIALIZE:
                a = serialization_param("valid", f'{self.name}_{x.name}', 'bool')
                a.args.append(serialization_arg('self', f'const {self.name}&'))
                p.append(a)
            if x.len and x.len.startswith('latexmath') and not _serialization_type == DESERIALIZE:
                a = serialization_param('length', f'{self.name}_{x.name}', 'uint64_t')
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
        return p

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
                print(f"Adding a type that already exists {tn}")

    def __init__(self, vk, max_version, extensions):    
        extensions = sort_extensions(vk, extensions)    
        self.types = {}
        self.commands = {}
        self.types = {x.attrib['name']: platform_type(x.attrib['name']) for x in  vk.findall(f'./types/type/[@requires="vk_platform"]')}
        self.constants = {}
        self.add_versions(vk, max_version)
        for ext in extensions:
            self.add_extension(vk, ext, extensions)
        for x in self.types.values():
            if type(x) == struct:
                x.finalize(vk, self)

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
            print(f"Adding constant {name} again")
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
            be.values[ee.attrib['name']] = (1 << int(ee.attrib['bitpos'], 0), be.BITMASK)
        elif 'value' in ee.attrib:
            be.values[ee.attrib['name']] = (int(ee.attrib['value'], 0),  be.ENUM)
        elif 'offset' in ee.attrib:
            if 'number' in ee.attrib:
                val = int(ee.attrib['offset'], 0) + 1000000000 + (int(ee.attrib['extnumber'], 0) - 1)* 1000
            else:
                val = int(ee.attrib['offset'], 0) + 1000000000 + (ext_num - 1)* 1000
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
        types = [tp.attrib["name"] for tp in vk.findall(f'./feature/[@number="{version}"]/require/type')]
        
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
        extnum = int(vk.find(f'./extensions/extension/[@name="{name}"]').attrib['number'], 0)
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


def get_sorted_structs(definition):
    structs = [x for x in definition.types.values() if type(x) == struct or type(x) == union]
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

def output_member_dec(x, struct_name, memberid, idx, depth, vop, fdec):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            print(depth + f"uint32_t pNext{memberid} = 0;", file=fdec)
            print(depth + f"{vop}{x.name}{idx} = nullptr;", file=fdec)
            print(depth + f"const void** ppNext{memberid} = const_cast<const void**>(&{vop}{x.name}{idx});", file=fdec)
            print(depth + f"while ((pNext{memberid} = dec->decode<uint32_t>())) {{", file=fdec)
            depth = depth + "  "
            print(depth + f"switch(pNext{memberid}) {{", file=fdec)
            for y in x.extended_by:
                print(depth + f"  case {y[0]}: {{", file=fdec)
                print(depth + f"    {y[1].name}* pn = dec->get_typed_memory<{y[1].name}>(1);", file=fdec)
                print(depth + f"    deserialize_{y[1].name}(updater, *pn, dec);", file=fdec)
                print(depth + f"    *ppNext{memberid} = pn;", file=fdec)
                print(depth + f"    ppNext{memberid} = const_cast<const void**>(&pn->pNext);", file=fdec)
                print(depth + f"    break;", file=fdec)
                print(depth + f"  }}", file=fdec)
            if (struct_name == 'VkDeviceCreateInfo'):
                print(depth + f"  case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO:", file=fdec)
                print(depth + f"    break;", file=fdec)
            print(depth + f"  default:", file=fdec)
            print(depth + f'    GAPID2_ERROR("Unexpected pNext");', file=fdec)
            print(depth + f"}}", file=fdec)
            depth = depth[:-2]
            print(depth + f"}}", file=fdec)
        else:
            print(depth + 'if (dec->decode<uint32_t>()) { GAPID2_ERROR("Unexpected pNext"); }', file=fdec)    
            print(depth + f'{vop}{x.name}{idx} = nullptr;', file=fdec)    
    elif type(tp) == union:
        print(depth + f"_custom_deserialize_{tp.name}(updater, {vop}{x.name}{idx}, dec);", file=fdec)
    elif type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "dec"]
        prms.extend([f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params(DESERIALIZE)])
        print(depth + f"deserialize_{tp.name}(updater, {', '.join(prms)});", file=fdec)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"dec->decode<{enc_type}>(&{vop}{x.name}{idx});", file=fdec)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"dec->decode<{enc_type}>(&{vop}{x.name}{idx});", file=fdec)
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["val", "dec"]
            print(depth + f"_{struct_name}_{x.name}_deserialize({', '.join(prms)});", file=fdec)
            return
        
        
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")

            #decode
            print(depth + f"uint64_t length{memberid} = dec->decode<uint64_t>();", file=fdec)
            print(depth + f"if (length{memberid}) {{", file=fdec)
            print(depth + f"    char* tmp_ = static_cast<char*>(dec->get_memory(length{memberid}));", file=fdec)
            print(depth + f"    dec->decode_primitive_array(tmp_, length{memberid});", file=fdec)
            print(depth + f"    {vop}{x.name}{idx} = tmp_;", file=fdec)
            print(depth + f"}} else {{", file=fdec)
            print(depth + f"    {vop}{x.name}{idx} = nullptr;", file=fdec)
            print(depth + f"}}", file=fdec)
            return

        ct = "1"
        if x.optional or  x.noautovalidity:
            print(depth + f"if(dec->decode<char>()) {{", file=fdec)
            depth = depth + "  "

        if x.len:
            ct = f'temp_len{memberid}'
            print(depth + f"uint64_t {ct} = dec->decode<uint64_t>();", file=fdec)

        tmp = f'temp{memberid}'
        print(depth + f"{tp.get_non_const_member(tmp)}  = dec->get_typed_memory<{tp.pointee.get_non_const_member('')[0:-1]}>({ct});", file=fdec)

        mem_idx = f'[0]'
        if x.len:
            ii = len(depth)
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {ct}; ++i_{ii}) {{", file=fdec)
            depth += "  "
            mem_idx = f'[i_{ii}]'
        
        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        xm.name = tmp
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_member_dec(xm, struct_name, memberid + 1, mem_idx, depth, f"", fdec)

        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=fdec)
        print(depth + f"{vop}{x.name}{idx} = {tmp};", file=fdec)
        if x.optional or x.noautovalidity:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fdec)
            print(depth + f"  {vop}{x.name}{idx} = nullptr;", file=fdec)
            print(depth + f"}}", file=fdec)
        
    elif type(tp) == enum:
        print(depth + f"dec->decode<{tp.base_type.name}>(&{vop}{x.name}{idx});", file=fdec)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"dec->decode<uint64_t>(reinterpret_cast<uintptr_t*>(&{vop}{x.name}{idx}));", file=fdec)
        else:
            print(depth + f"dec->decode<uint64_t>(reinterpret_cast<uint64_t*>(&{vop}{x.name}{idx}));", file=fdec)
    elif type(tp) == array_type:
        ii = len(depth)
        print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{", file=fdec)
        depth += "  "
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_member_dec(xm, struct_name, memberid + 1, f"[i_{ii}]", depth, vop, fdec)
        print(depth + f"}}", file=fdec)
def output_member_enc(x, struct_name, memberid, idx, depth, vop, fenc):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            print(depth + f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);", file=fenc)
            print(depth + f"while(baseStruct) {{", file=fenc)
            print(depth + f"  enc->template encode<uint32_t>(baseStruct->sType);", file=fenc)
            print(depth + f"  switch(baseStruct->sType) {{", file=fenc)
            for y in x.extended_by:
                print(depth + f"    case {y[0]}:", file=fenc)
                print(depth + f"      if (baseStruct->pNext != nullptr) {{", file=fenc)
                print(depth + f"        {y[1].name} _tmp = *reinterpret_cast<const {y[1].name}*>(baseStruct);", file=fenc)
                print(depth + f"        _tmp.pNext = nullptr;", file=fenc)
                prms = [f"_tmp", "enc"]
                prms.extend([z.name() for z in y[1].get_serialization_params()])
                print(depth + f"        serialize_{y[1].name}(updater, {', '.join(prms)});", file=fenc)
                print(depth + f"      }} else {{", file=fenc)
                prms = [f"*reinterpret_cast<const {y[1].name}*>(baseStruct)", "enc"]
                prms.extend([z.name() for z in y[1].get_serialization_params()])
                print(depth + f"        serialize_{y[1].name}(updater, {', '.join(prms)});", file=fenc)
                print(depth + f"      }}", file=fenc)
                print(depth + f"      break;", file=fenc)
            if (struct_name == 'VkDeviceCreateInfo'):
                print(depth + f"    case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO:", file=fenc)
                print(depth + f"      break;", file=fenc)
            print(depth + f"     default:", file=fenc)
            print(depth + f'      GAPID2_ERROR("Unexpected pNext");', file=fenc)
            print(depth + f"  }}", file=fenc)
            print(depth + f"  baseStruct = baseStruct->pNext;", file=fenc)
            print(depth + f"}}", file=fenc)
            print(depth + f"enc->template encode<uint32_t>(0); // No more pNext", file=fenc)
        elif (struct_name == "VkInstanceCreateInfo"):
            print(depth + f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);", file=fenc)
            print(depth + f"while(baseStruct) {{", file=fenc)
            print(depth + f"  switch(baseStruct->sType) {{", file=fenc)
            print(depth + f"    case VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO:", file=fenc)
            print(depth + f"      break;", file=fenc)
            print(depth + f"    default:", file=fenc)
            print(depth + f'      GAPID2_ERROR("Unexpected pNext");', file=fenc)
            print(depth + f"  }}", file=fenc)
            print(depth + f"  baseStruct = baseStruct->pNext;", file=fenc)
            print(depth + f"}}", file=fenc)
            print(depth + f"enc->template encode<uint32_t>(0); // No more pNext", file=fenc)
        else:
            print(depth + f'if({vop}pNext) {{ GAPID2_ERROR("Unexpected pNext"); }}', file=fenc)
            print(depth + f"enc->template encode<uint32_t>(0); // pNext", file=fenc)
    elif type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend([z.name() for z in tp.get_serialization_params()])

        print(depth + f"_custom_serialize_{tp.name}(updater, {', '.join(prms)});", file=fenc)
    elif type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend([f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params()])
        print(depth + f"serialize_{tp.name}(updater, {', '.join(prms)});", file=fenc)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"enc->template encode<{enc_type}>({vop}{x.name}{idx});", file=fenc)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"enc->template encode<{enc_type}>({vop}{x.name}{idx});", file=fenc)
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["val", "enc"]
            print(depth + f"_{struct_name}_{x.name}_serialize({', '.join(prms)});", file=fenc)
            return
        
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            #encode
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fenc)
            print(depth + f"  uint64_t len = strlen({vop}{x.name}{idx});", file=fenc)
            print(depth + f"  enc->template encode<uint64_t>(len + 1);", file=fenc)
            print(depth + f"  enc->template encode_primitive_array<char>({vop}{x.name}{idx}, len + 1);", file=fenc)
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  enc->template encode<uint64_t>(0);", file=fenc)
            print(depth + f"}}", file=fenc)
            return

        if x.noautovalidity:
            print(depth + f"if (_{struct_name}_{x.name}_valid(val)) {{", file=fenc)
            print(depth + f"  enc->template encode<char>(1);", file=fenc)
            depth = depth + "  "
        elif x.optional:
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fenc)
            print(depth + f"  enc->template encode<char>(1);", file=fenc)
            depth = depth + "  "

        mem_idx = f'{idx}[0]'
        if x.len:
            ll = f"{vop}{x.len.split(',')[0]}"
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                ll = f"_{struct_name}_{x.name}_length(val)"
            print(depth + f"enc->template encode<uint64_t>({ll}); // array_len", file=fenc)
            ii = len(depth)
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {ll}; ++i_{ii}) {{", file=fenc)
            depth += "  "
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_member_enc(xm, struct_name, memberid + 1, mem_idx, depth, f"{vop}", fenc)
            
        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=fenc)
        if x.noautovalidity:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  enc->template encode<char>(0);", file=fenc)
            print(depth + f"}}", file=fenc)
        elif x.optional:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  enc->template encode<char>(0);", file=fenc)
            print(depth + f"}}", file=fenc)

    elif type(tp) == enum:
        print(depth + f"enc->template encode<{tp.base_type.name}>({vop}{x.name}{idx});", file=fenc)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"enc->template encode<uint64_t>(reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));", file=fenc)
        else:
            print(depth + f"enc->template encode<uint64_t>(reinterpret_cast<uint64_t>({vop}{x.name}{idx}));", file=fenc)
    elif type(tp) == array_type:
        ii = len(depth)
        print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{", file=fenc)
        depth += "  "
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_member_enc(xm, struct_name, memberid + 1, f"[i_{ii}]", depth, vop, fenc)
        print(depth + f"}}", file=fenc)

def output_member_clone(x, xdst, struct_name, memberid, idx, depth, vop, dop, fclone):
    tp = x.type
    dt = xdst.type
    while(type(tp) == const_type):
        tp = tp.child
    while(type(dt) == const_type):
        dt = dt.child
    if x.name == "pNext":
        if len(x.extended_by):
            print(depth + f"auto srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>({vop}pNext);", file=fclone)
            print(depth + f"void* dstBaseStructBase = nullptr;", file=fclone)
            print(depth + f"auto dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&dstBaseStructBase);", file=fclone)
            print(depth + f"while(srcBaseStruct) {{", file=fclone)
            print(depth + f"    switch(srcBaseStruct->sType) {{", file=fclone)
            for y in x.extended_by:
                print(depth + f"        case {y[0]}: {{", file=fclone)
                print(depth + f"          {y[1].name}* _val = mem->get_typed_memory<{y[1].name}>(1);", file=fclone)
                print(depth + f"          if (srcBaseStruct->pNext != nullptr) {{", file=fclone)
                print(depth + f"            auto srctmp_ = *reinterpret_cast<const {y[1].name}*>(srcBaseStruct);", file=fclone)
                print(depth + f"            srctmp_.pNext = nullptr;", file=fclone)
                prms = [f"srctmp_", "*_val", "mem"]
                prms.extend([z.name() for z in y[1].get_serialization_params(CLONE)])
                print(depth + f"            clone<HandleUpdater>(updater, {', '.join(prms)});", file=fclone)
                print(depth + f"            *dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);", file=fclone)
                print(depth + f"          }} else {{", file=fclone)
                prms = [f"*reinterpret_cast<const {y[1].name}*>(srcBaseStruct)", "*_val", "mem"]
                prms.extend([z.name() for z in y[1].get_serialization_params(CLONE)])
                print(depth + f"            clone<HandleUpdater>(updater, {', '.join(prms)});", file=fclone)
                print(depth + f"            *dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);", file=fclone)
                print(depth + f"          }}", file=fclone)
                print(depth + f"          break;", file=fclone)
                print(depth + f"        }}", file=fclone)
            if (struct_name == 'VkDeviceCreateInfo'):
                print(depth + f"        case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO: {{", file=fclone)
                print(depth + f"          VkLayerDeviceCreateInfo* _val = mem->get_typed_memory<VkLayerDeviceCreateInfo>(1);", file=fclone)
                print(depth + f"          memcpy(_val, srcBaseStruct, sizeof(VkLayerDeviceCreateInfo));", file=fclone)
                print(depth + f"          *dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);", file=fclone)
                print(depth + f"          break;", file=fclone)
                print(depth + f"        }}", file=fclone)
            print(depth + f"        default:", file=fclone)
            print(depth + f'            GAPID2_ERROR("Unexpected pNext");', file=fclone)
            print(depth + f"    }}", file=fclone)
            print(depth + f"    dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&((*dstBaseStruct)->pNext));", file=fclone)
            print(depth + f"    srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>(srcBaseStruct->pNext);", file=fclone)
            print(depth + f"}}", file=fclone)
            print(depth + f"{dop}pNext = dstBaseStructBase;", file=fclone)
        elif (struct_name == "VkInstanceCreateInfo"):
            print(depth + f"auto srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>({vop}pNext);", file=fclone)
            print(depth + f"void* dstBaseStructBase = nullptr;", file=fclone)
            print(depth + f"auto dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&dstBaseStructBase);", file=fclone)
            print(depth + f"while(srcBaseStruct) {{", file=fclone)
            print(depth + f"  switch(srcBaseStruct->sType) {{", file=fclone)
            print(depth + f"    case VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO: {{", file=fclone)
            print(depth + f"          VkLayerInstanceCreateInfo* _val = mem->get_typed_memory<VkLayerInstanceCreateInfo>(1);", file=fclone)
            print(depth + f"          memcpy(_val, srcBaseStruct, sizeof(VkLayerInstanceCreateInfo));", file=fclone)
            print(depth + f"          *dstBaseStruct = reinterpret_cast<VkBaseOutStructure*>(_val);", file=fclone)
            print(depth + f"          break;", file=fclone)
            print(depth + f"    }}", file=fclone)
            print(depth + f"    default:", file=fclone)
            print(depth + f'      GAPID2_ERROR("Unexpected pNext");', file=fclone)
            print(depth + f"  }}", file=fclone)
            print(depth + f"  dstBaseStruct = reinterpret_cast<VkBaseOutStructure**>(&((*dstBaseStruct)->pNext));", file=fclone)
            print(depth + f"  srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>(srcBaseStruct->pNext);", file=fclone)
            print(depth + f"}}", file=fclone)
            print(depth + f"{dop}pNext = dstBaseStructBase;", file=fclone)
        else:
            print(depth + f'if({vop}pNext) {{ GAPID2_ERROR("Unexpected pNext"); }}', file=fclone)
            print(depth + f"{dop}pNext = nullptr;", file=fclone)
    elif type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", f"{dop}{xdst.name}{idx}", "mem"]
        prms.extend([z.name() for z in tp.get_serialization_params(CLONE)])
        print(depth + f"_custom_clone_{tp.name}<HandleUpdater>(updater, {', '.join(prms)});", file=fclone)
    elif type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", f"{dop}{xdst.name}{idx}", "mem"]
        prms.extend([f'bind_first(_{struct_name}{z.name()}, src)' for z in tp.get_serialization_params(CLONE)])
        print(depth + f"clone<HandleUpdater>(updater, {', '.join(prms)});", file=fclone)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};", file=fclone)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};", file=fclone)
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = ["src", "dst", "mem"]
            print(depth + f"_{struct_name}_{x.name}_clone({', '.join(prms)});", file=fclone)
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            #encode
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fclone)
            print(depth + f"    uint64_t _len = strlen({vop}{x.name}{idx});", file=fclone)
            print(depth + f"    {{", file=fclone)
            print(depth + f"        char* __tmp = mem->get_typed_memory<char>(_len+1);", file=fclone)
            print(depth + f"        memcpy(__tmp, {vop}{x.name}{idx}, _len+1);", file=fclone)
            print(depth + f"        {dop}{xdst.name}{idx} = __tmp;", file=fclone)
            print(depth + f"    }}", file=fclone)
            print(depth + f"}} else {{", file=fclone)
            print(depth + f"    {dop}{xdst.name}{idx} = nullptr;", file=fclone)
            print(depth + f"}}", file=fclone)
            return
        ct = "1"
        if x.noautovalidity:
            print(depth + f"if (_{struct_name}_{x.name}_valid(src)) {{", file=fclone)
            depth = depth + "  "
        elif x.optional:
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fclone)
            depth = depth + "  "        

        mem_idx = f'{idx}[0]'
        if x.len:
            ll = f"{vop}{x.len.split(',')[0]}"
            ct = ll
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                print(depth + f"uint64_t _len{memberid} = _{struct_name}_{x.name}_length(src);", file=fclone)
                ct = f"_len{memberid}"
            
        tmp = f'temp{memberid}'
        print(depth + f"{tp.get_non_const_member(tmp)}  = mem->get_typed_memory<{tp.pointee.get_non_const_member('')[0:-1]}>({ct});", file=fclone)

        if x.len:
            ii = len(depth)
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {ct}; ++i_{ii}) {{", file=fclone)
            depth += "  "
            mem_idx = f'{idx}[i_{ii}]'        


        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        xm.name = f'{vop}{xm.name}'
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        x2 = copy.deepcopy(xdst)
        x2.type = dt.pointee
        x2.name = tmp

        output_member_clone(xm, x2, struct_name, memberid + 1, mem_idx, depth, f"", f"", fclone)
        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=fclone)
        print(depth + f"{dop}{xdst.name}{idx} = {tmp};", file=fclone)
        if x.optional or x.noautovalidity:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fclone)
            print(depth + f"  {dop}{xdst.name}{idx} = nullptr;", file=fclone)
            print(depth + f"}}", file=fclone)
    elif type(tp) == enum:
        print(depth + f"{dop}{xdst.name}{idx} = {vop}{x.name}{idx};", file=fclone)
    elif type(tp) == handle:
        print(depth + f"{dop}{xdst.name}{idx} = updater->cast_in({vop}{x.name}{idx});", file=fclone)
    elif type(tp) == array_type:
        ii = len(depth)
        print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{", file=fclone)
        depth += "  "
        xm = copy.deepcopy(x)
        xm.type = tp.child
        x2 = copy.deepcopy(xdst)
        x2.type = dt.child
        output_member_clone(xm, x2, struct_name, memberid + 1, f"[i_{ii}]", depth, vop, dop, fclone)
        print(depth + f"}}", file=fclone)
def output_member_fix(x, struct_name, memberid, idx, depth, vop, ffix):
    if not x.has_handle():
        return
    tp = x.type
    
    while(type(tp) == const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            print(depth + f"auto srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>({vop}pNext);", file=ffix)
            print(depth + f"while(srcBaseStruct) {{", file=ffix)
            print(depth + f"    switch(srcBaseStruct->sType) {{", file=ffix)
            for y in x.extended_by:
                print(depth + f"        case {y[0]}: {{", file=ffix)
                print(depth + f"            break;", file=ffix)
                print(depth + f"        }}", file=ffix)
            print(depth + f"        default:", file=ffix)
            print(depth + f'            GAPID2_ERROR("Unexpected pNext");', file=ffix)
            print(depth + f"    }}", file=ffix)
            print(depth + f"    srcBaseStruct = reinterpret_cast<const VkBaseOutStructure*>(srcBaseStruct->pNext);", file=ffix)
            print(depth + f"}}", file=ffix)
        else:
            print(depth + f'if({vop}pNext) {{ GAPID2_ERROR("Unexpected pNext"); }}', file=ffix)
    elif type(tp) == union:
        pass
    elif type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}"]
        prms.extend([f'bind_first(_{struct_name}{z.name()}, src)' for z in tp.get_serialization_params(FIX_HANDLES)])
        print(depth + f"fix_handles({', '.join(prms)});", file=ffix)
    elif type(tp) == basetype:
        pass
    elif type(tp) == platform_type:
        pass
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            print(depth + f"// _{struct_name}_{x.name} hopefully doesn't have any handles :)")
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            return
        ct = "1"
        if x.noautovalidity:
            print(depth + f"if (_{struct_name}_{x.name}_valid(src)) {{", file=ffix)
            depth = depth + "  "
        elif x.optional:
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=ffix)
            depth = depth + "  "        

        mem_idx = f'{idx}[0]'
        if x.len:
            ll = f"{vop}{x.len.split(',')[0]}"
            ct = ll
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                print(depth + f"uint64_t _len{memberid} = _{struct_name}_{x.name}_length(src);", file=ffix)
                ct = f"_len{memberid}"
            
        tmp = f'temp{memberid}'
        
        if x.len:
            ii = len(depth)
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {ct}; ++i_{ii}) {{", file=ffix)
            depth += "  "
            mem_idx = f'{idx}[i_{ii}]'        


        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        xm.name = f'{vop}{xm.name}'
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        
        output_member_fix(xm, struct_name, memberid + 1, mem_idx, depth, f"", ffix)
        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=ffix)
        if x.optional or x.noautovalidity:
            depth = depth[:-2]
            print(depth + f"}}", file=ffix)
    elif type(tp) == enum:
        pass
    elif type(tp) == handle:
        print(depth + f"  {vop}{x.name}{idx} = cast_out({vop}{x.name}{idx});", file=ffix)

def output_struct_enc_dec(x, fenc, fdec):
    
    memberid = 0

    serialize_params = [f"const {x.name}& val", "encoder* enc"]
    serialize_params.extend([y.function() for y in x.get_serialization_params()])
    print(f"template<typename HandleUpdater>", file=fenc)
    print(f"inline void serialize_{x.name}(HandleUpdater* updater, {', '.join(serialize_params)}) {{", file=fenc)

    deserialize_params = [f"{x.name}& val", "decoder* dec"]
    deserialize_params.extend([y.function() for y in x.get_serialization_params(DESERIALIZE)])
    
    print(f"template<typename HandleUpdater>", file=fdec)
    print(f"inline void deserialize_{x.name}(HandleUpdater* updater, {', '.join(deserialize_params)}) {{", file=fdec)
    for y in x.members:
        output_member_enc(y, x.name, memberid, "", "  ", "val.", fenc)
        output_member_dec(y, x.name, memberid, "", "  ", "val.", fdec)
        memberid += 1
    print("}\n", file=fenc)
    print("}\n", file=fdec)

def output_struct_clone(x, fclone):
    memberid = 0
    serialize_params = ["HandleUpdater* updater", f"const {x.name}& src", f"{x.name}& dst", "temporary_allocator* mem"]
    serialize_params.extend([y.function() for y in x.get_serialization_params(CLONE)])
    print(f"template<typename HandleUpdater>", file=fclone)
    print(f"inline void clone({', '.join(serialize_params)}) {{", file=fclone)
    
    for y in x.members:
        output_member_clone(y, y, x.name, memberid, "", "    ", "src.", "dst.", fclone)
        memberid += 1
    print("}\n", file=fclone)

def output_union_clone(x, fclone):
    pass

def output_union_enc_dec(x, fenc, fdec):
    pass

def output_structs_enc_dec(definition, dir, structs_to_handle = None):
    with open(os.path.join(dir, "struct_serialization.h"), mode="w") as fenc:
        with open(os.path.join(dir, "struct_deserialization.h"), mode="w") as fdec:
            print("#pragma once", file=fenc)
            print("#pragma once", file=fdec)
        
            print('#include <functional>', file=fenc)
            print('#include "encoder.h"', file=fenc)
            print('#include <vulkan.h>', file=fenc)
            print('#include "helpers.h"', file=fenc)
            print('#include "forwards.h"', file=fenc)
            print('namespace gapid2 {', file=fenc)

            print('#include <functional>', file=fdec)
            print('#include <vulkan.h>', file=fdec)
            print('#include "decoder.h"', file=fdec)
            print('#include "helpers.h"', file=fdec)
            print('#include "forwards.h"', file=fdec)
            print('namespace gapid2 {', file=fdec)
            for x in get_sorted_structs(definition):
                if structs_to_handle != None:
                    if not x in structs_to_handle:
                        continue
                if type(x) == union:
                    output_union_enc_dec(x, fenc, fdec)
                elif type(x) == struct:
                    output_struct_enc_dec(x, fenc, fdec)
            print('}', file=fenc)
            print('}', file=fdec)



def output_print_member(x, struct_name, memberid, idx, depth, vop, fprint):
    nm = f"\"{x.name}\""
    if idx != "" and idx != "[0]":
        nm = "\"\""
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child
    if x.name == "pNext":
        if len(x.extended_by):
            print(depth + f'printer->begin_array("pNext");', file=fprint)
            print(depth + f"auto baseStruct = reinterpret_cast<const VkBaseInStructure*>({vop}pNext);", file=fprint)
            print(depth + f"while(baseStruct) {{", file=fprint)
            print(depth + f"  switch(baseStruct->sType) {{", file=fprint)
            for y in x.extended_by:
                print(depth + f"    case {y[0]}:", file=fprint)
                print(depth + f"      if (baseStruct->pNext != nullptr) {{", file=fprint)
                print(depth + f"        {y[1].name} _tmp = *reinterpret_cast<const {y[1].name}*>(baseStruct);", file=fprint)
                print(depth + f"        _tmp.pNext = nullptr;", file=fprint)
                prms = [f"_tmp", "printer"]
                prms.extend([z.name() for z in y[1].get_serialization_params()])
                print(depth + f"        print_{y[1].name}(\"\", updater, {', '.join(prms)});", file=fprint)
                print(depth + f"      }} else {{", file=fprint)
                prms = [f"*reinterpret_cast<const {y[1].name}*>(baseStruct)", "printer"]
                prms.extend([z.name() for z in y[1].get_serialization_params()])
                print(depth + f"        print_{y[1].name}(\"\", updater, {', '.join(prms)});", file=fprint)
                print(depth + f"      }}", file=fprint)
                print(depth + f"      break;", file=fprint)
            if (struct_name == 'VkDeviceCreateInfo'):
                print(depth + f"    case VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO:", file=fprint)
                print(depth + f"      break;", file=fprint)
            print(depth + f"     default:", file=fprint)
            print(depth + f'      GAPID2_ERROR("Unexpected pNext");', file=fprint)
            print(depth + f"  }}", file=fprint)
            print(depth + f"  baseStruct = baseStruct->pNext;", file=fprint)
            print(depth + f"}}", file=fprint)
            print(depth + f'printer->end_array();', file=fprint)
        else:
            print(depth + f'if({vop}pNext) {{ GAPID2_ERROR("Unexpected pNext"); }}', file=fprint)
            print(depth + f'printer->begin_array("pNext");', file=fprint)
            print(depth + f'printer->end_array();', file=fprint)
    elif type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", "printer"]
        prms.extend([z.name() for z in tp.get_serialization_params()])
        print(depth + f"// ignoring union for now", file=fprint)
        #print(depth + f"_custom_print{tp.name}({nm}, updater, {', '.join(prms)});", file=fprint)
    elif type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "printer"]
        prms.extend([f'bind_first(_{struct_name}{z.name()}, val)' for z in tp.get_serialization_params()])
        print(depth + f"print_{tp.name}({nm}, updater, {', '.join(prms)});", file=fprint)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"printer->print<{enc_type}>({nm}, {vop}{x.name}{idx});", file=fprint)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"printer->print<{enc_type}>({nm}, {vop}{x.name}{idx});", file=fprint)
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            print(depth + f"// Ignoring nullptr for now", file=fprint)
            #prms = ["val", "printer"]
            #print(depth + f"_{struct_name}_{x.name}_print({nm}, {', '.join(prms)});", file=fprint)
            return
        
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            print(depth + f"printer->print_string({nm}, {vop}{x.name}{idx});", file=fprint)
            return

        if x.noautovalidity:
            print(depth + f"if (_{struct_name}_{x.name}_valid(val)) {{", file=fprint)
            depth = depth + "  "
        elif x.optional:
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fprint)
            depth = depth + "  "

        mem_idx = f'{idx}[0]'
        if x.len:
            print(depth + f"printer->begin_array({nm});", file=fprint)
            ll = f"{vop}{x.len.split(',')[0]}"
            # Special case for strings
            if x.len.startswith('latexmath'):
                prms = ["val"]
                ll = f"_{struct_name}_{x.name}_length(val)"
            ii = len(depth)
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {ll}; ++i_{ii}) {{", file=fprint)
            depth += "  "
            mem_idx = f'{idx}[i_{ii}]'
            

        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_print_member(xm, struct_name, memberid + 1, mem_idx, depth, f"{vop}", fprint)
            
        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=fprint)
            print(depth + f"printer->end_array();", file=fprint)
        if x.noautovalidity:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fprint)
            print(depth + f"  printer->print<void*>({nm}, nullptr);", file=fprint)
            print(depth + f"}}", file=fprint)
        elif x.optional:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fprint)
            print(depth + f"  printer->print<void*>({nm}, nullptr);", file=fprint)
            print(depth + f"}}", file=fprint)

    elif type(tp) == enum:
        print(depth + f"printer->print<{tp.base_type.name}>({nm}, {vop}{x.name}{idx});", file=fprint)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"printer->print<uint64_t>({nm}, reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));", file=fprint)
        else:
            print(depth + f"printer->print<uint64_t>({nm}, reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));", file=fprint)
    elif type(tp) == array_type:
        if tp.get_noncv_type().name == 'char':
            print(depth + f"printer->print_char_array({nm}, {vop}{x.name}{idx}, {tp.size});", file=fprint)
            return    
        print(depth + f"printer->begin_array({nm});", file=fprint)
        ii = len(depth)
        print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {tp.size}; ++i_{ii}) {{", file=fprint)
        depth += "  "
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_print_member(xm, struct_name, memberid + 1, f"[i_{ii}]", depth, vop, fprint)
        depth = depth[:-2]
        print(depth + f"}}", file=fprint)
        print(depth + f"printer->end_array();", file=fprint)

def output_print_union(x, fprint):
    pass

def output_print_struct(x, fprint):
    memberid = 0

    serialize_params = [f"const {x.name}& val", "Printer* printer"]
    serialize_params.extend([y.function() for y in x.get_serialization_params()])
    print(f"template<typename HandleUpdater, typename Printer>", file=fprint)
    print(f"inline void print_{x.name}(const char* name, HandleUpdater* updater, {', '.join(serialize_params)}) {{", file=fprint)
    print(f"  printer->begin_object(name);", file=fprint)
    deserialize_params = [f"{x.name}& val", "decoder* dec"]
    deserialize_params.extend([y.function() for y in x.get_serialization_params(DESERIALIZE)])

    for y in x.members:
        output_print_member(y, x.name, memberid, "", "  ", "val.", fprint)
        memberid += 1
    print(f"  printer->end_object();", file=fprint)
    print("}\n", file=fprint)

def output_struct_printer(definition, dir, structs_to_handle = None):
    with open(os.path.join(dir, "struct_printer.h"), mode="w") as fprint:
            print("#pragma once", file=fprint)
        
            print('#include <functional>', file=fprint)
            print('#include "encoder.h"', file=fprint)
            print('#include <vulkan.h>', file=fprint)
            print('#include "helpers.h"', file=fprint)
            print('#include "forwards.h"', file=fprint)
            print('namespace gapid2 {', file=fprint)
            for x in get_sorted_structs(definition):
                if structs_to_handle != None:
                    if not x in structs_to_handle:
                        continue
                if type(x) == union:
                    output_print_union(x, fprint)
                elif type(x) == struct:
                    output_print_struct(x, fprint)
            print('}', file=fprint)

def output_structs_clone(definition, dir, structs_to_handle = None):
    with open(os.path.join(dir, "struct_clone.h"), mode="w") as fclone:
            print("#pragma once", file=fclone)
        
            print('#include <functional>', file=fclone)
            print('#include "encoder.h"', file=fclone)
            print('#include <vulkan.h>', file=fclone)
            print('#include "helpers.h"', file=fclone)
            print('#include "forwards.h"', file=fclone)
            print('#include "temporary_allocator.h"', file=fclone)
            print('namespace gapid2 {', file=fclone)

            for x in get_sorted_structs(definition):
                if structs_to_handle != None:
                    if not x in structs_to_handle:
                        continue
                if type(x) == union:
                    output_union_clone(x, fclone)
                elif type(x) == struct:
                    output_struct_clone(x, fclone)
            print('}', file=fclone)

def output_arg_ptr(param, tp, ct):
    prms = ["&updater_", param.name, ct,  "&clone_allocator_"]
    prms.extend([z.name() for z in tp.get_serialization_params(CLONE)])
    if (type(param.type.get_noncv_type()) == handle):
        return f"clone_handle<HandleUpdater>({', '.join(prms)})"
    else:
        return f"clone_struct<HandleUpdater>({', '.join(prms)})"
        

def output_arg_convert(cmd, param):
    tp = param.type
    while type(tp) == const_type:
        tp = tp.child
    if param.no_auto_validity and type(tp) == pointer_type and tp.pointee.name == 'void':
        args = [x.name for x in cmd.args]
        return f"_custom_unwrap_{cmd.name}_{param.name}(&updater_, &clone_allocator_, {', '.join(args)})"
    if not tp.has_handle():
        return param.name
    if type(tp) == handle:
        return f"updater_.cast_in({param.name})"
    if type(tp) == pointer_type:
        if type(param.type) != const_type:
            # Return pointer dont need to copy
            return param.name
        ct = "1"
        if param.len:
            ct = param.len

        return output_arg_ptr(param, tp.pointee, ct)
    error("We got here")

def output_arg_create(cmd, param, file):
    tp = param.type
    if not tp.has_handle():
        return
    if type(tp) != pointer_type:
        return
    if type(tp) == pointer_type:
        if type(param.type) == const_type:
            return
        pt = param.type.pointee.name
        if cmd.name == "vkCreateInstance":
            prms = ['&updater_', param.name]
            print(f"    create_instance<HandleUpdater, {pt}Wrapper<HandleUpdater>>({', '.join(prms)});", file=file)
            return
        prms = ['&updater_', cmd.args[0].name, param.name]
        zn = cmd.args[0].type.name
        if param.len:
            prms.append(param.len)
        else:
            prms.append("1")
        if type(tp.pointee) == struct:
            print(f"    create_handle_from_struct<HandleUpdater>({', '.join(prms)});", file=file)
        else:
            print(f"    create_handle<HandleUpdater, {zn}, {pt}, {pt}Wrapper<HandleUpdater>>({', '.join(prms)});", file=file)

def output_arg_enc(cmd, x, vop, arg_idx, idx, depth, fenc):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child
  
    if type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend([x.name() for x in tp.get_serialization_params()])

        print(depth + f"_custom_serialize_{tp.name}(updater, {', '.join(prms)});", file=fenc)
    if type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "enc"]
        prms.extend([x.name() for x in tp.get_serialization_params()])

        print(depth + f"serialize_{tp.name}(updater, {', '.join(prms)});", file=fenc)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"enc->template encode<{enc_type}>({vop}{x.name}{idx});", file=fenc)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"enc->template encode<{enc_type}>({vop}{x.name}{idx});", file=fenc)
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = [x.name for x in cmd.args]
            prms.append("enc")
            print(depth + f"_custom_serialize_{cmd.name}_{x.name}(updater, {', '.join(prms)});", file=fenc)
            return
        # Special case for strings
        if x.len and x.len.startswith('latexmath'):
            print(depth + f"// Latexmath string", file=fenc)
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            #encode
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fenc)
            print(depth + f"  uint64_t len = strlen({vop}{x.name}{idx});", file=fenc)
            print(depth + f"  enc->template encode<uint64_t>(len + 1);", file=fenc)
            print(depth + f"  enc->template encode_primitive_array<char>({vop}{x.name}{idx}, len + 1);", file=fenc)
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  enc->template encode<uint64_t>(0);", file=fenc)
            print(depth + f"}}", file=fenc)
            return

        if x.optional:
            print(depth + f"if ({vop}{x.name}{idx}) {{", file=fenc)
            print(depth + f"  enc->template encode<char>(1);", file=fenc)
            depth = depth + "  "

        mem_idx = f'{idx}[0]'
        if x.len:
            ii = len(depth)
            argct = f"{vop}{x.len.split(',')[0]}"
            p = [x for x in cmd.args if x.name == argct]
            if len(p):
                if type(p[0].type) == pointer_type:
                    argct = f"*{argct}"
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {argct}; ++i_{ii}) {{", file=fenc)
            depth += "  "
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_arg_enc(cmd, xm, vop, arg_idx, mem_idx, depth, fenc)
            
        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=fenc)
        if x.optional:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  enc->template encode<char>(0);", file=fenc)
            print(depth + f"}}", file=fenc)

    elif type(tp) == enum:
        print(depth + f"enc->template encode<{tp.base_type.name}>({vop}{x.name}{idx});", file=fenc)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"enc->template encode<uint64_t>(reinterpret_cast<uintptr_t>({vop}{x.name}{idx}));", file=fenc)
        else:
            print(depth + f"enc->template encode<uint64_t>(reinterpret_cast<uint64_t>({vop}{x.name}{idx}));", file=fenc)
    elif type(tp) == array_type:
        print(depth + f"for (size_t i = 0; i < {tp.size}; ++i) {{", file=fenc)
        depth = depth + "  "
        mem_idx = f'{idx}[i]'
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_arg_enc(cmd, xm, vop, arg_idx, mem_idx, depth, fenc)
        depth = depth[:-2]
        print(depth + f"}}", file=fenc)

def output_command(cmd, definition, fbod, only_return = False):
    print(f"  {cmd.short_str()} override {{", file=fbod)
    # Special case. Anything that can unblock the CPU we have to actually cause 
    # block around call. Otherwise the returns might encode out of order
    # which means we deadlock on replay.
    if (cmd.name == "vkSignalSemaphore" or cmd.name == "vkSignalSemaphoreKHR" or cmd.name == "vkSetEvent" or cmd.name == "vkQueueSubmit"):
        print(f"    auto enc = get_locked_encoder();", file=fbod)
    else:
        print(f"    auto enc = get_encoder();", file=fbod)
    sha = int.from_bytes(hashlib.sha256(cmd.name.encode('utf-8')).digest()[:8], 'little')
    print(f"    enc->template encode<uint64_t>({sha}u);", file=fbod)
    if not only_return:
        arg_idx = 0
        for arg in cmd.args:
            if arg.name == 'pAllocator':
                print(f"    // Skipping: {arg.name} for as it cannot be replayed", file=fbod)
                continue
            if type(arg.type) == pointer_type and not arg.type.const:
                if arg.inout():
                    print(f"    // Inout: {arg.name}", file=fbod)
                else:
                    print(f"    // Skipping: {arg.name} for now as it is an output param", file=fbod)
                    continue
            output_arg_enc(cmd, arg, "", arg_idx, "",  "    ", fbod)
            arg_idx += 1
    args = ", ".join(x.name for x in cmd.args)
    if (cmd.ret.name == 'void'):
        print(f"    T::{cmd.name}({args});", file=fbod)
    else:
        print(f"    const auto ret = T::{cmd.name}({args});", file=fbod)

    arg_idx = 0
    for arg in cmd.args:
        if type(arg.type) == pointer_type and not arg.type.const:
            if arg.inout():
                print(f"    // Inout value: {arg.name}", file=fbod) 
            else:
                print(f"    // Return value: {arg.name}", file=fbod)
            
        else:
            continue
        output_arg_enc(cmd, arg, "", arg_idx, "",  "    ", fbod)
        arg_idx += 1
    if cmd.ret.name != "void":
        if cmd.ret.name == "VkResult":
            print(f"    enc->template encode<uint32_t>(ret);", file=fbod)
        print(f"    return ret;", file=fbod)
    print(f"  }}", file=fbod)

def output_commands(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "commands.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include "struct_clone.h"', file=fbod)
        print('#include "struct_serialization.h"', file=fbod)
        print('#include "helpers.h"', file=fbod)
        print('#include "forwards.h"', file=fbod)
        print('#include "command_processor.h"', file=fbod)
        print('#include <iostream>', file=fbod)
        
        print(
'''
namespace gapid2 {
template<typename T>
class CommandSerializer: public T {
  static_assert(std::is_base_of<CommandProcessor, T>::value, "Expected T to be derived from CommandProcessor");
  public:
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            output_command(cmd, definition, fbod)
        print(
'''
    virtual encoder_handle get_encoder() = 0;
    virtual encoder_handle get_locked_encoder() = 0;
};
} // namespace gapid2
''', file=fbod)

def output_returns_serializer(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "return_serializer.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include "struct_clone.h"', file=fbod)
        print('#include "struct_serialization.h"', file=fbod)
        print('#include "helpers.h"', file=fbod)
        print('#include "forwards.h"', file=fbod)
        print('#include "command_processor.h"', file=fbod)
        print('#include <iostream>', file=fbod)
        
        print(
'''
namespace gapid2 {
template<typename T>
class ReturnSerializer: public T {
  static_assert(std::is_base_of<CommandProcessor, T>::value, "Expected T to be derived from CommandProcessor");
  public:
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            output_command(cmd, definition, fbod, True)
        print(
'''
    virtual encoder_handle get_locked_encoder() = 0;
    virtual encoder_handle get_encoder() = 0;
};
} // namespace gapid2
''', file=fbod)


def output_command_header(cmd, definition, fbod):
    print(f"  {cmd.short_str()};", file=fbod)
    pass

def output_command_forward(cmd, definition, fbod):
    print(f"  {cmd.short_str()} {{", file=fbod)
    args = ", ".join(x.name for x in cmd.args)
    print(f"    return spy()->{cmd.name}({args});", file=fbod)
    print(f"  }}", file=fbod)

def output_call_forwards(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "call_forwards.cpp"), mode="w") as fbod:
        with open(os.path.join(dir, "call_forwards.h"), mode="w") as fcall:
            print("#pragma once", file=fcall)
            print('#include <vulkan.h>', file=fcall)
            print('#include "call_forwards.h"', file=fbod)
            print('#include "layer_setup.h"', file=fcall)
            print('#include "spy.h"', file=fcall)
            print(
'''
namespace gapid2 {
''', file=fbod)
            print(
'''

namespace gapid2 {
''', file=fcall)

            for cmd in definition.commands.values():
                if commands_to_handle:
                    if not cmd in commands_to_handle:
                        continue
                output_command_header(cmd, definition, fcall)
                output_command_forward(cmd, definition, fbod)
            print(
'''
  PFN_vkVoidFunction get_instance_function(const char* name) {
''', file=fbod)
            for cmd in definition.commands.values():
                if commands_to_handle:
                    if not cmd in commands_to_handle:
                        continue
                print(f"    if (!strcmp(name, \"{cmd.name}\")) {{", file=fbod)
                print(f"      return (PFN_vkVoidFunction) {cmd.name};", file=fbod)
                print(f"    }}", file=fbod)
            print(
'''
    return nullptr;
  }
  PFN_vkVoidFunction get_device_function(const char* name) {
''', file=fbod)
            for cmd in definition.commands.values():
                if commands_to_handle:
                    if not cmd in commands_to_handle:
                        continue
                if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue':
                    continue
                print(f"    if (!strcmp(name, \"{cmd.name}\")) {{", file=fbod)
                print(f"      return (PFN_vkVoidFunction) {cmd.name};", file=fbod)
                print(f"    }}", file=fbod)
            print(
'''
    return nullptr;
  }
''', file=fbod)
            print(
'''
} // namespace gapid2
''', file=fbod)
            print(
'''
  PFN_vkVoidFunction get_instance_function(const char* name);
  PFN_vkVoidFunction get_device_function(const char* name);
} // namespace gapid2
''', file=fcall)

def output_command_processor(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "command_processor.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include "helpers.h"', file=fbod)
        
        print(
'''
namespace gapid2 {
class CommandProcessor {
  public:
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            ret = ""
            if (cmd.ret.name != 'void'):
                ret = f"return {cmd.ret.short_str()}();"

            print(f"  virtual {cmd.short_str()} {{{ret}}};", file=fbod)
        print(
'''
};
} // namespace gapid2
''', file=fbod)

def output_command_caller(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "command_caller.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include "helpers.h"', file=fbod)
        print('#include "command_processor.h"', file=fbod)
        print('#include "struct_clone.h"', file=fbod)
        print('#include "forwards.h"', file=fbod)
        print('#include <vulkan.h>', file=fbod)

        print(
'''
namespace gapid2 {
template<typename HandleUpdater>
class CommandCaller : public CommandProcessor {
  public:
    using Updater = HandleUpdater;
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            print(f"  {cmd.short_str()} override {{", file=fbod)
            print(f"    temporary_allocator clone_allocator_;", file=fbod)
            
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and  cmd.args[0].name != 'queue' and cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                print(f"    // ----- Special ----- ", file=fbod)
                print(f"    const auto fn = _{cmd.name};", file=fbod)
            else:
                print(f"    const auto fn = updater_.cast_from_vk({cmd.args[0].name})->_functions->{cmd.name};", file=fbod)
            
            args = ", ".join(output_arg_convert(cmd, x) for x in cmd.args)
            if (cmd.ret.name == 'void'):
                print(f"    fn({args});", file=fbod)
            else:
                print(f"    const auto ret = fn({args});", file=fbod)
            for a in cmd.args:
                output_arg_create(cmd, a, fbod)

            if (cmd.ret.name != 'void'):            
                print(f"    return ret;", file=fbod)
            print(f"  }}", file=fbod)
        for cmd in definition.commands.values():
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and  cmd.args[0].name != 'queue' and cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                print(f"  PFN_{cmd.name} _{cmd.name};", file=fbod)
        print(
'''
  public:
  HandleUpdater updater_;
  HandleUpdater* updater = &updater_;
};
} // namespace gapid2
''', file=fbod)

def print_arg_ptr(cmd, param, tp, count, depth, fprint):
    nm = f"{param.name}"
    tp = tp.get_noncv_type()
    idx = "[0]"


    p = [x for x in cmd.args if x.name == count]
    if len(p):
        if type(p[0].type) == pointer_type:
            count = f"*{count}"

    if count != "1":
        print(depth + f"printer->begin_array(\"{nm}\");", file=fprint)
        print(depth + f"for (size_t i = 0; i < {count}; ++i) {{", file=fprint)
        idx = "[i]"
        nm = ""
        depth += "  "
    if tp.name == "void":
        print(depth + f"// Ignoring void for now", file=fprint)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"printer->print<uint64_t>(\"{nm}\", reinterpret_cast<uintptr_t>({param.name}{idx}));", file=fprint)
        else:
            print(depth + f"printer->print<uint64_t>(\"{nm}\", reinterpret_cast<uintptr_t>({param.name}{idx}));", file=fprint)
    elif type(tp) == struct:
        prms = [f"\"{nm}\"", f"updater", f"{param.name}{idx}", "printer"]
        prms.extend([x.name() for x in tp.get_serialization_params()])
        print(depth + f"print_{tp.name}({', '.join(prms)});", file=fprint)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"printer->print<{enc_type}>(\"{nm}\", {param.name}{idx});", file=fprint)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"printer->print<{enc_type}>(\"{nm}\", {param.name}{idx});", file=fprint)
    elif type(tp) == union:
        print(depth + f"// Ignoring union for now", file=fprint)
    elif type(tp) == enum:
        print(depth + f"printer->print<{tp.base_type.name}>(\"{nm}\", {param.name}{idx});", file=fprint)
    else:
        error(f'Error printing {param.name} type: {tp.name}')
    if count != "1":
        depth = depth[:-2]
        print(depth + f"}}", file=fprint)
        print(depth + f"printer->end_array();", file=fprint)
    
def output_arg_print(cmd, param, depth, fprint):
    if param.name == "pAllocator":
        return
    tp = param.type
    nm = f"{param.name}"
    while type(tp) == const_type:
        tp = tp.child
    if type(tp) == enum:
        print(depth + f"printer->print<{tp.base_type.name}>(\"{nm}\", {nm});", file=fprint)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"printer->print<uint64_t>(\"{nm}\", reinterpret_cast<uintptr_t>({nm}));", file=fprint)
        else:
            print(depth + f"printer->print<uint64_t>(\"{nm}\", reinterpret_cast<uintptr_t>({nm}));", file=fprint)
    elif type(tp) == basetype:
        enc_type = tp.base_type
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"printer->print<{enc_type}>(\"{nm}\", {nm});", file=fprint)
    elif type(tp) == platform_type:
        enc_type = str(tp)
        if enc_type == "size_t":
            enc_type = "uint64_t"
        print(depth + f"printer->print<{enc_type}>(\"{nm}\", {nm});", file=fprint)
    elif type(tp) == pointer_type:
        if param.len and param.len == 'null-terminated':
            if tp.pointee.get_noncv_type().name != 'char':
                error("Non-char null terminated type")
            print(depth + f"printer->print_string(\"{nm}\", {nm});", file=fprint)
            return
        if param.optional:
            print(depth + f"if ({nm}) {{", file=fprint)
            depth = depth + "  "
        ct = "1"
        if param.len:
            ct = param.len
        print_arg_ptr(cmd, param, tp.pointee, ct, depth, fprint)
        if param.optional:
            depth = depth[:-2]
            print(depth + f"}} else {{ ", file=fprint)
            print(depth + f"  printer->print_null(\"{nm}\");", file=fprint)
            print(depth + f"}}", file=fprint)
        
    elif type(tp) == array_type:
        print_arg_ptr(cmd, param, tp.get_noncv_type(), f'{tp.size}', depth, fprint)
    else:
        error(f'Error printing {param.name} type: {tp.name}')

def output_command_printer(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "command_printer.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include "helpers.h"', file=fbod)
        print('#include "command_processor.h"', file=fbod)
        print('#include "struct_clone.h"', file=fbod)
        print('#include "forwards.h"', file=fbod)
        print('#include "struct_printer.h"', file=fbod)
        print('#include <vulkan.h>', file=fbod)

        print(
'''
namespace gapid2 {
template<typename Printer, typename T>
class CommandPrinter : public T {
  public:
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            print(f"  {cmd.short_str()} override {{", file=fbod)
            print(f"    temporary_allocator clone_allocator_;", file=fbod)
            print(f"    printer->begin_object(\"{cmd.name}\");", file=fbod)
            for x in cmd.args:
                output_arg_print(cmd, x, "    ", fbod)
            
            args = ", ".join(x.name for x in cmd.args)
            if (cmd.ret.name != 'void'):
                print(f"    auto ret = T::{cmd.name}({args});", file=fbod)
            else:
                print(f"    T::{cmd.name}({args});", file=fbod)

            print(f"    printer->end_object();", file=fbod)
            if (cmd.ret.name != 'void'):
                print(f"    return ret;", file=fbod)
            print(f"  }}", file=fbod)
        print(
'''
  public:
  Printer printer_;
  Printer* printer = &printer_;
};
} // namespace gapid2
''', file=fbod)

def output_null_caller(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "null_caller.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include <vulkan.h>', file=fbod)
        print(
'''
namespace gapid2 {
template<typename HandleUpdater>
class NullCaller : public CommandProcessor {
  public:
    using Updater = HandleUpdater;
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            print(f"  {cmd.short_str()} override {{", file=fbod)            
            for a in cmd.args:
                output_arg_create(cmd, a, fbod)

            if (cmd.ret.name != 'void'):
                print(f"    return {cmd.ret.short_str()}();", file=fbod)
            print(f"  }}", file=fbod)
        print(
'''
  public:
  HandleUpdater updater_;
  HandleUpdater* updater = &updater_;
};
} // namespace gapid2
''', file=fbod)    

def get_alldeps(t):
    all_deps = set()
    if type(t) == struct:
        all_deps.add(t)
        for y in t.members:
            if y.name == 'pNext':
                for eb in y.extended_by:
                        all_deps = all_deps.union(get_alldeps(eb[1]))
            all_deps = all_deps.union(get_alldeps(y.type))
    elif type(t) == union:
        all_deps.add(t)
        for y in t.members:
            all_deps = all_deps.union(get_alldeps(y.type))
    elif type(t) == pointer_type:
        all_deps = all_deps.union(get_alldeps(t.pointee))
    elif type(t) == const_type:
        all_deps = all_deps.union(get_alldeps(t.child))
    return all_deps

def get_command_deps(c):
    deps = set()
    for a in c.args:
        if a.name == "pAllocator":
            continue
        deps = deps.union(deps, get_alldeps(a.type))
    return deps


exts = [
#DXVK AND NORMAL
    "VK_KHR_get_physical_device_properties2",
    "VK_KHR_multiview",
    "VK_KHR_maintenance2",
    "VK_AMD_memory_overallocation_behavior",
    "VK_KHR_swapchain",
    "VK_KHR_surface",
    "VK_KHR_win32_surface",
    "VK_KHR_image_format_list",
    "VK_AMD_shader_fragment_mask",
    "VK_EXT_4444_formats",
    "VK_EXT_conservative_rasterization",
    "VK_EXT_custom_border_color",
    "VK_EXT_depth_clip_enable",
    "VK_EXT_extended_dynamic_state",
    "VK_EXT_full_screen_exclusive",
    "VK_EXT_host_query_reset",
    "VK_EXT_memory_budget",
    "VK_EXT_memory_priority",
    "VK_EXT_robustness2",
    "VK_EXT_shader_demote_to_helper_invocation",
    "VK_EXT_shader_stencil_export",
    "VK_EXT_shader_viewport_index_layer",
    "VK_EXT_transform_feedback",
    "VK_EXT_vertex_attribute_divisor",
    "VK_KHR_buffer_device_address",
    "VK_KHR_create_renderpass2",
    "VK_KHR_depth_stencil_resolve",
    "VK_KHR_draw_indirect_count",
    "VK_KHR_driver_properties",
    "VK_KHR_image_format_list",
    "VK_KHR_sampler_mirror_clamp_to_edge",
    "VK_KHR_shader_float_controls",
    "VK_KHR_swapchain",
    "VK_KHR_get_surface_capabilities2",

#MORE!
    "VK_KHR_maintenance3",
    "VK_KHR_storage_buffer_storage_class",
    "VK_KHR_shader_draw_parameters",
    "VK_KHR_16bit_storage",
    "VK_KHR_shader_atomic_int64",
    "VK_KHR_shader_float16_int8",
    "VK_KHR_timeline_semaphore",
    "VK_EXT_depth_range_unrestricted",
    "VK_EXT_descriptor_indexing",
    "VK_EXT_fragment_shader_interlock",
    "VK_EXT_shader_image_atomic_int64",
    "VK_EXT_scalar_block_layout",
    "VK_AMD_shader_core_properties",
    "VK_EXT_sample_locations",
    "VK_NV_shader_sm_builtins"
]

def output_forwards(definition, dir, commands_to_handle = None):
    all_enc_args = []
    with open(os.path.join(dir, "forwards.h"), mode="w") as fhead:
        print("#pragma once", file=fhead)
        print('#include <vulkan.h>', file=fhead)
        print('#include "helpers.h"', file=fhead)
        print('namespace gapid2 {', file=fhead)
        print('struct encoder;', file=fhead)
        print('struct decoder;', file=fhead)

        output = []
        for cmd in definition.commands.values():
                if commands_to_handle:
                    if not cmd in commands_to_handle:
                        continue
                for arg in cmd.args:
                    if arg.name == 'pAllocator':
                        continue
                    if type(arg.type) == pointer_type and not arg.type.const:
                        continue
                    tp = arg.type.get_noncv_type()
                    all_enc_args.extend([y.signature() for y in tp.get_serialization_params()])
                    all_enc_args.extend([y.signature() for y in tp.get_serialization_params(DESERIALIZE)])
                    all_enc_args.extend([y.signature() for y in tp.get_serialization_params(CLONE)])
        for x in all_enc_args:
            if not x in output:
                print(f'{x};', file=fhead)
                output.append(x)
        print('}', file=fhead)
def verify_unique_commands(definition):
    shas = {}
    for cmd in definition.commands.values():
        sha = int.from_bytes(hashlib.sha256(cmd.name.encode('utf-8')).digest()[:8], 'little')
        if sha in shas:
            error(f'Two commands hash to the same value {cmd.name} and {shas[sha]}')
        shas[sha] = cmd.name

def output_device_functions(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "device_functions.h"), mode="w") as fhead:
        print("#pragma once", file=fhead)
        print('#include <vulkan.h>\n\n', file=fhead)
        print('namespace gapid2 {', file=fhead)
        print('    struct DeviceFunctions {', file=fhead)
        print('        DeviceFunctions(VkDevice device, PFN_vkGetDeviceProcAddr get_device_proc_addr) {', file=fhead)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue':
                continue
            print(f'            {cmd.name} = reinterpret_cast<PFN_{cmd.name}>(get_device_proc_addr(device, "{cmd.name}"));', file=fhead)
        print('        };', file=fhead)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and cmd.args[0].name != 'queue':
                continue
            print(f'        PFN_{cmd.name} {cmd.name};', file=fhead)
        print('    };', file=fhead)
        print('}', file=fhead)
                
def output_instance_functions(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "instance_functions.h"), mode="w") as fhead:
        print("#pragma once", file=fhead)
        print('#include <vulkan.h>\n\n', file=fhead)
        print('namespace gapid2 {', file=fhead)
        print('    struct InstanceFunctions {', file=fhead)
        print('        InstanceFunctions(VkInstance instance, PFN_vkGetInstanceProcAddr get_instance_proc_addr) {', file=fhead)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            if cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                continue
            print(f'            {cmd.name} = reinterpret_cast<PFN_{cmd.name}>(get_instance_proc_addr(instance, "{cmd.name}"));', file=fhead)
        print('        };', file=fhead)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            if cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                continue
            print(f'        PFN_{cmd.name} {cmd.name};', file=fhead)
        print('    };', file=fhead)
        print('}', file=fhead)
            
def output_empty_object(cmd, x, vop, arg_idx, idx, depth, fenc):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child
  
    if type(tp) != pointer_type:
        error("Expected null-terminated char list")

    

def output_arg_dec(cmd, x, vop, arg_idx, idx, depth, fenc):
    tp = x.type
    while(type(tp) == const_type):
        tp = tp.child
  
    if type(tp) == union:
        prms = [f"{vop}{x.name}{idx}", "decoder_"]
        prms.extend([x.name() for x in tp.get_serialization_params(DESERIALIZE)])

        print(depth + f"_custom_deserialize_{tp.name}(updater, {', '.join(prms)});", file=fenc)
    if type(tp) == struct:
        prms = [f"{vop}{x.name}{idx}", "decoder_"]
        prms.extend([x.name() for x in tp.get_serialization_params(DESERIALIZE)])
        print(depth + f"deserialize_{tp.name}(updater, {', '.join(prms)});", file=fenc)
    elif type(tp) == basetype:
        if tp.base_type == "size_t":
            print(depth + f"{vop}{x.name}{idx} = static_cast<size_t>(decoder_->decode<uint64_t>());", file=fenc)
        else:
            print(depth + f"{vop}{x.name}{idx} = decoder_->decode<{tp.base_type}>();", file=fenc)
    elif type(tp) == platform_type:
        if str(tp) == "size_t":
            print(depth + f"{vop}{x.name}{idx} = static_cast<size_t>(decoder_->decode<uint64_t>());", file=fenc)
        else:
            print(depth + f"{vop}{x.name}{idx} = decoder_->decode<{str(tp)}>();", file=fenc)
    elif type(tp) == pointer_type:
        if (tp.pointee.name == "void"):
            prms = [x.name for x in cmd.args]
            prms.append("decoder_")
            print(depth + f"_custom_deserialize_{cmd.name}_{x.name}(updater, {', '.join(prms)});", file=fenc)
            return
        # Special case for strings
        if x.len and x.len.startswith('latexmath'):
            print(depth + f"// Latexmath string", file=fenc)
            return
        if x.len and x.len == 'null-terminated':
            if tp.get_noncv_type().name != 'char':
                error("Expected null-terminated char list")
            #encode
            print(depth + f"uint64_t len_ = decoder_->decode<uint64_t>();", file=fenc)
            print(depth + f"if (len_) {{", file=fenc)
            print(depth + f"  {vop}{x.name}{idx} = static_cast<{x.type.get_noncv_type().get_member('')[0:-1]}*>(decoder_->get_memory(len_));", file=fenc)
            print(depth + f"  decoder_->decode_primitive_array<char>({vop}{x.name}{idx}, len_);", file=fenc)
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  {vop}{x.name}{idx} = nullptr;", file=fenc)
            print(depth + f"}}", file=fenc)
            return

        ct = "1"
        if x.len:
            ct = f"{vop}{x.len.split(',')[0]}"
            p = [x for x in cmd.args if x.name == ct]
            if len(p):
                if type(p[0].type) == pointer_type:
                    ct = f"*{ct}"

        if x.optional:
            print(depth + f"if (decoder_->decode<char>()) {{", file=fenc)
            depth += "  "

        if x.optional or x.len:
            print(depth + f"{vop}{x.name}{idx} = decoder_->get_typed_memory<{tp.pointee.get_non_const_member('')[0:-1]}>({ct});", file=fenc)

        mem_idx = f'{idx}[0]'
        if x.len:
            ii = len(depth)
            argct = f"{vop}{x.len.split(',')[0]}"
            p = [x for x in cmd.args if x.name == argct]
            if len(p):
                if type(p[0].type) == pointer_type:
                    argct = f"*{argct}"
            print(depth + f"for (size_t i_{ii} = 0; i_{ii} < {argct}; ++i_{ii}) {{", file=fenc)
            depth += "  "
            mem_idx = f'{idx}[i_{ii}]'

        xm = copy.deepcopy(x)
        xm.type = tp.pointee
        
        if x.len:
            xm.len = ",".join(x.len.split(",")[1:])
        output_arg_dec(cmd, xm, vop, arg_idx, mem_idx, depth, fenc)
            
        if x.len:
            depth = depth[:-2]
            print(depth + f"}}", file=fenc)
        if x.optional:
            depth = depth[:-2]
            print(depth + f"}} else {{", file=fenc)
            print(depth + f"  {vop}{x.name}{idx} = nullptr;", file=fenc)
            print(depth + f"}}", file=fenc)

    elif type(tp) == enum:
        print(depth + f"{vop}{x.name}{idx} = static_cast<{tp.name}>(decoder_->decode<{tp.base_type.name}>());", file=fenc)
    elif type(tp) == handle:
        if tp.dispatch == handle.DISPATCHABLE:
            print(depth + f"{vop}{x.name}{idx} = reinterpret_cast<{tp.name}>(static_cast<uintptr_t>(decoder_->decode<uint64_t>()));", file=fenc)
        else:
            print(depth + f"{vop}{x.name}{idx} = reinterpret_cast<{tp.name}>(decoder_->decode<uint64_t>());", file=fenc)
    elif type(tp) == array_type:
        print(depth + f"for (size_t i = 0; i < {tp.size}; ++i) {{", file=fenc)
        depth = depth + "  "
        mem_idx = f'{idx}[i]'
        xm = copy.deepcopy(x)
        xm.type = tp.child
        output_arg_dec(cmd, xm, vop, arg_idx, mem_idx, depth, fenc)
        depth = depth[:-2]
        print(depth + f"}}", file=fenc)

def output_handle_defines(definition, dir, types_to_handle = None):
    with open(os.path.join(dir, "handle_defines.inl"), mode="w") as fhead:
        print("#ifndef PROCESS_HANDLE", file=fhead)
        print('#error "Please define PROCESS_HANDLE"', file=fhead)
        print('#endif', file=fhead)
        print('', file=fhead)
        
        for x in definition.types.values():
            if types_to_handle != None:
                if not x in types_to_handle:
                    continue
            if type(x) == handle:
                print(f'PROCESS_HANDLE({x.name})',  file=fhead)

def output_arg_register(cmd, param, file):
    tp = param.type
    if not tp.has_handle():
        return
    if type(tp) != pointer_type:
        return
    if type(tp) == pointer_type:
        if type(param.type) == const_type:
            return
        pt = param.type.pointee.name
        zn = cmd.args[0].type.name
        len = "1"
        if param.len:
            len = param.len
        if type(tp.pointee) == struct:
            prms = [param.name, len]
            print(f"    this->updater_.register_handle_from_struct({', '.join(prms)});", file=file)
        else:
            prms = [param.name, len]
            print(f"    this->updater_.register_handle({', '.join(prms)});", file=file)

def output_command_deserializer(cmd, definition, fbod):
    print(f"  virtual void {cmd.name}() {{", file=fbod)
    arg_idx = 0
    print(f"    // -------- Args ------", file=fbod)
    for arg in cmd.args:
        t = arg.type
        while(type(t) == const_type):
            t = t.child
        if type(t) == array_type:
            print(f"    {t.child.get_noncv_type().name} {arg.name}[{t.size}];", file=fbod)
            continue
        if type(t) != pointer_type:
            print(f"    {t.name} {arg.name};", file=fbod)
            continue
        
        if t.pointee.name == "void":
            print(f"    void* {arg.name};", file=fbod)
            continue
        if type(t.pointee) == pointer_type and t.pointee.pointee.name == "void":
            print(f"    void** {arg.name};", file=fbod)
            continue
        ptr = arg.type
        name = arg.name
        if arg.inout():
            name = f'tmp_{arg.name}'
        if not arg.optional and not arg.len:
            print(f"    {t.pointee.get_noncv_type().name} {name}[1];", file=fbod)
        elif arg.optional:
            # Is a pointer type
            print(f"    {arg.type.get_noncv_type().name}* {name}; // optional", file=fbod)
        elif arg.len:
            print(f"    {arg.type.get_noncv_type().name}* {name}; // length {arg.len}", file=fbod)
    print(f"    // -------- Serialized Params ------", file=fbod)
    for arg in cmd.args:
        if arg.name == 'pAllocator':
            print(f"    pAllocator = nullptr; // pAllocator ignored on replay", file=fbod)
            continue
        t = ""
        if type(arg.type) == pointer_type and not arg.type.const:
            if not arg.inout():
                continue
            t = "tmp_"
        
        output_arg_dec(cmd, arg, t, arg_idx, "",  "    ", fbod)
        arg_idx += 1
    
    print(f"    // -------- Out Params ------", file=fbod)
    for arg in cmd.args:
        if not(type(arg.type) == pointer_type and not arg.type.const):
            continue

        if arg.inout():
            if not arg.optional and not arg.len:
                print(f"    {arg.type.pointee.name} {arg.name}[1]; // inout", file=fbod)
            elif arg.optional:
                # Is a pointer type
                print(f"    {arg.type.name} {arg.name}; // optional inout", file=fbod)
            elif arg.len:
                print(f"    {arg.type.name} {arg.name}; // length {arg.len} inout", file=fbod)
            output_arg_dec(cmd, arg, "", arg_idx, "",  "    ", fbod)
        else:
            output_arg_dec(cmd, arg, "", arg_idx, "",  "    ", fbod)
        arg_idx += 1

    print(f"    // -------- FixUp Params ------", file=fbod)
    for arg in cmd.args:
        output_arg_register(cmd, arg, fbod)
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
            print(f"    memcpy({arg.name}, tmp_{arg.name}, sizeof({arg.name}[0]) * {ct}); // setting inout properly", file=fbod)
    
    
    if cmd.ret.name == "VkResult":
        print(f"    current_return_ = decoder_->decode<VkResult>();", file=fbod)
    print(f"    // -------- Call ------", file=fbod)
    args = ", ".join(x.name for x in cmd.args)
    print(f"    T::{cmd.name}({args});", file=fbod)
    print(f'    GAPID2_ASSERT(this->updater_.tbd_handles.empty(), "Unprocessed handles");', file=fbod)
    print(f"  }}", file=fbod)

def output_call_deserializer(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "command_deserializer.h"), mode="w") as fbod:
        print("#pragma once", file=fbod)
        print('#include "struct_deserialization.h"', file=fbod)
        print('#include "decoder.h"', file=fbod)
        print('#include <iostream>', file=fbod)

        print(
'''
namespace gapid2 {
template<typename T>
class CommandDeserializer : public T {
  static_assert(std::is_base_of<CommandProcessor, T>::value, "Expected T to be derived from CommandProcessor");
  public:
''', file=fbod)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and  cmd.args[0].name != 'queue' and cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                print("  // ---- Special", file=fbod)
            output_command_deserializer(cmd, definition, fbod)
        
        
        print("  void DeserializeStream() {", file=fbod)
        print("    do {", file=fbod)
        print("      const uint64_t data_left = decoder_->data_left();", file=fbod)
        print("      if (data_left < sizeof(uint64_t)) { return; }", file=fbod)
        print("      if (data_left - sizeof(uint64_t) < decoder_->decode<uint64_t>()) { return; } ", file=fbod)
        print("      uint64_t command_idx = decoder_->decode<uint64_t>();", file=fbod)
        print("      switch(command_idx) {", file=fbod)

        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            if cmd.args[0].name != 'device' and cmd.args[0].name != 'commandBuffer' and  cmd.args[0].name != 'queue' and cmd.args[0].name != 'instance' and cmd.args[0].name != 'physicalDevice':
                print("  // ---- Special", file=fbod)
            sha = int.from_bytes(hashlib.sha256(cmd.name.encode('utf-8')).digest()[:8], 'little')
            print(f"        case {sha}u: {cmd.name}(); continue;", file=fbod)
        print('''
        default:
            std::abort();
        case 0:  { // mapped_memory_write
            VkDeviceMemory mem = reinterpret_cast<VkDeviceMemory>(decoder_->decode<uint64_t>()); 
            VkDeviceSize offset = decoder_->decode<VkDeviceSize>();
            VkDeviceSize size = decoder_->decode<VkDeviceSize>();
            void* data_loc = get_memory_write_location(mem, offset, size);
            decoder_->decode_primitive_array(reinterpret_cast<char*>(data_loc), size);
            continue;
        }
''', file=fbod)
        print("      }", file=fbod)
        print("    } while(true);", file=fbod)
        print("  }", file=fbod)
        print(
'''
  public:
    virtual void *get_memory_write_location(VkDeviceMemory, VkDeviceSize, VkDeviceSize) {
        return nullptr;
    } 
  decoder* decoder_;
  VkResult current_return_;
};
} // namespace gapid2
''', file=fbod)

def output_forward_header(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "layer_internal.inl"), mode="w") as fcall:
        print('\nvoid* user_data;', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            print(f'void *call_{cmd.name}_user_data;', file=fcall)
            print(f'{cmd.ret.short_str()} (*call_{cmd.name})(void*, {", ".join(prms)});', file=fcall)
            
        for x in definition.types.values():
            if type(x) == handle:
                print(f'{x.name} (*get_raw_handle_{x.name})(void* data_, {x.name} in);', file=fcall)
        print(
'''
extern "C" {
__declspec(dllexport) void SetupLayerInternal(void* user_data_, void* (fn)(void*, const char*, void**), void*(tf)(void*, const char*)) {
  user_data = user_data_;
''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            print(f'  call_{cmd.name} = ({cmd.ret.short_str()}(*)(void*, {", ".join(prms)}))fn(user_data_, "{cmd.name}", &call_{cmd.name}_user_data);', file=fcall)
        for x in definition.types.values():
            if type(x) == handle:
                print(f'  get_raw_handle_{x.name} = ({x.name}(*)(void*, {x.name}))tf(user_data_, "{x.name}");', file=fcall)
        print(
'''
  SetupInternalPointers(user_data_, fn);
}
}
''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            print(
f'''
inline {cmd.ret.short_str()} {cmd.name}({', '.join(prms)}) {{
  return (*call_{cmd.name})(call_{cmd.name}_user_data, {', '.join(args)});
}}''', file=fcall)
        for x in definition.types.values():
            if type(x) == handle:
                print(
f'''
{x.name} get_raw_handle({x.name} in) {{
  return (*get_raw_handle_{x.name})(user_data, in);
}}''', file=fcall)
        print(
'''
#undef VKAPI_ATTR
#undef VKAPI_CALL
#define VKAPI_CALL
#define VKAPI_ATTR extern "C" __declspec(dllexport)
''', file=fcall)

def output_layerer(definition, dir, commands_to_handle = None):
    with open(os.path.join(dir, "layerer.h"), mode="w") as fcall:
        print('#pragma once', file=fcall)
        print('#include <vulkan/vulkan.h>', file=fcall)
        print('#include <filesystem>', file=fcall)
        print('#include "algorithm/sha1.hpp"', file=fcall)
        print('#include "command_processor.h"', file=fcall)
        print('''namespace gapid2 {
  const std::string version_string = "1";
  
  std::string inline get_file_sha(const std::string& str) {
    std::ifstream t(str);
    if (t.bad()) {
      return "";
    }
    std::stringstream buffer;
    buffer << t.rdbuf();
    digestpp::sha1 hasher;
    hasher.absorb(version_string);
  #ifndef NDEBUG
    hasher.absorb("Debug");
  #else
    hasher.absorb("RelWithDebInfo");
  #endif
    hasher.absorb(buffer.str());
    return hasher.hexdigest();
  }
  
''', file=fcall)
        
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            print(
f'''
    {cmd.ret.short_str()} inline forward_{cmd.name}(void* fn, {", ".join(prms)}) {{
        return (*({cmd.ret.short_str()}(*)({", ".join(prms)}))(fn))({", ".join(args)});
    }}''', file=fcall)
        
        print('    struct fns {', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            print(f'      {cmd.ret.short_str()}(*fn_{cmd.name})(void*, {", ".join(prms)});', file=fcall)
            print(f'      void* {cmd.name}_user_data;', file=fcall)
        print('''
    };

    template<typename T>
    class Layerer: public T {
      using super = T;
      public:''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            print(
f'''
        static {cmd.ret.short_str()} next_layer_{cmd.name}(void* data_, {", ".join(prms)}) {{
            return reinterpret_cast<T*>(data_)->T::{cmd.name}({", ".join(args)});
        }}''', file=fcall)

        print(
f'''
        
        template<typename TT>
        static TT get_raw_handle(void* data_, TT in) {{
            return reinterpret_cast<T*>(data_)->updater_.cast_in(in);
        }}''', file=fcall)

        print(
'''        
        ~Layerer() {
            for (auto& mod: modules) {
                FreeLibrary(mod);
            }
        }
        fns f;
        void RunUserSetup(HMODULE module);
        void* ResolveHelperFunction(const char* name, void** fout);
''', file=fcall)
        print('''        bool initializeLayers(std::vector<std::string> layers) {
          char cp[MAX_PATH];
          GetModuleFileName(NULL, cp, MAX_PATH);
          std::vector<std::string> layer_dlls;
          char cwd[MAX_PATH];
          GetCurrentDirectoryA(MAX_PATH, cwd);
          digestpp::sha1 hasher;
          hasher.absorb(cp);
          hasher.absorb(cwd);
          hasher.absorb(version_string);
#ifndef NDEBUG
          hasher.absorb("Debug");
#else
          hasher.absorb("RelWithDebInfo");
#endif

          std::string work_path = hasher.hexdigest();
          for (auto& layer : layers) {
            if (!std::filesystem::exists(layer)) {
              GAPID2_ERROR("Could not find layer file");
            }
            auto file = std::filesystem::absolute(layer);
            std::string sha = get_file_sha(file.string());
            if (sha.empty()) {
              GAPID2_ERROR("Could not get sha for file");
            }
            char* t = getenv("TEMP");
            std::string dll(t);
            dll += std::string("\\\\") + work_path + "\\\\" + sha + ".dll";
            if (std::filesystem::exists(dll)) {
              std::cout << "Using existing layer " << dll << std::endl;
              layer_dlls.push_back(dll);
              continue;
            }
            std::string v = "cmd /c C:\\\\src\\\\gapid\\\\build_file.bat ";
            v += file.string();
            v += " ";
            v += sha;
#ifndef NDEBUG
            v += " Debug";
#else
            v += " RelWithDebInfo";
#endif
            v += std::string(" ") + t + std::string("\\\\") + work_path + "\\\\";

            int ret = system(v.c_str());
            if (ret != 0) {
              GAPID2_ERROR("Could not build layer");
            }
            layer_dlls.push_back(dll);
            std::cout << "Built and readied layer " << layer_dlls.back() << std::endl;
          }                
            ''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            print(f'            f.fn_{cmd.name} = &next_layer_{cmd.name};', file=fcall)
            print(f'            f.{cmd.name}_user_data = static_cast<T*>(this);', file=fcall)
        print('''
        for (const auto& layer: layer_dlls) {
          auto lib = LoadLibraryA(layer.c_str());
          if (!lib) {
              std::cerr << "Could not load library " << layer << std::endl;
              return false;
          }
          modules.push_back(lib);
          auto setup = (void (*)(void*, void* (*)(void*, const char*, void**), void*(tf)(void*, const char*)))GetProcAddress(lib, "SetupLayerInternal");
          if (!setup) {
              std::cerr << "Could not find library setup for " << layer << std::endl;
              return false;
          }
          std::cerr << "Setting up library " << layer << std::endl;
          setup(this, [](void* this__, const char* fn, void** fout) -> void* {
            Layerer<T>* this_ = reinterpret_cast<Layerer<T>*>(this__);''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            print(f'            if (!strcmp(fn, "{cmd.name}")) {{', file=fcall)
            print(f'              *fout = this_->f.{cmd.name}_user_data;', file=fcall)
            print(f'              return reinterpret_cast<void*>(this_->f.fn_{cmd.name});', file=fcall)
            print(f'            }}', file=fcall)
        print('''
            auto ret = this_->ResolveHelperFunction(fn, fout);
            if (!ret) {
                std::cerr << "Could not resolve function " << fn << std::endl;
            }
            return ret;
          }, [](void* this__, const char* tp) -> void* {''', file=fcall)
        for x in definition.types.values():
            if type(x) == handle:
                print(f'            if (!strcmp(tp, "{x.name}")) {{', file=fcall)
                print(f'              auto r = Layerer<T>::get_raw_handle<{x.name}>;', file=fcall)
                print(f'              return r;', file=fcall)
                print(f'            }}', file=fcall)
        print('''            std::cerr << "Could not resolve handle type " << tp << std::endl;''', file=fcall)
        print('''          });''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.type.name for x in cmd.args]
            print(f'            auto f_{cmd.name} = ({cmd.ret.short_str()}(*)({", ".join(prms)}))GetProcAddress(lib, "override_{cmd.name}");', file=fcall)
            print(f'            if (f_{cmd.name}) {{', file=fcall)
            print(f'              f.{cmd.name}_user_data = f_{cmd.name};', file=fcall)
            print(f'              f.fn_{cmd.name} = &forward_{cmd.name};', file=fcall)
            print(f'              std::cerr << "Found function override_{cmd.name} in layer, setting up chain" << std::endl;', file=fcall)
            print(f'            }}', file=fcall)
        print(
'''
          RunUserSetup(lib);
        }
        return true;
      }
''', file=fcall)
        for cmd in definition.commands.values():
            if commands_to_handle:
                if not cmd in commands_to_handle:
                    continue
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            print(
f'''
        {cmd.ret.short_str()} {cmd.name}({", ".join(prms)}) override {{
            return f.fn_{cmd.name}(f.{cmd.name}_user_data, {", ".join(args)});
        }}''', file=fcall)
        print('''
      std::vector<HMODULE> modules;
  };
}
#include "layerer.inl"
''', file=fcall)
def main():
    parser = argparse.ArgumentParser(description='Processes the vulkan XML into code')
    parser.add_argument('vulkan_xml', type=str,
                        help='an integer for the accumulator')
    parser.add_argument('output_location', type=str,
                        help='an integer for the accumulator')
    args = parser.parse_args() 
    vk = load_vulkan(args)
    definition = api_definition(vk, 1.2, exts)
    
    output_structs_enc_dec(definition, args.output_location)
    output_structs_clone(definition, args.output_location)

    verify_unique_commands(definition)
    output_commands(definition, args.output_location)
    output_returns_serializer(definition, args.output_location)
    output_command_processor(definition, args.output_location)
    output_command_caller(definition, args.output_location)
    output_forwards(definition, args.output_location)
    output_device_functions(definition, args.output_location)
    output_instance_functions(definition, args.output_location)
    output_call_forwards(definition, args.output_location)
    output_handle_defines(definition, args.output_location)
    output_call_deserializer(definition, args.output_location)
    output_forward_header(definition, args.output_location)
    output_layerer(definition, args.output_location)
    output_struct_printer(definition, args.output_location)
    output_command_printer(definition, args.output_location)
    output_null_caller(definition, args.output_location)

if __name__ == "__main__":
    main()
