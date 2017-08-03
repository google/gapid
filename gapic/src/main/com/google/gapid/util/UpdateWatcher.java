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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.service.Service.Release;
import com.google.gapid.server.Client;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;

/**
 * Utility class for checking for new releases of GAPID.
 */
public class UpdateWatcher {
  private static final long CHECK_INTERVAL_MS = TimeUnit.HOURS.toMillis(8);
  private static final boolean INCLUDE_PRE_RELEASES = true;

  private final Settings settings;
  private final Client client;
  private final Listener listener;

  /** Callback interface */
  public interface Listener {
    /** Called whenever a new release is found. */
    void onNewReleaseAvailable(Release release);
  }

  public UpdateWatcher(Settings settings, Client client, Listener listener) {
    this.settings = settings;
    this.client = client;
    this.listener = listener;
    if (settings.updateAvailable) {
      Scheduler.EXECUTOR.schedule(this::doCheck, 0, TimeUnit.MILLISECONDS);
    } else {
      scheduleCheck();
    }
  }

  private void scheduleCheck() {
    long now = System.currentTimeMillis();
    long timeSinceLastUpdateMS = now - settings.lastCheckForUpdates;
    long delay = Math.max(CHECK_INTERVAL_MS - timeSinceLastUpdateMS, 0);
    Scheduler.EXECUTOR.schedule(this::doCheck, delay, TimeUnit.MILLISECONDS);
  }

  private void doCheck() {
    if (settings.autoCheckForUpdates) {
      ListenableFuture<Release> future = client.checkForUpdates(INCLUDE_PRE_RELEASES);
      settings.updateAvailable = false;
      try {
        Release release = future.get();
        if (release != null) {
          settings.updateAvailable = true;
          listener.onNewReleaseAvailable(release);
        }
      } catch (InterruptedException | ExecutionException e) {
        /* never mind */
      }
    }
    settings.lastCheckForUpdates = System.currentTimeMillis();
    settings.save();
    scheduleCheck();
  }
}
