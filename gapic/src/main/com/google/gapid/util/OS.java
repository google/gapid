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

import java.io.File;
import java.io.IOException;

public class OS {
  public static final String name;
  public static final String arch;
  public static final boolean isWindows, isMac, isLinux;
  public static final String userHomeDir;
  public static final String exeExtension;
  public static final String cwd;

  static {
    name = System.getProperty("os.name", "").toLowerCase();
    arch = System.getProperty("os.arch", "").toLowerCase();
    isWindows = name.indexOf("win") >= 0;
    isMac = name.indexOf("mac") >= 0;
    isLinux = name.indexOf("nux") >= 0;
    userHomeDir = System.getProperty("user.home", ".");
    exeExtension = isWindows ? ".exe" : "";
    cwd = java.nio.file.Paths.get(".").toAbsolutePath().toString();
  }

  public static void openFileInSystemExplorer(File file) throws IOException {
    String cmd = getSystemExplorerCommand(file.toURI().toString(), file.isDirectory());
    if (isLinux || isMac) {
      Runtime.getRuntime().exec(new String[] { "/bin/sh", "-c", cmd }, null, null);
    } else {
      Runtime.getRuntime().exec(cmd, null, null);
    }
  }

  private static String getSystemExplorerCommand(String file, boolean isDir) {
    if (isLinux) {
      return "dbus-send --print-reply --dest=org.freedesktop.FileManager1 "
          + "/org/freedesktop/FileManager1 org.freedesktop.FileManager1.ShowItems "
          + "array:string:\"" + file + "\" string:\"\"";
    } else if (isWindows) {
      return "explorer /E," + (isDir ? "" : "/select,") + file;
    } else if (isMac) {
      return "open -R \"" + file + "\"";
    } else {
      return file;
    }
  }
}
