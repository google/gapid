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
This module is responsible for testing Vulkan extensions and aliases

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import extensions_parser


def test_vulkan_extension() -> None:
    """"Test the case for a simple Vulkan extension"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <extensions comment="Vulkan extension interface definitions">
        <extension name="VK_NV_glsl_shader" number="13" type="device" author="NV"
            contact="Piers Daniell @pdaniell-nv" supported="vulkan" deprecatedby="">
            <require>
                <enum value="1"                                                 name="VK_NV_GLSL_SHADER_SPEC_VERSION"/>
                <enum value="&quot;VK_NV_glsl_shader&quot;"                     name="VK_NV_GLSL_SHADER_EXTENSION_NAME"/>
                <enum offset="0" extends="VkResult" dir="-"                     name="VK_ERROR_INVALID_SHADER_NV"/>
            </require>
        </extension>
    </extensions>"""

    vulkan_extensions = extensions_parser.parse(ET.fromstring(xml))
    assert "VK_NV_glsl_shader" in vulkan_extensions

    extension = vulkan_extensions["VK_NV_glsl_shader"]
    assert isinstance(extension, types.VulkanExtension)
    assert extension.name == "VK_NV_glsl_shader"
    assert extension.number == 13
    assert extension.extension_type == "device"
    assert not extension.deprecating_extension
    assert not extension.required_extensions
    assert not extension.platform

    assert len(extension.requirements) == 1
    requirements = extension.requirements[0]

    assert len(requirements.features) == 3
    features = requirements.features

    ext_name = features["VK_NV_GLSL_SHADER_EXTENSION_NAME"]
    assert ext_name.name == "VK_NV_GLSL_SHADER_EXTENSION_NAME"
    assert ext_name.value == '"VK_NV_glsl_shader"'
    assert ext_name.feature_type == "type"

    spec_version = features["VK_NV_GLSL_SHADER_SPEC_VERSION"]
    assert spec_version.name == "VK_NV_GLSL_SHADER_SPEC_VERSION"
    assert spec_version.value == "1"
    assert spec_version.feature_type == "type"


def test_vulkan_extension_adding_enum_field_with_sign() -> None:
    """"Test the case for Vulkan extension adds an enum field with sign"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <extensions comment="Vulkan extension interface definitions">
        <extension name="VK_NV_glsl_shader" number="13" type="device" author="NV"
            contact="Piers Daniell @pdaniell-nv" supported="vulkan" deprecatedby="">
            <require>
                <enum value="1"                                                 name="VK_NV_GLSL_SHADER_SPEC_VERSION"/>
                <enum value="&quot;VK_NV_glsl_shader&quot;"                     name="VK_NV_GLSL_SHADER_EXTENSION_NAME"/>
                <enum offset="0" extends="VkResult" dir="-"                     name="VK_ERROR_INVALID_SHADER_NV"/>
            </require>
        </extension>
    </extensions>"""

    vulkan_extensions = extensions_parser.parse(ET.fromstring(xml))
    features = vulkan_extensions["VK_NV_glsl_shader"].requirements[0].features

    new_feature = features["VK_ERROR_INVALID_SHADER_NV"]
    assert new_feature.name == "VK_ERROR_INVALID_SHADER_NV"
    assert new_feature.feature_type == "enum"

    assert isinstance(new_feature.feature_extension, types.VulkanFeatureExtensionEnum)
    assert new_feature.feature_extension.offset == "0"
    assert new_feature.feature_extension.basetype == "VkResult"
    assert new_feature.feature_extension.sign == "-"


def test_vulkan_extension_adding_enum_field_alias() -> None:
    """"Test the case for Vulkan extension adds an enum field alias"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <extensions comment="Vulkan extension interface definitions">
        <extension name="VK_EXT_sampler_filter_minmax" number="131" type="device" author="NV" requires="VK_KHR_get_physical_device_properties2" contact="Jeff Bolz @jeffbolznv" supported="vulkan" promotedto="VK_VERSION_1_2">
            <require>
                <enum value="2"                                             name="VK_EXT_SAMPLER_FILTER_MINMAX_SPEC_VERSION"/>
                <enum value="&quot;VK_EXT_sampler_filter_minmax&quot;"      name="VK_EXT_SAMPLER_FILTER_MINMAX_EXTENSION_NAME"/>
                <enum extends="VkStructureType"                             name="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SAMPLER_FILTER_MINMAX_PROPERTIES_EXT" alias="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SAMPLER_FILTER_MINMAX_PROPERTIES"/>
                <enum extends="VkStructureType"                             name="VK_STRUCTURE_TYPE_SAMPLER_REDUCTION_MODE_CREATE_INFO_EXT" alias="VK_STRUCTURE_TYPE_SAMPLER_REDUCTION_MODE_CREATE_INFO"/>
                <enum extends="VkFormatFeatureFlagBits"                     name="VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_MINMAX_BIT_EXT" alias="VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_MINMAX_BIT"/>
                <enum extends="VkSamplerReductionMode"                      name="VK_SAMPLER_REDUCTION_MODE_WEIGHTED_AVERAGE_EXT" alias="VK_SAMPLER_REDUCTION_MODE_WEIGHTED_AVERAGE"/>
                <enum extends="VkSamplerReductionMode"                      name="VK_SAMPLER_REDUCTION_MODE_MIN_EXT" alias="VK_SAMPLER_REDUCTION_MODE_MIN"/>
                <enum extends="VkSamplerReductionMode"                      name="VK_SAMPLER_REDUCTION_MODE_MAX_EXT" alias="VK_SAMPLER_REDUCTION_MODE_MAX"/>
                <type name="VkSamplerReductionModeEXT"/>
                <type name="VkSamplerReductionModeCreateInfoEXT"/>
                <type name="VkPhysicalDeviceSamplerFilterMinmaxPropertiesEXT"/>
            </require>
        </extension>
    </extensions>"""

    vulkan_extensions = extensions_parser.parse(ET.fromstring(xml))
    features = vulkan_extensions["VK_EXT_sampler_filter_minmax"].requirements[0].features
    feature_extension = features["VK_SAMPLER_REDUCTION_MODE_WEIGHTED_AVERAGE_EXT"].feature_extension
    assert isinstance(feature_extension, types.VulkanFeatureExtensionEnum)
    assert feature_extension.basetype == "VkSamplerReductionMode"
    assert feature_extension.alias == "VK_SAMPLER_REDUCTION_MODE_WEIGHTED_AVERAGE"
