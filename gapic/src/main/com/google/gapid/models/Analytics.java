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

import static java.util.logging.Level.FINE;
import static java.util.logging.Level.WARNING;

import com.google.gapid.proto.service.Service;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.ExceptionHandler;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class Analytics implements ExceptionHandler {
  private static final Logger LOG = Logger.getLogger(Analytics.class.getName());

  public static enum View {
    Main, LeftTabs, RightTabs,
    About, Help, GotoCommand, GotoMemory, Licenses, Settings, Trace, Welcome,
    // See MainWindow.MainTab.Type
    FilmStrip, Profile, Commands, Framebuffer, Pipeline, Textures, TextureView, Geometry, Shaders, Performance, Report, Log, State, Memory,
    ContextSelector, ReplayDeviceSelector, QueryMetadata;
  }

  private final Client client;
  private final Settings settings;
  private final ExceptionHandler handler;

  public Analytics(Client client, Settings settings, ExceptionHandler handler) {
    this.client = client;
    this.settings = settings;
    this.handler = handler;
  }

  @Override
  public void reportException(Throwable thrown) {
    if (settings.preferences().getReportCrashes()) {
      handler.reportException(thrown);
    }
  }

  public void postInteraction(View view, Service.ClientAction action) {
    LOG.log(FINE, "Interaction {1} on {0}", new Object[] { view, action });
    if (settings.analyticsEnabled()) {
      Rpc.listen(client.postEvent(interaction(view, action)), Analytics::logIfFailure);
    }
  }

  // TODO: Maybe find a better place for this...
  public void updateSettings() {
    Rpc.listen(settings.updateOnServer(client), Analytics::logIfFailure);
  }

  private static void logIfFailure(Rpc.Result<?> result) {
    try {
      result.get();
    } catch (RpcException | ExecutionException e) {
      LOG.log(WARNING, "Failed to toggle analytics/crash reporting", e);
    }
  }

  private static Service.ClientInteraction interaction(View view, Service.ClientAction action) {
    return Service.ClientInteraction.newBuilder()
        .setView(view.name())
        .setAction(action)
        .build();
  }
}
