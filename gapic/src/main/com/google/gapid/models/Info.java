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
package com.google.gapid.models;

import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Version;

/**
 * Basic information retrieved from and about the server.
 */
public class Info {
  private static Service.ServerInfo serverInfo = Service.ServerInfo.getDefaultInstance();

  private Info() {
  }

  public static void setServerInfo(Service.ServerInfo serverInfo) {
    Info.serverInfo = serverInfo;
  }

  public static Service.ServerInfo getServerInfo() {
    return serverInfo;
  }

  public static String getServerName() {
    return (serverInfo.getName().isEmpty()) ? "<unknown>" : serverInfo.getName();
  }

  public static Version getServerVersion() {
    return Version.fromProto(serverInfo);
  }
}
