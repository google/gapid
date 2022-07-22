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

""" This module is responsible for parsing Spirv capabilities"""

from typing import Optional
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


def parse(spirv_element: ET.Element) -> internal_types.SpirvCapability:
    """Parses a Spirv capability or alias from the XML element that defines it

    A sample spirv capability:
    s<spirvcapability name="StoragePushConstant16">
            <enable struct="VkPhysicalDeviceVulkan11Features" feature="storagePushConstant16"
                requires="VK_VERSION_1_2"/>
            <enable struct="VkPhysicalDevice16BitStorageFeatures" feature="storagePushConstant16"
                requires="VK_KHR_16bit_storage"/>
    </spirvcapability>
    """

    name = spirv_element.attrib["name"]
    version: Optional[str] = None
    feature: Optional[str] = None
    vulkan_property: Optional[str] = None
    extension: Optional[str] = None

    for enable in spirv_element:
        if "version" in enable.attrib:
            version = enable.attrib["version"]
        elif "struct" in enable.attrib:
            # e.g. VkPhysicalDeviceVulkan11Features::storagePushConstant16
            feature = f"{enable.attrib['struct']}::{enable.attrib['feature']}"
        elif "property" in enable.attrib:
            # e.g. VkPhysicalDeviceVulkan11Properties::subgroupSupportedOperations::VK_SUBGROUP_FEATURE_BASIC_BIT
            vulkan_property = f"{enable.attrib['property']}::{enable.attrib['member']}::{enable.attrib['value']}"
        elif "extension" in enable.attrib:
            extension = enable.attrib["extension"]
        else:
            raise SyntaxError(f"Unknown Spirv capability type: {ET.tostring(spirv_element, 'utf-8')}")

    return internal_types.SpirvCapability(
        name=name,
        version=version,
        feature=feature,
        property=vulkan_property,
        extension=extension)
