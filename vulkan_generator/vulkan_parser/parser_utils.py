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

"""This package contains common functionalities used parsing different Vulkan types"""

from typing import Dict
from typing import Optional
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_utils import parsing_utils


def parse_enum_extension(enum_element: ET.Element) -> Optional[types.VulkanFeatureExtensionEnum]:
    """Parses the enum field extension added by the core version"""
    basetype = parsing_utils.try_get_attribute(enum_element, "extends")
    if not basetype:
        return None

    alias = parsing_utils.try_get_attribute(enum_element, "alias")
    extnumber = parsing_utils.try_get_attribute(enum_element, "extnumber")
    offset = parsing_utils.try_get_attribute(enum_element, "offset")
    bitpos = parsing_utils.try_get_attribute(enum_element, "bitpos")
    value = parsing_utils.try_get_attribute(enum_element, "value")
    sign = parsing_utils.try_get_attribute(enum_element, "dir")

    return types.VulkanFeatureExtensionEnum(
        basetype=basetype,
        alias=alias,
        extnumber=extnumber,
        offset=offset,
        sign=sign,
        bitpos=bitpos,
        value=value)


def parse_requirement(require_element: ET.Element) -> types.VulkanExtensionRequirement:
    """Parses requirement for Vulkan Core versions and extensions"""
    features: Dict[str, types.VulkanFeature] = {}

    version = parsing_utils.try_get_attribute(require_element, "feature")
    extension = parsing_utils.try_get_attribute(require_element, "extension")

    for required_feature_element in require_element:
        if required_feature_element.tag == "comment":
            continue

        feature_name = required_feature_element.attrib["name"]
        feature_type = required_feature_element.tag
        feature_value: Optional[str] = None

        feature_extension: Optional[types.VulkanFeatureExtension] = None
        if required_feature_element.tag == "enum":
            # Enums are expanded in core versions
            feature_extension = parse_enum_extension(required_feature_element)
            if not feature_extension:
                # If there is no enum override, then enum is actually a Vulkan API constant(C++ define)
                feature_type = "type"
                feature_value = parsing_utils.try_get_attribute(required_feature_element, "value")

        features[feature_name] = types.VulkanFeature(
            name=feature_name,
            feature_type=feature_type,
            value=feature_value,
            feature_extension=feature_extension)

    return types.VulkanExtensionRequirement(
        required_version=version,
        required_extension=extension,
        features=features,
    )
