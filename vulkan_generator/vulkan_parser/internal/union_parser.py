# Copyright (C) 2022 Google Inc.
#
# Licensed under the Apache License, Verlon 2.0 (the "License");
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

"""This module is responsible for parsing Vulkan Types"""


from typing import Dict
import xml.etree.ElementTree as ET


from vulkan_generator.vulkan_parser.internal import parser_utils
from vulkan_generator.vulkan_parser.internal import internal_types


def parse_member(member_element: ET.Element) -> internal_types.VulkanUnionMember:
    """Parses a Vulkan Union Member"""
    typ = parser_utils.get_text_from_tag_in_children(member_element, "type")
    name = parser_utils.get_text_from_tag_in_children(member_element, "name")

    # Currently if this attribute exists, it's always true
    no_auto_validity = member_element.get("noautovalidity") == "true"
    selection = member_element.get("selection")
    size = parser_utils.parse_member_sizes(member_element)

    return internal_types.VulkanUnionMember(
        member_type=typ,
        member_name=name,
        no_auto_validity=no_auto_validity,
        selection=selection,
        size=size,
    )


def parse(union_element: ET.Element) -> internal_types.VulkanUnion:
    """Parses Vulkan Union from the XML that defines it"""
    name = union_element.attrib["name"]

    # Currently if this attribute exists, it's always true
    returned_only = union_element.get("returnedonly") == "true"

    members: Dict[str, internal_types.VulkanUnionMember] = {}

    for member_element in union_element:
        member = parse_member(member_element)
        members[member.member_name] = member

    return internal_types.VulkanUnion(
        typename=name,
        returned_only=returned_only,
        members=members
    )
