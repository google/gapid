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

""" This module is responsible for parsing Vulkan basetypes.

    Vulkan basetypes can be considered as C++ typedefs for
    either type declaration or forward declaration
    e.g.
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import parser_utils


def parse(basetype_elem: ET.Element) -> types.VulkanBaseType:
    """Returns a Vulkan Basetype from the XML element that defines it.

    A sample Vulkan Basetype:
    <type category="basetype">typedef <type>uint32_t</type> <name>VkSampleMask</name>;</type>

    Sometimes Vulkan basetypes are used as forward declarations:
    <type category="basetype">struct <name>ANativeWindow</name>;</type>
    """

    name = parser_utils.get_text_from_tag_in_children(basetype_elem, "name")
    basetype = parser_utils.try_get_text_from_tag_in_children(basetype_elem, "type")

    if basetype:
        # If basetype is a pointer, pointer attribute is in the tail of the type
        basetype_attribute = parser_utils.try_get_tail_from_tag_in_children(basetype_elem, "type")
        if basetype_attribute:
            basetype_attribute = parser_utils.clean_type_string(basetype_attribute)
            basetype = f"{basetype}{basetype_attribute}"

    return types.VulkanBaseType(typename=name, basetype=basetype)
