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
This module is responsible for testing Spirv information

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import spirv_capabilities_parser
from vulkan_generator.vulkan_parser.internal import spirv_extensions_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_spirv_extension() -> None:
    """""Test the case with a spirv extension enables a Vulkan extension"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvextension name="SPV_AMD_gcn_shader">
        <enable extension="VK_AMD_gcn_shader"/>
    </spirvextension>"""

    spirv_extension = spirv_extensions_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_extension, internal_types.SpirvExtension)
    assert spirv_extension.name == "SPV_AMD_gcn_shader"
    assert spirv_extension.extension == "VK_AMD_gcn_shader"


def test_spirv_extension_with_version() -> None:
    """""Test the case with a spirv extension with version"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvextension name="SPV_KHR_variable_pointers">
        <enable version="VK_VERSION_1_1"/>
        <enable extension="VK_KHR_variable_pointers"/>
    </spirvextension>"""

    spirv_extension = spirv_extensions_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_extension, internal_types.SpirvExtension)
    assert spirv_extension.version == "VK_VERSION_1_1"


def test_spirv_capability_with_version() -> None:
    """""Test the case with a spirv capability with version"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvcapability name="Shader">
        <enable version="VK_VERSION_1_0"/>
    </spirvcapability>"""

    spirv_capability = spirv_capabilities_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_capability, internal_types.SpirvCapability)
    assert spirv_capability.name == "Shader"
    assert spirv_capability.version == "VK_VERSION_1_0"


def test_spirv_capability_with_extension() -> None:
    """""Test the case with a spirv capability enables a Vulkan extension"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvcapability name="ShaderClockKHR">
        <enable extension="VK_KHR_shader_clock"/>
    </spirvcapability>"""

    spirv_capability = spirv_capabilities_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_capability, internal_types.SpirvCapability)
    assert spirv_capability.name == "ShaderClockKHR"
    assert spirv_capability.extension == "VK_KHR_shader_clock"


def test_spirv_capability_with_feature() -> None:
    """""Test the case with a spirv capability enables a feature"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvcapability name="Int64ImageEXT">
        <enable struct="VkPhysicalDeviceShaderImageAtomicInt64FeaturesEXT"
            feature="shaderImageInt64Atomics" requires="VK_EXT_shader_image_atomic_int64"/>
    </spirvcapability>"""

    spirv_capability = spirv_capabilities_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_capability, internal_types.SpirvCapability)
    assert spirv_capability.name == "Int64ImageEXT"
    assert spirv_capability.feature == "VkPhysicalDeviceShaderImageAtomicInt64FeaturesEXT::shaderImageInt64Atomics"


def test_spirv_capability_with_property() -> None:
    """""Test the case with a spirv capability enables a property"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvcapability name="GroupNonUniform">
        <enable property="VkPhysicalDeviceVulkan11Properties"
            member="subgroupSupportedOperations" value="VK_SUBGROUP_FEATURE_BASIC_BIT" requires="VK_VERSION_1_1"/>
    </spirvcapability>"""

    spirv_capability = spirv_capabilities_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_capability, internal_types.SpirvCapability)
    assert spirv_capability.name == "GroupNonUniform"

    expected = "VkPhysicalDeviceVulkan11Properties::subgroupSupportedOperations::VK_SUBGROUP_FEATURE_BASIC_BIT"
    assert spirv_capability.property == expected
