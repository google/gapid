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
    Main, FilmStrip, LeftTabs, RightTabs,
    About, GotoCommand, GotoMemory, Licenses, Settings, Trace, Welcome,
    // See MainWindow.MainTab.Type
    Commands, Framebuffer, Textures, Geometry, Shaders, Report, Log, State, Memory,
    ContextSelector, ReplayDeviceSelector;
  }

  private final Client client;
  private final ExceptionHandler handler;
  private boolean enabled;
  private boolean reportCrashes;

  public Analytics(Client client, Settings settings, ExceptionHandler handler) {
    this.client = client;
    this.handler = handler;
    this.reportCrashes = settings.reportCrashes;
    this.enabled = settings.analyticsEnabled();

    settings.addListener(s -> update(s));
  }

  @Override
  public void reportException(Throwable thrown) {
    if (reportCrashes) {
      handler.reportException(thrown);
    }
  }

  public void postInteraction(View view, Service.ClientAction action) {
    LOG.log(FINE, "Interaction {1} on {0}", new Object[] { view, action });
    if (enabled) {
      Rpc.listen(client.postEvent(interaction(view, action)), Analytics::logIfFailure);
    }
  }

  private void update(Settings s) {
    if (reportCrashes != s.reportCrashes) {
      reportCrashes = s.reportCrashes;
      Rpc.listen(client.setCrashReportsEnabled(reportCrashes), Analytics::logIfFailure);
    }
    if (enabled != s.analyticsEnabled()) {
      enabled = s.analyticsEnabled();
      Rpc.listen(client.setAnalyticsEnabled(enabled, s.analyticsClientId), Analytics::logIfFailure);
    }
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
