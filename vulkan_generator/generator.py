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

from pathlib import Path
import pprint

from vulkan_generator.vulkan_parser import parser as vulkan_parser
from vulkan_generator.vulkan_parser import types


def print_vulkan_metadata(vulkan_metadata: types.VulkanMetadata) -> None:
    """Prints all the vulkan information that is extracted"""

    pretty_printer = pprint.PrettyPrinter(depth=4)

    vulkan_types = vulkan_metadata.types

    print("=== Vulkan Defines ===")
    pretty_printer.pprint(vulkan_types.defines)

    print("=== Vulkan Bitmasks ===")
    pretty_printer.pprint(vulkan_types.bitmasks)

    print("=== Vulkan Bitmask Aliases ===")
    pretty_printer.pprint(vulkan_types.bitmask_aliases)

    print("=== Vulkan Enums ===")
    pretty_printer.pprint(vulkan_types.enums)

    print("=== Vulkan Enums Aliases ===")
    pretty_printer.pprint(vulkan_types.enum_aliases)

    print("=== Vulkan Handles ===")
    pretty_printer.pprint(vulkan_types.handles)

    print("=== Vulkan Handle Aliases ===")
    pretty_printer.pprint(vulkan_types.handle_aliases)

    print("=== Vulkan Structs ===")
    pretty_printer.pprint(vulkan_types.structs)

    print("=== Vulkan Struct Aliases ===")
    pretty_printer.pprint(vulkan_types.struct_aliases)

    print("=== Vulkan Function Pointers ===")
    pretty_printer.pprint(vulkan_types.funcpointers)

    vulkan_commands = vulkan_metadata.commands

    print("=== Vulkan Commands ===")
    pretty_printer.pprint(vulkan_commands.commands)

    print("=== Vulkan Command Aliases ===")
    pretty_printer.pprint(vulkan_commands.command_aliases)

    spirv_metadata = vulkan_metadata.spirv_metadata

    print("=== Spirv Extensions ===")
    pretty_printer.pprint(spirv_metadata.extensions)

    print("=== Spirv Capabilities ===")
    pretty_printer.pprint(spirv_metadata.capabilities)


def generate(vulkan_xml_path: Path) -> bool:
    """ Generator function """
    vulkan_info = vulkan_parser.parse(vulkan_xml_path)
    print_vulkan_metadata(vulkan_info)
    return True
