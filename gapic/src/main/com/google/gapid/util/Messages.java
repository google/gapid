/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.gapid.util;

import static com.google.gapid.util.GapidVersion.GAPID_VERSION;

public interface Messages {
  public static final String WINDOW_TITLE = "Graphics API Debugger";
  public static final String LOADING_CAPTURE = "Loading capture...";
  public static final String CAPTURE_LOAD_FAILURE = "Failed to load capture.";
  public static final String NO_FRAMES_IN_CONTEXT = "No frames in selected context.";
  public static final String SELECT_COMMAND = "Select a frame or command.";
  public static final String SELECT_DRAW_CALL = "Select a draw call.";
  public static final String SELECT_MEMORY =
      "Select a command and observation or a pointer in the command list.";
  public static final String SELECT_TEXTURE = "Select a texture.";
  public static final String SELECT_OBSERVATION = "Select an observed memory range.";
  public static final String SELECT_STRUCT_OBSERVATION = "Select an observed struct memory.";
  public static final String SELECT_SHADER = "Select a shader.";
  public static final String SELECT_PROGRAM = "Select a program.";
  public static final String NO_IMAGE_DATA = "No image data available at this point in the trace.";
  public static final String NO_TEXTURES = "No textures have been created by this point.";
  public static final String VIEW_DETAILS = "View Details";
  public static final String LICENSES = "Licenses";
  public static final String ABOUT_TITLE = "About " + WINDOW_TITLE;
  public static final String ABOUT_COPY = "Copyright © 2017 Google Inc.";
  public static final String GOTO = "Goto...";
  public static final String GOTO_COMMAND = "Goto API Call";
  public static final String GOTO_MEMORY = "Goto Memory Location";
  public static final String COMMAND_ID = "API Call Number";
  public static final String MEMORY_ADDRESS = "Memory Address";
  public static final String MEMORY_POOL = "Memory Pool";
  public static final String MEMORY_BLOCK_TAB_TEXT = "Block";
  public static final String MEMORY_STRUCT_TAB_TEXT = "Struct";
  public static final String CAPTURE_TRACE_GRAPHICS = "Capture Graphics Trace";
  public static final String CAPTURE_TRACE_PERFETTO = "Capture System Profile";
  public static final String CAPTURING_TRACE = "Capturing Graphics Trace...";
  public static final String CAPTURE_DIRECTORY = "Capture output directory...";
  public static final String CAPTURE_EXECUTABLE = "Executable to trace...";
  public static final String CAPTURE_CWD = "Application working directory...";
  public static final String BROWSE = "Browse";
  public static final String SELECT_ACTIVITY = "Select an Application to Trace";
  public static final String WELCOME_TITLE = WINDOW_TITLE + " - Welcome";
  public static final String WELCOME_SUBTITLE = "Get started with " + WINDOW_TITLE;
  public static final String WELCOME_TEXT = "GAPID allows you to inspect, tweak, and replay calls" +
      " from an application to a\ngraphics API. To begin, let us know where adb is located on" +
      " your computer.";
  public static final String WELCOME_BUTTON = "Get Started";
  public static final String ANALYTICS_OPTION =
      "Help improve GAPID by sending usage statistics to Google";
  public static final String CRASH_REPORTING_OPTION =
      "Help GAPID identify issues by sending crash reports to Google";
  public static final String UPDATE_CHECK_OPTION = "Automatically check for GAPID updates (please restart GAPID to force an update check)";
  public static final String UPDATE_CHECK_PRERELEASE_OPTION = "Include developer preview releases";
  public static final String PRIVACY_POLICY =
      "Google's <a href=\"TOS\">APIs Terms of Service</a> and <a href=\"PP\">Privacy Policy</a>" +
      " govern your use of this application.";
  public static final String NO_REPLAY_DEVICE = "No replay device available for this capture.";
  public static final String SETTINGS_TITLE = "Modify Settings";
  public static final String ERROR_MESSAGE =
      "The application encountered an error:\n%s\n\nPlease check the logs for details.";
  public static final String BUG_BODY =
      "GAPID Version: " + GAPID_VERSION.toString() + "\n" +
      "OS: " + OS.name + " " + OS.arch + "\n\n" +
      "Please provide detailed steps that led to the error and copy-paste the stack trace.\n" +
      "Extra details from the logs and the trace file would be extra helpful.\n\n";
  public static final String NO_OPENGL =
      "Failed to create an OpenGL context. OpenGL is required to use this application.";
  public static final String GEO_SEMANTICS_TITLE = "Vertex Semantics";
  public static final String GEO_SEMANTICS_HINT = "Manually configure the vertex stream semantics:";
}
