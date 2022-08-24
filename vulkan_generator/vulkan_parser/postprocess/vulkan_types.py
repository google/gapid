# Copyright (C) 2022 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
This module responsible for processing the Vulkan types

All the stringly typed references will be linked during this stage.
"""


from typing import Dict
from typing import List
from typing import Optional
from typing import Union

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.internal import parser_utils
from vulkan_generator.vulkan_parser.api import types
from vulkan_generator.vulkan_parser.api import query


def process_enums(internal_enums: Dict[str, internal_types.VulkanEnum]) -> Dict[str, types.VulkanEnum]:
    """
    Post process enums
    All the field aliases are linked in here to their aliased fields
    """
    new_enums: Dict[str, types.VulkanEnum] = {}

    for enum in internal_enums.values():
        new_enum = types.VulkanEnum(
            typename=enum.typename,
            bitmask=enum.bitmask,
            bit64=enum.bit64,
        )

        for field in enum.fields.values():
            new_enum.fields[field.name] = types.VulkanEnumField(
                name=field.name,
                value=field.value,
                representation=field.representation,
                parent=new_enum,
            )

        for alias in enum.aliases.values():
            # Aliases can be chained, in that case each alias should
            # directly show the actual enum field
            aliased_typename = alias.aliased_typename
            while aliased_typename in enum.aliases.keys():
                aliased_typename = enum.aliases[aliased_typename].aliased_typename

            new_enum.aliases[alias.typename] = types.VulkanEnumFieldAlias(
                name=alias.typename,
                aliased_field=new_enum.fields[aliased_typename],
                extension=alias.extension,
                parent=new_enum
            )

        new_enums[enum.typename] = new_enum

    return new_enums


def process_union(
        internal_union: internal_types.VulkanUnion,
        new_types: types.VulkanTypeInfo) -> Union[types.VulkanUnion, str]:
    """
    Post process a Vulkan Union
    All the referenced types will be linked here
    """
    new_members: Dict[str, types.VulkanUnionMember] = {}

    for member in internal_union.members.values():
        if member.member_type not in new_types.all_types:
            return member.member_type

        typ = new_types.all_types[member.member_type]
        fullname = parser_utils.get_full_static_array_name(member.member_name, member.size)
        new_sizes = [int(s) for s in member.size] if member.size else None

        new_members[member.member_name] = types.VulkanUnionMember(
            type=typ,
            name=member.member_name,
            full_member_name=fullname,
            no_auto_validity=member.no_auto_validity,
            selection=member.selection,
            size=new_sizes
        )

    return types.VulkanUnion(
        typename=internal_union.typename,
        returned_only=internal_union.returned_only,
        members=new_members,
    )


def process_funcptr(
        internal_funcptr: internal_types.VulkanFunctionPtr,
        new_types: types.VulkanTypeInfo) -> Union[str, types.VulkanFunctionPtr]:
    """
    Post process an individual Function pointer
    All the referenced types will be linked here
    """
    new_arguments: Dict[str, types.VulkanFunctionArgument] = {}

    for argument in internal_funcptr.arguments.values():
        plain_typename = parser_utils.get_plain_typename(argument.argument_type)
        typ = new_types.all_types.get(plain_typename)
        if not typ:
            return plain_typename

        new_arguments[argument.argument_name] = types.VulkanFunctionArgument(
            type=typ,
            name=argument.argument_name,
            full_typename=argument.argument_type,
        )

    plain_return_typename = parser_utils.get_plain_typename(internal_funcptr.return_type)
    return_type = new_types.all_types.get(plain_return_typename)
    if not return_type:
        return plain_return_typename

    return types.VulkanFunctionPtr(
        typename=internal_funcptr.typename,
        return_type=return_type,
        arguments=new_arguments,
    )


def convert_size(size: List[str]) -> Union[str, List[int]]:
    """Convert simple arrays with non referencing sizes"""

    # Multidimensional arrays does not refer other types
    if len(size) > 1:
        return [int(s) for s in size]

    if size[0].isdigit():
        return [int(size[0])]

    # return the plain size reference for linking
    return parser_utils.clean_size_reference(size[0])


def process_struct(
        internal_struct: internal_types.VulkanStruct,
        new_types: types.VulkanTypeInfo,
        new_defines: Dict[str, types.VulkanDefine]) -> Union[str, types.VulkanStruct]:
    """
    process an individual Vulkan struct
    All the referenced types will be linked here
    """

    new_struct = types.VulkanStruct(
        typename=internal_struct.typename,
    )

    for member in internal_struct.members.values():
        expected_value: Optional[types.VulkanEnumField] = None
        if member.expected_value:
            expected_value = new_types.enums[member.member_type].fields[member.expected_value]

        size: Optional[Union[types.VulkanDefine, types.VulkanStructMember, List[int]]] = None
        static = True

        if member.size:
            new_size = convert_size(member.size)
            if isinstance(new_size, list):
                size = new_size
            else:
                size = new_defines.get(new_size)
                if not size:
                    new_struct.members.get(new_size)
                    static = False

        # Sometimes size has a calculation
        size_calculation: Optional[str] = None
        if member.size:
            if any(sign in member.size[0] for sign in ["*", "/", "+", "-"]):
                size_calculation = member.size[0]

        typ: Optional[types.VulkanType] = None
        plain_typename = parser_utils.get_plain_typename(member.member_type)

        # Sometimes structs refer the aliases instead of types themselves. In that
        # case we should deduct the actual type and use it.
        typ = query.try_get_type_or_deducted_type(plain_typename, new_types)
        if not typ:
            return plain_typename

        # Get the full variable name if it's a C bitfield or static array
        full_member_name = member.member_name
        if member.c_bitfield_size:
            full_member_name = parser_utils.get_full_c_bitfield_name(member.member_name, member.c_bitfield_size)
        elif static:
            full_member_name = parser_utils.get_full_static_array_name(member.member_name, member.size)

        new_struct.members[member.member_name] = types.VulkanStructMember(
            type=typ,
            name=member.member_name,
            parent=new_struct,
            full_typename=member.member_type,
            full_member_name=full_member_name,
            no_auto_validity=member.no_auto_validity,
            expected_value=expected_value,
            size=size,
            size_calculation=size_calculation,
            c_bitfield_size=member.c_bitfield_size,
            optional=member.optional,
        )

    if internal_struct.base_structs:
        for internal_base_struct in internal_struct.base_structs:
            base_struct = new_types.structs.get(internal_base_struct)
            if not base_struct:
                return internal_base_struct
            new_struct.base_structs[internal_base_struct] = base_struct

    # We are sure that creatng the current structure is succeed
    # So add this struct to it's base structs pNext chain
    for base_struct in new_struct.base_structs.values():
        base_struct.pnexts[new_struct.typename] = new_struct

    return new_struct


def process_type_with_dependencies(
        typename: str,
        unions: Dict[str, internal_types.VulkanUnion],
        funcptrs: Dict[str, internal_types.VulkanFunctionPtr],
        structs: Dict[str, internal_types.VulkanStruct],
        new_types: types.VulkanTypeInfo,
        new_defines: Dict[str, types.VulkanDefine]) -> None:
    """Recursively processes all the unions, function pointers and structs related with the given type"""

    # In Vulkan XML sometimes types are referred before they have been defined and internal structs keeps this order
    # Therefore any time a definition has not found that means that the referenced type has not been processed yet.
    # Therefore this function will try to process the given type, any time it fails to fetch a linked type,
    # it will process the linked type and the retry the original type until it succeeds.

    if typename in unions:
        internal_union = unions[typename]
        new_union = process_union(internal_union, new_types)
        if isinstance(new_union, str):
            process_type_with_dependencies(new_union, unions, funcptrs, structs, new_types, new_defines)
            process_type_with_dependencies(typename, unions, funcptrs, structs, new_types, new_defines)
        else:
            new_types.unions[new_union.typename] = new_union
            new_types.all_types[new_union.typename] = new_union

    if typename in funcptrs:
        internal_funcptr = funcptrs[typename]
        new_funcptr = process_funcptr(internal_funcptr, new_types)
        if isinstance(new_funcptr, str):
            process_type_with_dependencies(new_funcptr, unions, funcptrs, structs, new_types, new_defines)
            process_type_with_dependencies(typename, unions, funcptrs, structs, new_types, new_defines)
        else:
            new_types.funcpointers[new_funcptr.typename] = new_funcptr
            new_types.all_types[new_funcptr.typename] = new_funcptr

    if typename in structs:
        internal_struct = structs[typename]
        new_struct = process_struct(internal_struct, new_types, new_defines)
        if isinstance(new_struct, str):
            process_type_with_dependencies(new_struct, unions, funcptrs, structs, new_types, new_defines)
            process_type_with_dependencies(typename, unions, funcptrs, structs, new_types, new_defines)
        else:
            new_types.structs[new_struct.typename] = new_struct
            new_types.all_types[new_struct.typename] = new_struct


def process_vk_base_in_out_structure(
        internal_struct: internal_types.VulkanStruct,
        new_types: types.VulkanTypeInfo) -> None:
    """Process the VkBaseInStructure and VkBaseOutStructure which references itself"""

    new_struct = types.VulkanStruct(
        typename=internal_struct.typename,
    )

    stype = internal_struct.members["sType"]
    new_struct.members[stype.member_name] = types.VulkanStructMember(
        type=new_types.enums[stype.member_type],
        name=stype.member_name,
        parent=new_struct,
        full_typename=stype.member_type,
        full_member_name=stype.member_name,
        expected_value=None,
        no_auto_validity=None,
        size=None,
        size_calculation=None,
        c_bitfield_size=None,
        optional=None,
    )

    # Add itself as member
    pnext = internal_struct.members["pNext"]
    new_struct.members["pNext"] = types.VulkanStructMember(
        type=new_struct,
        name=pnext.member_name,
        parent=new_struct,
        full_typename=pnext.member_type,
        full_member_name=pnext.member_name,
        expected_value=None,
        no_auto_validity=None,
        size=None,
        size_calculation=None,
        c_bitfield_size=None,
        optional=None,
    )

    # Add itself as pNext
    new_struct.pnexts[new_struct.typename] = new_struct

    new_types.structs[new_struct.typename] = new_struct
    new_types.all_types[new_struct.typename] = new_struct


def process_structs_unions_and_funcptrs(
        unions: Dict[str, internal_types.VulkanUnion],
        funcptrs: Dict[str, internal_types.VulkanFunctionPtr],
        structs: Dict[str, internal_types.VulkanStruct],
        new_types: types.VulkanTypeInfo,
        new_defines: Dict[str, types.VulkanDefine]) -> None:
    """
    process Vulkan structs unions and funcpointers.

    All the referenced types will be linked here.

    This also takes care of irregularities of struct order in the XML
    """

    # Unions, structs and funcpointers are pretty entangled and irregular in terms of order and referencing

    for union in unions.values():
        if union.typename not in new_types.unions:
            # This calls a recursive function that will add all the dependencies
            # Do not add again the union if it is already added
            process_type_with_dependencies(union.typename, unions, funcptrs, structs, new_types, new_defines)

    for funcptr in funcptrs.values():
        if funcptr.typename not in new_types.funcpointers:
            # This calls a recursive function that will add all the dependencies
            # Do not add again the function pointer if it is already added
            process_type_with_dependencies(funcptr.typename, unions, funcptrs, structs, new_types, new_defines)

    for struct in structs.values():
        # Self reference
        if struct.typename in ["VkBaseOutStructure", "VkBaseInStructure"]:
            process_vk_base_in_out_structure(struct, new_types)
            continue

        if struct.typename not in new_types.structs:
            # This calls a recursive function that will add all the dependencies
            # Do not add again the struct if it is already added
            process_type_with_dependencies(struct.typename, unions, funcptrs, structs, new_types, new_defines)

    # Check if every type added in structs, unions and function pointers
    if len(new_types.unions) != len(unions):
        raise RuntimeError(f"""Unexpected number of unions:
        {len(new_types.unions)} : {len(unions)}""")

    if len(new_types.funcpointers) != len(funcptrs):
        raise RuntimeError(f"""Unexpected number of funcpointers:
        {len(new_types.funcpointers)} : {len(funcptrs)}""")

    if len(new_types.structs) != len(structs):
        raise RuntimeError(f"""Unexpected number of structs:
        {len(new_types.structs)} : {len(structs)}""")


def process(
        internal_vulkan_types: internal_types.AllVulkanTypes,
        new_defines: Dict[str, types.VulkanDefine]) -> types.VulkanTypeInfo:
    """This method converts the internal types to the public types with necessary linking and processing"""
    new_types = types.VulkanTypeInfo()

    # Skip the includes and defines that are not Vulkan types.
    # They will be handled separately.

    for external_type in internal_vulkan_types.external_types.values():
        new_types.external_types[external_type.typename] = types.ExternalType(
            typename=external_type.typename,
            source_header=external_type.source_header,
            ctype=external_type.ctype,
        )

    new_types.all_types.update(new_types.external_types)

    for basetype in internal_vulkan_types.basetypes.values():
        new_types.basetypes[basetype.typename] = types.VulkanBaseType(
            basetype.typename,
            basetype.basetype,
        )

    new_types.all_types.update(new_types.basetypes)

    for handle in internal_vulkan_types.handles.values():
        new_types.handles[handle.typename] = types.VulkanHandle(
            typename=handle.typename,
            dispatchable=handle.dispatchable,
        )

    new_types.all_types.update(new_types.handles)

    for handle_alias in internal_vulkan_types.handle_aliases.values():
        new_types.handle_aliases[handle_alias.typename] = types.VulkanHandleAlias(
            handle_alias.typename,
            aliased_type=new_types.handles[handle_alias.aliased_typename],
        )

    new_types.all_aliases.update(new_types.handle_aliases)

    # Handles enums separately as they are a bit complicated
    new_types.enums = process_enums(internal_vulkan_types.enums)

    new_types.all_types.update(new_types.enums)

    for enum_alias in internal_vulkan_types.enum_aliases.values():
        new_types.enum_aliases[enum_alias.typename] = types.VulkanEnumAlias(
            typename=enum_alias.typename,
            aliased_type=new_types.enums[enum_alias.aliased_typename]
        )

    new_types.all_aliases.update(new_types.enum_aliases)

    for bitmask in internal_vulkan_types.bitmasks.values():
        field_type: Optional[types.VulkanEnum] = None
        if bitmask.field_type:
            field_type = new_types.enums[bitmask.field_type]

        new_types.bitmasks[bitmask.typename] = types.VulkanBitmask(
            typename=bitmask.typename,
            field_type=field_type,
            field_basetype=new_types.basetypes[bitmask.field_basetype],
        )

    new_types.all_types.update(new_types.bitmasks)

    for bitmask_alias in internal_vulkan_types.bitmask_aliases.values():
        new_types.bitmask_aliases[bitmask_alias.typename] = types.VulkanBitmaskAlias(
            typename=bitmask_alias.typename,
            aliased_type=new_types.bitmasks[bitmask_alias.aliased_typename],
        )

    new_types.all_aliases.update(new_types.bitmask_aliases)

    # Post processing structs and unions are tied to each other, so let's do them together
    process_structs_unions_and_funcptrs(
        internal_vulkan_types.unions,
        internal_vulkan_types.funcpointers,
        internal_vulkan_types.structs,
        new_types,
        new_defines,
    )

    for struct_alias in internal_vulkan_types.struct_aliases.values():
        new_types.struct_aliases[struct_alias.typename] = types.VulkanStructAlias(
            typename=struct_alias.typename,
            aliased_type=new_types.structs[struct_alias.aliased_typename]
        )

    new_types.all_aliases.update(new_types.struct_aliases)
    return new_types
