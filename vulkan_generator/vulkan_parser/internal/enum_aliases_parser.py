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

""" This module is responsible for parsing Vulkan Enum aliases

    Unfortunately the actual enums defined under a different subtree of `enums` tag
    in Vulkan XML. Therefore they will be parsed by a different module.
"""

from typing import Optional
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


def parse(enum_elem: ET.Element) -> Optional[internal_types.VulkanEnumAlias]:
    """Returns a Vulkan enum alias from the XML element that defines it.

    A sample Vulkan Enum Alias:
    <type category="enum" name="VkSemaphoreTypeKHR" alias="VkSemaphoreType"/>
    """

    alias_name = enum_elem.get("alias")

    if not alias_name:
        # The actual enums and their fields of enums are defined under enums tag
        # in the XML so they will be parsed separately.
        return None

    enum_name = enum_elem.attrib["name"]
    return internal_types.VulkanEnumAlias(typename=enum_name, aliased_typename=alias_name)
