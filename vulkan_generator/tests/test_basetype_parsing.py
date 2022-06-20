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
This module is responsible for testing Vulkan basetypes

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import basetype_parser
from vulkan_generator.vulkan_parser import types


def test_vulkan_basetype_type_declaration() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="basetype">typedef <type>uint32_t</type> <name>VkSampleMask</name>;</type>"""

    basetype = basetype_parser.parse(ET.fromstring(xml))

    assert isinstance(basetype, types.VulkanBaseType)
    assert basetype.typename == "VkSampleMask"
    assert basetype.basetype == "uint32_t"


def test_vulkan_basetype_forward_declaration() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="basetype">struct <name>ANativeWindow</name>;</type>"""

    basetype = basetype_parser.parse(ET.fromstring(xml))

    assert isinstance(basetype, types.VulkanBaseType)
    assert basetype.typename == "ANativeWindow"
    assert not basetype.basetype


def test_vulkan_basetype_with_pointer() -> None:
    """""Test the case if the basetype has a pointer"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="basetype">typedef <type>void</type>* <name>VkRemoteAddressNV</name>;</type>"""

    basetype = basetype_parser.parse(ET.fromstring(xml))

    assert isinstance(basetype, types.VulkanBaseType)
    assert basetype.basetype == "void*"
