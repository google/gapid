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
This module is responsible for testing Vulkan core versions

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import version_features_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_vulkan_core_version() -> None:
    """"Test the case for required feaures for a core Vulkan version"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <feature api="vulkan" name="VK_VERSION_1_0" number="1.0" comment="Vulkan core API interface definitions">
        <require comment="Header boilerplate">
            <type name="vk_platform"/>
            <type name="VK_DEFINE_HANDLE"/>
            <type name="VK_USE_64_BIT_PTR_DEFINES"/>
            <type name="VK_DEFINE_NON_DISPATCHABLE_HANDLE"/>
            <type name="VK_NULL_HANDLE"/>
        </require>
    </feature>"""

    vulkan_version = version_features_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_version, internal_types.VulkanCoreVersion)
    assert vulkan_version.name == "VK_VERSION_1_0"
    assert vulkan_version.number == "1.0"

    assert len(vulkan_version.features) == 5
    assert vulkan_version.features["vk_platform"].name == "vk_platform"
    assert vulkan_version.features["vk_platform"].feature_type == "type"
    assert not vulkan_version.features["vk_platform"].feature_extension


def test_vulkan_core_version_enum_extension_offset() -> None:
    """""Test the case if a core version adds an enum with offset"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <feature api="vulkan" name="VK_VERSION_1_1" number="1.1" comment="Vulkan 1.1 core API interface definitions.">
        <require comment="Originally based on VK_KHR_subgroup (extension 94), but the actual enum block used was, incorrectly, that of extension 95">
            <enum extends="VkStructureType" extnumber="95"  offset="0"          name="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SUBGROUP_PROPERTIES"/>
            <type                                       name="VkPhysicalDeviceSubgroupProperties"/>
            <type                                       name="VkSubgroupFeatureFlags"/>
            <type                                       name="VkSubgroupFeatureFlagBits"/>
        </require>
    </feature>"""

    vulkan_version = version_features_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_version, internal_types.VulkanCoreVersion)

    assert vulkan_version.features["VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SUBGROUP_PROPERTIES"].feature_type == "enum"
    assert vulkan_version.features["VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SUBGROUP_PROPERTIES"].feature_extension

    feature_extension = vulkan_version.features[
        "VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SUBGROUP_PROPERTIES"].feature_extension
    assert isinstance(feature_extension, internal_types.VulkanFeatureExtensionEnum)
    assert feature_extension.basetype == "VkStructureType"
    assert feature_extension.extnumber == "95"
    assert feature_extension.offset == "0"


def test_vulkan_core_version_enum_extension_bitpos() -> None:
    """""Test the case if a core version adds an enum with bitpos"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <feature api="vulkan" name="VK_VERSION_1_1" number="1.1" comment="Vulkan 1.1 core API interface definitions.">
        <require comment="Promoted from VK_EXT_pipeline_creation_cache_control (extension 298)">
            <enum offset="0" extends="VkStructureType"  extnumber="298"         name="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PIPELINE_CREATION_CACHE_CONTROL_FEATURES"/>
            <type name="VkPhysicalDevicePipelineCreationCacheControlFeatures"/>
            <enum bitpos="8" extends="VkPipelineCreateFlagBits"                 name="VK_PIPELINE_CREATE_FAIL_ON_PIPELINE_COMPILE_REQUIRED_BIT"/>
            <enum bitpos="9" extends="VkPipelineCreateFlagBits"                 name="VK_PIPELINE_CREATE_EARLY_RETURN_ON_FAILURE_BIT"/>
            <enum offset="0" extends="VkResult"         extnumber="298"         name="VK_PIPELINE_COMPILE_REQUIRED"/>
            <enum bitpos="0" extends="VkPipelineCacheCreateFlagBits"            name="VK_PIPELINE_CACHE_CREATE_EXTERNALLY_SYNCHRONIZED_BIT"/>
        </require>
    </feature>"""

    vulkan_version = version_features_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_version, internal_types.VulkanCoreVersion)

    assert vulkan_version.features["VK_PIPELINE_CREATE_EARLY_RETURN_ON_FAILURE_BIT"].feature_type == "enum"
    assert vulkan_version.features["VK_PIPELINE_CREATE_EARLY_RETURN_ON_FAILURE_BIT"].feature_extension

    feature_extension = vulkan_version.features["VK_PIPELINE_CREATE_EARLY_RETURN_ON_FAILURE_BIT"].feature_extension
    assert isinstance(feature_extension, internal_types.VulkanFeatureExtensionEnum)
    assert feature_extension.basetype == "VkPipelineCreateFlagBits"
    assert feature_extension.bitpos == "9"


def test_vulkan_core_version_enum_extension_value() -> None:
    """""Test the case if a core version adds an enum with bitpos"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <feature api="vulkan" name="VK_VERSION_1_1" number="1.1" comment="Vulkan 1.1 core API interface definitions.">
       <require comment="Promoted from VK_KHR_maintenance4 (extension 414)">
            <enum offset="0" extends="VkStructureType" extnumber="414"          name="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_MAINTENANCE_4_FEATURES"/>
            <enum offset="1" extends="VkStructureType" extnumber="414"          name="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_MAINTENANCE_4_PROPERTIES"/>
            <enum offset="2" extends="VkStructureType" extnumber="414"          name="VK_STRUCTURE_TYPE_DEVICE_BUFFER_MEMORY_REQUIREMENTS"/>
            <enum offset="3" extends="VkStructureType" extnumber="414"          name="VK_STRUCTURE_TYPE_DEVICE_IMAGE_MEMORY_REQUIREMENTS"/>
            <enum value="0"  extends="VkImageAspectFlagBits"                    name="VK_IMAGE_ASPECT_NONE"/>
            <type name="VkPhysicalDeviceMaintenance4Features"/>
            <type name="VkPhysicalDeviceMaintenance4Properties"/>
            <type name="VkDeviceBufferMemoryRequirements"/>
            <type name="VkDeviceImageMemoryRequirements"/>
            <command name="vkGetDeviceBufferMemoryRequirements"/>
            <command name="vkGetDeviceImageMemoryRequirements"/>
            <command name="vkGetDeviceImageSparseMemoryRequirements"/>
        </require>
    </feature>"""

    vulkan_version = version_features_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_version, internal_types.VulkanCoreVersion)

    assert vulkan_version.features["VK_IMAGE_ASPECT_NONE"].feature_type == "enum"
    assert vulkan_version.features["VK_IMAGE_ASPECT_NONE"].feature_extension

    feature_extension = vulkan_version.features["VK_IMAGE_ASPECT_NONE"].feature_extension
    assert isinstance(feature_extension, internal_types.VulkanFeatureExtensionEnum)
    assert feature_extension.basetype == "VkImageAspectFlagBits"
    assert feature_extension.value == "0"
