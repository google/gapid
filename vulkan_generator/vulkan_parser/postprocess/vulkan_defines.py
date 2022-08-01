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
This module responsible for postprocessing the Vulkan defines

All the stringly typed references will be linked during this stage.
"""

from typing import Dict

from vulkan_generator.vulkan_parser.api import types
from vulkan_generator.vulkan_parser.internal import internal_types


def process(internal_defines: Dict[str, internal_types.VulkanDefine]) -> Dict[str, types.VulkanDefine]:
    """Post process defines"""
    new_defines: Dict[str, types.VulkanDefine] = {}

    for define in internal_defines.values():
        new_defines[define.key] = types.VulkanDefine(
            key=define.key,
            name=define.variable_name,
            value=define.value,
            extension=define.extension,
        )

    return new_defines
