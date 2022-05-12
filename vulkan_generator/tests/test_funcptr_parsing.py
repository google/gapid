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
This package is responsible for testing Vulkan Parser

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import funcptr_parser
from vulkan_generator.vulkan_parser import types


def test_vulkan_func_pointer() -> None:
    """""Test the parsing of a function pointer"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="funcpointer">typedef void (VKAPI_PTR *
    <name>PFN_vkInternalAllocationNotification</name>)(
    <type>void</type>*                                       pUserData,
    <type>size_t</type>                                      size,
    <type>VkInternalAllocationType</type>                    allocationType,
    <type>VkSystemAllocationScope</type>                     allocationScope);</type>
    """
    funcptr = funcptr_parser.parse(ET.fromstring(xml))

    assert isinstance(funcptr, types.VulkanFunctionPtr)

    assert funcptr.typename == "PFN_vkInternalAllocationNotification"
    assert funcptr.return_type == "void"
    assert len(funcptr.arguments) == 4

    assert funcptr.argument_order[0] == "pUserData"
    assert funcptr.arguments["pUserData"].argument_type == "void*"

    assert funcptr.argument_order[1] == "size"
    assert funcptr.arguments["size"].argument_type == "size_t"

    assert funcptr.argument_order[2] == "allocationType"
    assert funcptr.arguments["allocationType"].argument_type == "VkInternalAllocationType"

    assert funcptr.argument_order[3] == "allocationScope"
    assert funcptr.arguments["allocationScope"].argument_type == "VkSystemAllocationScope"


def test_vulkan_func_pointer_with_pointer_return_value() -> None:
    """""Test the parsing of a function pointer with a pointer return type"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="funcpointer">typedef void* (VKAPI_PTR *
    <name>PFN_vkReallocationFunction</name>)(
    <type>void</type>*                                       pUserData,
    <type>void</type>*                                       pOriginal,
    <type>size_t</type>                                      size,
    <type>size_t</type>                                      alignment,
    <type>VkSystemAllocationScope</type>                     allocationScope);</type>
    """

    funcptr = funcptr_parser.parse(ET.fromstring(xml))

    assert isinstance(funcptr, types.VulkanFunctionPtr)
    assert funcptr.return_type == "void*"


def test_vulkan_func_pointer_with_const_member() -> None:
    """""Test the parsing of a function pointer with a const pointer argument"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="funcpointer">typedef VkBool32 (VKAPI_PTR *
    <name>PFN_vkDebugReportCallbackEXT</name>)(
    <type>VkDebugReportFlagsEXT</type>                       flags,
    <type>VkDebugReportObjectTypeEXT</type>                  objectType,
    <type>uint64_t</type>                                    object,
    <type>size_t</type>                                      location,
    <type>int32_t</type>                                     messageCode,
    const <type>char</type>*                                 pLayerPrefix,
    const <type>char</type>*                                 pMessage,
    <type>void</type>*                                       pUserData);</type>
    """

    funcptr = funcptr_parser.parse(ET.fromstring(xml))
    assert funcptr.argument_order[4] == "messageCode"
    assert funcptr.arguments["pLayerPrefix"].argument_type == "const char*"
