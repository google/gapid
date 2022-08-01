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

"""This is the top level point for Vulkan Code Generator"""

import os

from pathlib import Path

from vulkan_generator.vulkan_parser import parser as vk_parser
from vulkan_generator.vulkan_parser.api import types
from vulkan_generator.handle_remapper import generator as handle_remapper_generator


def basic_generate(target: str,
                   output_dir: Path,
                   all_vulkan_types: types.VulkanInfo,
                   generate_header,
                   generate_cpp,
                   generate_test):

    generate_header(os.path.join(output_dir, target + ".h"), all_vulkan_types)
    generate_cpp(os.path.join(output_dir, target + ".cc"), all_vulkan_types)
    generate_test(os.path.join(output_dir, target + "_tests.cc"), all_vulkan_types)


def generate(vulkan_xml_path: Path, target: str, output_dir: Path) -> bool:
    """ Generator function """
    vulkan_metadata = vk_parser.parse(vulkan_xml_path)

    if output_dir == "":
        print("No output directory specified. Not generating anything.")
    elif target == "":
        print("No generate target specified. Not generating anything.")
    else:
        os.makedirs(output_dir, exist_ok=True)

        # Switch table for generate target. Add new targets here and throw exception for unknown targets
        if target == "handle_remapper":
            basic_generate(target, output_dir, vulkan_metadata,
                           handle_remapper_generator.generate_handle_remapper_h,
                           handle_remapper_generator.generate_handle_remapper_cpp,
                           handle_remapper_generator.generate_handle_remapper_tests)
        else:
            raise Exception("unknown generate target: " + target)

    return True
