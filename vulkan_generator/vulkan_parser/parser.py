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

"""This module is the entry point for the Vulkan parser that extracts information from Vulkan XML"""


from pathlib import Path

from vulkan_generator.vulkan_parser.internal import parser as internal_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def parse(filename: Path) -> internal_types.VulkanMetadata:
    return internal_parser.parse(filename)
