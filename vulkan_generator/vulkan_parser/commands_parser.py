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

""" This module is responsible for parsing Vulkan commands and aliases of them"""

from typing import Dict
from typing import List
from typing import NamedTuple

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_utils import parsing_utils
from vulkan_generator.vulkan_parser import types


class ParameterInformation(NamedTuple):
    """Temporary class to return argument information"""
    parameter_order: List[str]
    parameters: Dict[str, types.VulkanFunctionArgument]


def parse_arguments(command_elem: ET.Element) -> ParameterInformation:
    """Parses the arguments of Vulkan Commands"""

    parameter_order: List[str] = []
    parameters: Dict[str, types.VulkanCommandParam] = {}

    for param_elem in command_elem:
        if param_elem.tag == "proto":
            # This tag is for function name and return type
            continue

        if param_elem.tag == "implicitexternsyncparams":
            # We do not need to worry about this tag
            continue

        if param_elem.tag != "param":
            raise SyntaxError(f"Unknown tag in function parameters: {param_elem}")

        parameter_type = parsing_utils.get_text_from_tag_in_children(param_elem, "type")
        parameter_name = parsing_utils.get_text_from_tag_in_children(param_elem, "name")

        # This part(attributes and pointers) is very similar to parsing typenames in struct
        # But as we cannot guarantee that it will stay similar
        # It's better to write separately to be able to handle nuances in between.

        # If type is const, it's written under the param tag's text instead of type.
        # e.g. <param>const <type>VkInstanceCreateInfo</type>* <name>pCreateInfo</name></param>
        type_attributes = param_elem.text
        # some times it's just empty space or endline character
        if type_attributes:
            type_attributes = parsing_utils.clean_type_string(type_attributes)
            # it might be empty after cleaning
            if type_attributes:
                parameter_type = f"{type_attributes} {parameter_type}"

        pointers = parsing_utils.try_get_tail_from_tag_in_children(param_elem, "type")
        # some times it's just empty space or endline character
        if pointers:
            pointers = parsing_utils.clean_type_string(pointers)
            # it might be empty after cleaning
            if pointers:
                # Add space between "*" and "const"
                pointers = pointers.replace("const", " const")
                parameter_type = f"{parameter_type}{pointers}"

        if not parameter_type:
            raise SyntaxError(
                f"No parameter type found in : {ET.tostring(param_elem, 'utf-8')}")

        if not parameter_name:
            raise SyntaxError(
                f"No parameter name found in : {ET.tostring(param_elem, 'utf-8')}")

        # Is this parameter optional or has to be not null
        # When this field is "false, true"  it's always for the length of the array
        # Therefore it does not give any extra information.
        optional = parsing_utils.try_get_attribute(param_elem, "optional") == "true"

        # Is this parameter must be externally synced
        externally_synced = parsing_utils.try_get_attribute(param_elem, "externsync") == "true"

        # This is useful when the parameter is a pointer to an array
        # with a length given by another parameter
        array_size_reference = parsing_utils.try_get_attribute(param_elem, "len")
        if array_size_reference:
            # pointer to char array has this property, which is redundant
            array_size_reference = array_size_reference.replace("null-terminated", "")
            array_size_reference = parsing_utils.clean_type_string(array_size_reference)

        parameter_order.append(parameter_name)
        parameters[parameter_name] = types.VulkanCommandParam(
            parameter_name=parameter_name,
            parameter_type=parameter_type,
            optional=optional,
            externally_synced=externally_synced,
            array_size_reference=array_size_reference,
        )

    return ParameterInformation(parameter_order=parameter_order, parameters=parameters)


def parse_command(command_elem: ET.Element) -> types.VulkanCommand:
    """Returns a Vulkan command from the XML element that defines it.

    A sample Vulkan command:
    <command successcodes="VK_SUCCESS,VK_NOT_READY"
        errorcodes="VK_ERROR_OUT_OF_HOST_MEMORY,VK_ERROR_OUT_OF_DEVICE_MEMORY,VK_ERROR_DEVICE_LOST">

        <proto><type>VkResult</type> <name>vkGetQueryPoolResults</name></proto>
        <param><type>VkDevice</type> <name>device</name></param>
        <param><type>VkQueryPool</type> <name>queryPool</name></param>
        <param><type>uint32_t</type> <name>firstQuery</name></param>
        <param><type>uint32_t</type> <name>queryCount</name></param>
        <param><type>size_t</type> <name>dataSize</name></param>
        <param len="dataSize"><type>void</type>* <name>pData</name></param>
        <param><type>VkDeviceSize</type> <name>stride</name></param>
        <param optional="true"><type>VkQueryResultFlags</type> <name>flags</name></param>
    </command>
    """

    success_codes = parsing_utils.try_get_attribute_as_list(command_elem, "successcodes")
    error_codes = parsing_utils.try_get_attribute_as_list(command_elem, "errorcodes")
    queues = parsing_utils.try_get_attribute_as_list(command_elem, "queues")
    command_buffer_levels = parsing_utils.try_get_attribute_as_list(command_elem, "cmdbufferlevel")

    renderpass_allowance = parsing_utils.try_get_attribute(command_elem, "renderpass")

    name = parsing_utils.get_text_from_tag_in_children(command_elem[0], "name")
    return_type = parsing_utils.get_text_from_tag_in_children(command_elem[0], "type")

    parameter_info = parse_arguments(command_elem)
    return types.VulkanCommand(
        name=name,
        return_type=return_type,
        renderpass_allowance=renderpass_allowance,
        success_codes=success_codes,
        queues=queues,
        command_buffer_levels=command_buffer_levels,
        error_codes=error_codes,
        parameter_order=parameter_info.parameter_order,
        parameters=parameter_info.parameters)


def parse_command_alias(command_elem: ET.Element) -> types.VulkanCommandAlias:
    """Returns a Vulkan command alias from the XML element that defines it.

    A sample Vulkan command alias:
    <command name="vkResetQueryPoolEXT" alias="vkResetQueryPool"/>
    """
    alias = command_elem.attrib["alias"]
    name = command_elem.attrib["name"]
    return types.VulkanCommandAlias(command_name=name, aliased_command_name=alias)


def parse(commands_elem: ET.Element) -> types.AllVulkanCommands:
    """Parses all the Vulkan commands and aliases"""
    vulkan_commands = types.AllVulkanCommands()

    for command_elem in commands_elem:
        if command_elem.tag != "command":
            raise SyntaxError("Unknown tag in commands: {command_elem}")

        if "alias" in command_elem.attrib:
            command_alias = parse_command_alias(command_elem)
            vulkan_commands.command_aliases[command_alias.command_name] = command_alias
            continue

        command = parse_command(command_elem)
        vulkan_commands.commands[command.name] = command

    return vulkan_commands
