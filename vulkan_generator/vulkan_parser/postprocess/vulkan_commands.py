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
This module responsible for postprocessing the Vulkan commands

All the stringly typed references will be linked during this stage.
"""


from typing import Dict
from typing import List
from typing import Optional
from typing import Union

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.internal import parser_utils
from vulkan_generator.vulkan_parser.api import types
from vulkan_generator.vulkan_parser.api import query


def convert_render_pass_allowance(allowance: Optional[str]) -> types.RenderpassAllowance:
    """Convert render pass allowance information to a typed enum"""
    if not allowance:
        return types.RenderpassAllowance.UNKNOWN

    mapping: Dict[str, types.RenderpassAllowance] = {
        "both": types.RenderpassAllowance.BOTH,
        "inside": types.RenderpassAllowance.INSIDE,
        "outside": types.RenderpassAllowance.OUTSIDE,
    }

    if allowance not in mapping:
        raise ValueError(f"Unexpected renderpass Allowance: {allowance}")

    return mapping[allowance]


def convert_cmd_buffer_level(
        levels: Optional[List[str]],
        new_types: types.VulkanTypeInfo) -> Optional[List[types.VulkanEnumField]]:
    """Convert command buffer level information to Vulkan enum field"""
    if not levels:
        return None

    mapping: Dict[str, str] = {
        "primary": "VK_COMMAND_BUFFER_LEVEL_PRIMARY",
        "secondary": "VK_COMMAND_BUFFER_LEVEL_SECONDARY",
    }

    level_enum = new_types.enums["VkCommandBufferLevel"]
    return [level_enum.fields[mapping[level]] for level in levels]


def convert_vk_result_codes(
        codes: Optional[List[str]],
        new_types: types.VulkanTypeInfo) -> Optional[List[types.VulkanEnumField]]:
    """Convert Vulkan result code strings(success or error) to their corresponding enum field"""
    if not codes:
        return None

    results = new_types.enums["VkResult"]
    return [query.get_enum_field_or_deducted_field(results, code) for code in codes]


def convert_queues(
        queues: Optional[List[str]],
        new_types: types.VulkanTypeInfo) -> Optional[List[types.VulkanEnumField]]:
    """Convert Vulkan queue tag to VkQueueFlagBits"""
    if not queues:
        return None

    mapping: Dict[str, str] = {
        "graphics": "VK_QUEUE_GRAPHICS_BIT",
        "compute": "VK_QUEUE_COMPUTE_BIT",
        "transfer": "VK_QUEUE_TRANSFER_BIT",
        "sparse_binding": "VK_QUEUE_SPARSE_BINDING_BIT",
        "decode": "VK_QUEUE_VIDEO_DECODE_BIT_KHR",
        "encode": "VK_QUEUE_VIDEO_ENCODE_BIT_KHR",
    }

    queues_enum = new_types.enums["VkQueueFlagBits"]
    return [queues_enum.fields[mapping[queue]] for queue in queues]


# def process_vulkan_command_parameters()


def process(
        internal_commands: internal_types.AllVulkanCommands,
        vulkan_types: types.VulkanTypeInfo) -> types.VulkanCommandInfo:
    """
    Post process Vulkan Commands
    All The linking will be done here.
    """
    new_commands = types.VulkanCommandInfo()

    for command in internal_commands.commands.values():

        parameters: Dict[str, types.VulkanCommandParam] = {}
        for param in command.parameters.values():

            typ = query.get_type_or_deducted_type(parser_utils.get_plain_typename(param.parameter_type), vulkan_types)
            size: Optional[Union[str, types.VulkanCommandParam]] = None
            if param.array_size_reference:
                if "->" in param.array_size_reference:
                    size = param.array_size_reference
                else:
                    size = parameters[param.array_size_reference]

            parameters[param.parameter_name] = types.VulkanCommandParam(
                type=typ,
                name=param.parameter_name,
                full_typename=param.parameter_type,
                optional=param.optional,
                externally_synced=param.externally_synced,
                externally_synced_field=param.externally_synced_field,
                size=size,
            )

        return_type = vulkan_types.all_types[command.return_type]
        allowance = convert_render_pass_allowance(command.renderpass_allowance)
        success_codes = convert_vk_result_codes(command.success_codes, vulkan_types)
        error_codes = convert_vk_result_codes(command.error_codes, vulkan_types)
        queues = convert_queues(command.queues, vulkan_types)
        levels = convert_cmd_buffer_level(command.command_buffer_levels, vulkan_types)

        new_commands.commands[command.name] = types.VulkanCommand(
            name=command.name,
            return_type=return_type,
            renderpass_allowance=allowance,
            success_codes=success_codes,
            error_codes=error_codes,
            queues=queues,
            command_buffer_levels=levels,
            parameters=parameters)

    for alias in internal_commands.command_aliases.values():
        new_commands.command_aliases[alias.command_name] = types.VulkanCommandAlias(
            name=alias.command_name,
            aliased_command=new_commands.commands[alias.aliased_command_name]
        )

    return new_commands
