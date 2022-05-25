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
This module is responsible for testing Vulkan defines and constants

Because the XML is irregular, unfortunately the test structure is also irregular.

This module tests:
- Parsing constant defines defined under "API Constants" enum with enum_parser
- Parsing macro and constant defines from defines_parser

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import enums_parser
from vulkan_generator.vulkan_parser import defines_parser
from vulkan_generator.vulkan_parser import types


def test_vulkan_constant_with_value() -> None:
    """""Test the bitmask has "required" field"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <enums name="API Constants" comment="Vulkan hardcoded constants -
        not an enumerated type, part of the header boilerplate">
        <enum type="uint32_t" value="256"       name="VK_MAX_PHYSICAL_DEVICE_NAME_SIZE"/>
        <enum type="uint32_t" value="16"        name="VK_UUID_SIZE"/>
    </enums>
    """

    constants = enums_parser.parse(ET.fromstring(xml))

    assert isinstance(constants, dict)
    assert len(constants) == 2

    constants_as_list = list(constants)

    assert constants_as_list[0] == "VK_MAX_PHYSICAL_DEVICE_NAME_SIZE"
    assert constants["VK_MAX_PHYSICAL_DEVICE_NAME_SIZE"].value == "256"

    assert constants_as_list[1] == "VK_UUID_SIZE"
    assert constants["VK_UUID_SIZE"].value == "16"


def test_vulkan_define_with_name_tag() -> None:
    """""Test the bitmask has "required" field"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="define">// DEPRECATED: This define is deprecated. VK_API_VERSION_MAJOR should be used instead.
    #define <name>VK_VERSION_MAJOR</name>(version) ((uint32_t)(version) &gt;&gt; 22)</type>
    """

    define = defines_parser.parse(ET.fromstring(xml))

    assert isinstance(define, types.VulkanDefine)
    assert define.variable_name == "VK_VERSION_MAJOR(version)"
    assert define.value == "((uint32_t)(version) >> 22)"


def test_vulkan_define_with_name_tag_multiline() -> None:
    """""Test the bitmask has "required" field"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="define">// DEPRECATED: This define is deprecated. VK_MAKE_API_VERSION should be used instead.
#define <name>VK_MAKE_VERSION</name>(major, minor, patch) \\
    ((((uint32_t)(major)) &lt;&lt; 22) | (((uint32_t)(minor)) &lt;&lt; 12) | ((uint32_t)(patch)))</type>
    """

    define = defines_parser.parse(ET.fromstring(xml))

    assert isinstance(define, types.VulkanDefine)
    assert define.variable_name == "VK_MAKE_VERSION(major, minor, patch)"
    assert define.value == """\\
    ((((uint32_t)(major)) << 22) | (((uint32_t)(minor)) << 12) | ((uint32_t)(patch)))"""


# Melih: This test cannot be written in 120 lines
# pylint: disable=line-too-long

def test_vulkan_define_with_name_attrib() -> None:
    """""Test the bitmask has "required" field"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="define" name="VK_USE_64_BIT_PTR_DEFINES">
#ifndef VK_USE_64_BIT_PTR_DEFINES
    #if defined(__LP64__) || defined(_WIN64) || (defined(__x86_64__) &amp;&amp; !defined(__ILP32__) ) || defined(_M_X64) || defined(__ia64) || defined (_M_IA64) || defined(__aarch64__) || defined(__powerpc64__)
        #define VK_USE_64_BIT_PTR_DEFINES 1
    #else
        #define VK_USE_64_BIT_PTR_DEFINES 0
    #endif
#endif</type>
    """

    define = defines_parser.parse(ET.fromstring(xml))

    assert isinstance(define, types.VulkanDefine)
    assert define.variable_name == "VK_USE_64_BIT_PTR_DEFINES"
    assert define.value == """
#ifndef VK_USE_64_BIT_PTR_DEFINES
    #if defined(__LP64__) || defined(_WIN64) || (defined(__x86_64__) && !defined(__ILP32__) ) || defined(_M_X64) || """"""defined(__ia64) || defined (_M_IA64) || defined(__aarch64__) || defined(__powerpc64__)
        #define VK_USE_64_BIT_PTR_DEFINES 1
    #else
        #define VK_USE_64_BIT_PTR_DEFINES 0
    #endif
#endif"""

# pylint: enable=line-too-long
