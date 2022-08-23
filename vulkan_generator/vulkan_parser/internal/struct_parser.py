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

""" This module is responsible for parsing Vulkan structs and aliases of them"""

from typing import Dict
from typing import Optional
from typing import List

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import parser_utils
from vulkan_generator.vulkan_parser.internal import internal_types


def parse_struct_members(struct_element: ET.Element) -> Dict[str, internal_types.VulkanStructMember]:
    """Parses a Vulkan Struct member

    This is a bit of an irregular code because the XML itself has quite irregularities that
    makes is hard to parse type and name easily.

    For example a const pointer member is defined as:
     <member optional="true">const <type>void</type>*     <name>pNext</name></member>

    Where as a static array defined as:
    <member limittype="noauto">
        <type>char</type>
        <name>deviceName</name>[<enum>VK_MAX_PHYSICAL_DEVICE_NAME_SIZE</enum>]
    </member>
    """

    members: Dict[str, internal_types.VulkanStructMember] = {}
    struct_name = struct_element.attrib["name"]

    # This is a bit of an irregular code because the XML itself has quite irregularities that
    # makes is hard to parse type and name easily.
    #
    # This is not the code we wanted but it's the code that we needed and it's contained in a
    # small place so that XML irregularities does not leak into the rest of the code.
    for member_element in struct_element:
        if member_element.tag == "comment":
            # Melih TODO: We may want to support comments in the future
            continue

        if member_element.tag != "member":
            raise SyntaxError(
                f"No member tag found in : {ET.tostring(member_element, 'utf-8')}")

        member_type = parser_utils.get_text_from_tag_in_children(member_element, "type")
        member_name = parser_utils.get_text_from_tag_in_children(member_element, "name")

        # Type attributes(const, struct) and pointer attributes(*, const*, *const,*const*)
        # are usually in the text field of the member tag.
        #
        # Below is the example for a const pointer:
        #
        # <member optional="true">const <type>void</type>*     <name>pNext</name></member>
        #
        # In the below type is "const char* const*" but only "char" is in the type tag.
        # The rest of the information is scattered around the member's text
        # Therefore we need to retrieve and clean it so we can add it to the type.
        #
        # <member len="enabledLayerCount,null-terminated">const <type>char</type>
        # * const*      <name>ppEnabledLayerNames</name>
        #

        type_attributes = member_element.text
        # some times it's just empty space or endline character
        if type_attributes:
            type_attributes = parser_utils.clean_type_string(type_attributes)

            # It might be empty string after cleaning
            if type_attributes:
                member_type = f"{type_attributes} {member_type}"

        pointers = parser_utils.try_get_tail_from_tag_in_children(member_element, "type")
        if pointers:
            pointers = parser_utils.clean_type_string(pointers)

            # It might be empty string after cleaning
            if pointers:
                # Add space between "*" and "const"
                pointers = pointers.replace("const", " const")
                member_type = f"{member_type}{pointers}"

        if not member_type:
            raise SyntaxError(
                f"No member type found in : {ET.tostring(member_element, 'utf-8')}")

        if not member_name:
            raise SyntaxError(
                f"No member name found in : {ET.tostring(member_element, 'utf-8')}")

        # Currently if this attribute exists, it's always true
        no_auto_validity = member_element.get("noautovalidity") == "true"

        # This is useful for the sType where the correct value is already known
        expected_value = member_element.get("values")

        size = parser_utils.parse_member_sizes(member_element)

        c_bitfield_size: Optional[int] = None
        bitfield_str = parser_utils.try_get_tail_from_tag_in_children(member_element, "name")
        if bitfield_str and bitfield_str.startswith(":"):
            c_bitfield_size = int(bitfield_str[1:])

        # Is this field optional or has to be set
        # When this field is "false, true"  it's always for the length of the array
        # Therefore it does not give any extra information.
        #
        # Except for one case:
        # VkDescriptorBindingFlags in VkDescriptorSetLayoutBindingFlagsCreateInfo
        #
        # Instead of the count member, the actual array member is "false, true"
        # I think it's actually a bug in XML.
        # Melih TODO: Check if VkDescriptorBindingFlags is buggy in the XML
        optional = member_element.get("optional") == "true"

        members[member_name] = internal_types.VulkanStructMember(
            member_type=member_type,
            member_name=member_name,
            parent=struct_name,
            no_auto_validity=no_auto_validity,
            expected_value=expected_value,
            size=size,
            c_bitfield_size=c_bitfield_size,
            optional=optional,
        )

    return members


def parse(struct_elem: ET.Element) -> internal_types.VulkanType:
    """Returns a Vulkan struct or alias from the XML element that defines it.

    A sample Vulkan struct:
    <type category="struct" name="VkDevicePrivateDataCreateInfo"
        allowduplicate="true" structextends="VkDeviceCreateInfo">

        <member values="VK_STRUCTURE_TYPE_DEVICE_PRIVATE_DATA_CREATE_INFO">
            <type>VkStructureType</type> <name>sType</name>
        </member>
        <member optional="true">const <type>void</type>*<name>pNext</name></member>
        <member><type>uint32_t</type> <name>privateDataSlotRequestCount</name></member>
    </type>

    A sample Vulkan struct alias:
    <type category="struct" name="VkDevicePrivateDataCreateInfoEXT"
        alias="VkDevicePrivateDataCreateInfo"/>
    """

    struct_name = struct_elem.attrib["name"]

    alias_name = struct_elem.get("alias")
    if alias_name:
        return internal_types.VulkanStructAlias(typename=struct_name, aliased_typename=alias_name)

    base_structs = parser_utils.try_get_attribute_as_list(struct_elem, "structextends")

    members = parse_struct_members(struct_elem)
    return internal_types.VulkanStruct(
        typename=struct_name,
        base_structs=base_structs,
        members=members,
    )
