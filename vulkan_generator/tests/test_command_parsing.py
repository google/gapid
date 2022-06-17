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
This module is responsible for testing Vulkan commands and aliases

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import commands_parser


def test_vulkan_command() -> None:
    """"Tests Vulkan command"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command>
                <proto><type>void</type> <name>vkQueueEndDebugUtilsLabelEXT</name></proto>
                <param><type>VkQueue</type> <name>queue</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))

    assert len(typ.commands) == 1
    assert "vkQueueEndDebugUtilsLabelEXT" in typ.commands

    command = typ.commands["vkQueueEndDebugUtilsLabelEXT"]
    assert isinstance(command, types.VulkanCommand)

    assert command.return_type == "void"

    assert len(command.parameters) == 1
    assert command.parameter_order[0] == "queue"

    assert command.parameters["queue"].parameter_type == "VkQueue"
    assert command.parameters["queue"].parameter_name == "queue"


def test_vulkan_command_with_success_and_error_code() -> None:
    """"Tests Vulkan command with successcodes and errorcodes attribute"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command successcodes="VK_SUCCESS,VK_INCOMPLETE"
                errorcodes="VK_ERROR_OUT_OF_HOST_MEMORY,VK_ERROR_OUT_OF_DEVICE_MEMORY">
                <proto><type>VkResult</type> <name>vkEnumerateInstanceLayerProperties</name></proto>
                <param optional="false,true"><type>uint32_t</type>* <name>pPropertyCount</name></param>
                <param optional="true" len="pPropertyCount"><type>VkLayerProperties</type>
                    * <name>pProperties</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkEnumerateInstanceLayerProperties"]

    assert command.return_type == "VkResult"

    assert len(command.error_codes) == 2
    assert "VK_ERROR_OUT_OF_HOST_MEMORY" in command.error_codes
    assert "VK_ERROR_OUT_OF_DEVICE_MEMORY" in command.error_codes

    assert len(command.success_codes) == 2
    assert "VK_SUCCESS" in command.success_codes
    assert "VK_INCOMPLETE" in command.success_codes


def test_vulkan_command_with_renderpass_allowance() -> None:
    """"Tests Vulkan command with renderpass attribute"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command queues="graphics" renderpass="both" cmdbufferlevel="primary,secondary">
                <proto><type>void</type> <name>vkCmdSetLineWidth</name></proto>
                <param externsync="true"><type>VkCommandBuffer</type> <name>commandBuffer</name></param>
                <param><type>float</type> <name>lineWidth</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkCmdSetLineWidth"]
    assert command.renderpass_allowance == "both"


def test_vulkan_command_with_command_buffer_levels() -> None:
    """"Tests Vulkan command with cmdbufferlevel attribute"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command queues="graphics" renderpass="both" cmdbufferlevel="primary,secondary">
                <proto><type>void</type> <name>vkCmdSetLineWidth</name></proto>
                <param externsync="true"><type>VkCommandBuffer</type> <name>commandBuffer</name></param>
                <param><type>float</type> <name>lineWidth</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkCmdSetLineWidth"]

    assert len(command.command_buffer_levels) == 2
    assert "primary" in command.command_buffer_levels
    assert "secondary" in command.command_buffer_levels


def test_vulkan_command_with_queues() -> None:
    """"Tests Vulkan command with queues attribute"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command queues="transfer,graphics,compute" renderpass="both" cmdbufferlevel="primary">
                <proto><type>void</type> <name>vkCmdExecuteCommands</name></proto>
                <param externsync="true"><type>VkCommandBuffer</type> <name>commandBuffer</name></param>
                <param><type>uint32_t</type> <name>commandBufferCount</name></param>
                <param len="commandBufferCount">const
                    <type>VkCommandBuffer</type>* <name>pCommandBuffers</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkCmdExecuteCommands"]
    assert len(command.queues) == 3
    assert "transfer" in command.queues
    assert "graphics" in command.queues
    assert "compute" in command.queues


def test_vulkan_command_with_externally_synced_parameter() -> None:
    """"Tests Vulkan command that has a parameter that must be externally synced"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command queues="graphics" renderpass="inside" cmdbufferlevel="primary">
                <proto><type>void</type> <name>vkCmdEndRenderPass</name></proto>
                <param externsync="true"><type>VkCommandBuffer</type> <name>commandBuffer</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkCmdEndRenderPass"]
    assert command.parameters["commandBuffer"].externally_synced


def test_vulkan_command_with_an_optional_parameter() -> None:
    """"Tests Vulkan command that has a parameter that is optional"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command successcodes="VK_SUCCESS,VK_INCOMPLETE"
                errorcodes="VK_ERROR_OUT_OF_HOST_MEMORY,VK_ERROR_OUT_OF_DEVICE_MEMORY">
                <proto><type>VkResult</type> <name>vkGetPhysicalDeviceDisplayPropertiesKHR</name></proto>
                <param><type>VkPhysicalDevice</type> <name>physicalDevice</name></param>
                <param optional="false,true"><type>uint32_t</type>* <name>pPropertyCount</name></param>
                <param optional="true" len="pPropertyCount"><type>VkDisplayPropertiesKHR</type>*
                    <name>pProperties</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkGetPhysicalDeviceDisplayPropertiesKHR"]

    assert command.parameters["pProperties"].optional


def test_vulkan_command_with_an_array() -> None:
    """"Tests Vulkan command that has a parameter that is an array"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command successcodes="VK_SUCCESS,VK_INCOMPLETE"
                errorcodes="VK_ERROR_OUT_OF_HOST_MEMORY,VK_ERROR_OUT_OF_DEVICE_MEMORY">
                <proto><type>VkResult</type> <name>vkGetPhysicalDeviceDisplayPropertiesKHR</name></proto>
                <param><type>VkPhysicalDevice</type> <name>physicalDevice</name></param>
                <param optional="false,true"><type>uint32_t</type>* <name>pPropertyCount</name></param>
                <param optional="true" len="pPropertyCount"><type>VkDisplayPropertiesKHR</type>*
                    <name>pProperties</name></param>
            </command>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))
    command = typ.commands["vkGetPhysicalDeviceDisplayPropertiesKHR"]

    assert command.parameters["pProperties"].array_size_reference == "pPropertyCount"
    assert "pPropertyCount" in command.parameters


def test_vulkan_command_alias() -> None:
    """"Tests Vulkan command alias"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <commands>
            <command name="vkCmdEndRenderingKHR" alias="vkCmdEndRendering"/>
        </commands>
    """

    typ = commands_parser.parse(ET.fromstring(xml))

    assert len(typ.command_aliases) == 1
    assert "vkCmdEndRenderingKHR" in typ.command_aliases

    alias = typ.command_aliases["vkCmdEndRenderingKHR"]

    assert isinstance(alias, types.VulkanCommandAlias)
    assert alias.command_name == "vkCmdEndRenderingKHR"
    assert alias.aliased_command_name == "vkCmdEndRendering"
