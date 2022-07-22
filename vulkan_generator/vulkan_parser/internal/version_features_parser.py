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

""" This module is responsible for parsing features for each Vulkan version"""

from typing import Dict
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.internal import parser_utils


def parse(feature_element: ET.Element) -> internal_types.VulkanCoreVersion:
    """Parses features required by a specific Vulkan version"""
    if feature_element.attrib["api"] != "vulkan":
        raise SyntaxError(f"Unknown API {ET.tostring(feature_element, 'utf-8')!r}")

    version_name = feature_element.attrib["name"]
    version_number = feature_element.attrib["number"]

    features: Dict[str, internal_types.VulkanFeature] = {}

    for require_element in feature_element:
        if require_element.tag != "require":
            raise SyntaxError(f"Unknown Tag in Vulkan features {ET.tostring(require_element, 'utf-8')!r}")

        requirement = parser_utils.parse_requirement(require_element=require_element)
        features.update(requirement.features)

    return internal_types.VulkanCoreVersion(
        name=version_name,
        number=version_number,
        features=features
    )
