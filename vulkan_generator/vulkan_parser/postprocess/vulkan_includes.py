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


def process(includes: Dict[str, internal_types.ExternalInclude]) -> Dict[str, types.ExternalInclude]:
    """Post process Includes"""
    new_includes: Dict[str, types.ExternalInclude] = {}

    for include in includes.values():
        new_includes[include.header] = types.ExternalInclude(
            header=include.header,
            directive=include.directive,
        )

    return new_includes
